package main

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

func (a *app) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	clientID := query.Get("client_id")
	if clientID == "" {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "client_id is required")
		return
	}

	client, ok := a.clients[clientID]
	if !ok {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_client", "unknown client_id")
		return
	}

	redirectURI := query.Get("redirect_uri")
	if redirectURI == "" {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "redirect_uri is required")
		return
	}
	if !slices.Contains(client.RedirectURIs, redirectURI) {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "redirect_uri is not registered")
		return
	}

	state := query.Get("state")
	if state == "" {
		a.writeAuthorizeError(w, redirectURI, "", "invalid_request", "state is required")
		return
	}
	if query.Get("response_type") != "code" {
		a.writeAuthorizeError(w, redirectURI, state, "unsupported_response_type", "only response_type=code is supported")
		return
	}

	scopes := parseScopes(query.Get("scope"))
	if len(scopes) == 0 || !slices.Contains(scopes, "openid") {
		a.writeAuthorizeError(w, redirectURI, state, "invalid_scope", "openid scope is required")
		return
	}
	if !allScopesAllowed(scopes, client.Scopes) {
		a.writeAuthorizeError(w, redirectURI, state, "invalid_scope", "requested scope is not allowed")
		return
	}

	nonce := query.Get("nonce")
	if nonce == "" {
		a.writeAuthorizeError(w, redirectURI, state, "invalid_request", "nonce is required")
		return
	}

	prompt := query.Get("prompt")
	if prompt != "" && prompt != "login" && prompt != "none" {
		a.writeAuthorizeError(w, redirectURI, state, "invalid_request", "prompt may only be empty, login, or none")
		return
	}

	codeChallenge := query.Get("code_challenge")
	codeChallengeMethod := query.Get("code_challenge_method")
	if client.RequirePKCE && (codeChallenge == "" || codeChallengeMethod != "S256") {
		a.writeAuthorizeError(w, redirectURI, state, "invalid_request", "public clients must send an S256 PKCE challenge")
		return
	}
	if !client.RequirePKCE && codeChallenge != "" && codeChallengeMethod != "S256" {
		a.writeAuthorizeError(w, redirectURI, state, "invalid_request", "code_challenge_method must be S256 when a challenge is sent")
		return
	}

	if shouldPromptForLogin(r, strings.ToLower(query.Get("login_hint")), prompt, a.cfg.ProviderSessionCookieName) {
		a.redirectToLogin(w, r)
		return
	}

	user, sessionID, authCode, authDescription := a.resolveAuthorizationUser(r, strings.ToLower(query.Get("login_hint")), prompt)
	if authCode != "" {
		a.writeAuthorizeError(w, redirectURI, state, authCode, authDescription)
		return
	}

	if sessionID != "" {
		a.setProviderSessionCookie(w, sessionID)
	}

	code, err := randomToken(32)
	if err != nil {
		a.writeAuthorizeError(w, redirectURI, state, "server_error", "failed to issue authorization code")
		return
	}

	a.mu.Lock()
	a.authCodes[code] = AuthorizationCode{
		Value:               code,
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		Scopes:              scopes,
		User:                user,
		Nonce:               nonce,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		ExpiresAt:           time.Now().Add(a.cfg.CodeTTL),
	}
	a.mu.Unlock()

	redirect, err := url.Parse(redirectURI)
	if err != nil {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "redirect_uri is malformed")
		return
	}
	values := redirect.Query()
	values.Set("code", code)
	values.Set("state", state)
	redirect.RawQuery = values.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

func (a *app) redirectToLogin(w http.ResponseWriter, r *http.Request) {
	redirect := url.URL{Path: "/login"}
	values := redirect.Query()
	values.Set("return_to", r.URL.RequestURI())
	redirect.RawQuery = values.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

func (a *app) resolveAuthorizationUser(r *http.Request, loginHint string, prompt string) (User, string, string, string) {
	// The provider accepts either an explicit login_hint or an existing provider session.
	if loginHint != "" {
		user, ok := a.users[loginHint]
		if !ok {
			return User{}, "", "access_denied", "login_hint must be alice or bob for this demo"
		}
		sessionID, err := a.createProviderSession(user)
		if err != nil {
			return User{}, "", "server_error", "failed to persist provider session"
		}
		return user, sessionID, "", ""
	}

	user, ok := a.userFromProviderSession(r)
	if ok {
		return user, "", "", ""
	}

	if prompt == "none" {
		return User{}, "", "login_required", "no provider session is active"
	}

	return User{}, "", "access_denied", "login_hint is required when no provider session exists"
}

func shouldPromptForLogin(r *http.Request, loginHint string, prompt string, cookieName string) bool {
	if loginHint != "" {
		return false
	}
	if prompt == "none" {
		return false
	}
	if prompt == "login" {
		return true
	}
	return !hasProviderSessionCookie(r, cookieName)
}

func hasProviderSessionCookie(r *http.Request, cookieName string) bool {
	_, err := r.Cookie(cookieName)
	return err == nil
}

func (a *app) createProviderSession(user User) (string, error) {
	// Provider sessions make prompt=none and prompt=login observable in the browser.
	sessionID, err := randomToken(32)
	if err != nil {
		return "", err
	}
	a.mu.Lock()
	a.providerSessions[sessionID] = ProviderSession{ID: sessionID, User: user, ExpiresAt: time.Now().Add(a.cfg.ProviderSessionTTL)}
	a.mu.Unlock()
	return sessionID, nil
}

func (a *app) userFromProviderSession(r *http.Request) (User, bool) {
	cookie, err := r.Cookie(a.cfg.ProviderSessionCookieName)
	if err != nil {
		return User{}, false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	session, ok := a.providerSessions[cookie.Value]
	if !ok || time.Now().After(session.ExpiresAt) {
		delete(a.providerSessions, cookie.Value)
		return User{}, false
	}
	return session.User, true
}

func (a *app) setProviderSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.cfg.ProviderSessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(a.cfg.ProviderSessionTTL.Seconds()),
	})
}
