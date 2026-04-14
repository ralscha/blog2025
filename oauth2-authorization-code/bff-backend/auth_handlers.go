package main

import (
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

func (b *bffApp) cleanPendingLogins() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		b.mu.Lock()
		for state, p := range b.pending {
			if now.After(p.ExpiresAt) {
				delete(b.pending, state)
			}
		}
		b.mu.Unlock()
	}
}

func (b *bffApp) handleLogin(w http.ResponseWriter, r *http.Request) {
	user := strings.ToLower(r.URL.Query().Get("user"))
	if user != "alice" && user != "bob" {
		b.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user must be alice or bob"})
		return
	}

	state, err := randomToken(24)
	if err != nil {
		b.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	nonce, err := randomToken(24)
	if err != nil {
		b.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	codeVerifier, err := randomToken(32)
	if err != nil {
		b.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	b.mu.Lock()
	b.pending[state] = pendingLogin{User: user, Nonce: nonce, CodeVerifier: codeVerifier, ExpiresAt: time.Now().Add(5 * time.Minute)}
	b.mu.Unlock()

	authorizeURL := b.oauthConfig.AuthCodeURL(
		state,
		oauth2.S256ChallengeOption(codeVerifier),
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.SetAuthURLParam("login_hint", user),
	)
	http.Redirect(w, r, authorizeURL, http.StatusFound)
}

func (b *bffApp) handleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	b.mu.Lock()
	pending, ok := b.pending[state]
	if ok {
		delete(b.pending, state)
	}
	b.mu.Unlock()

	if !ok || time.Now().After(pending.ExpiresAt) {
		b.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "state not found or expired"})
		return
	}

	tokens, err := b.exchangeCode(code, pending.CodeVerifier)
	if err != nil {
		b.writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	idClaims, err := b.validateJWTClaims(tokens.IDToken, b.clientID)
	if err != nil {
		b.writeJSON(w, http.StatusBadGateway, map[string]string{"error": "id_token validation failed: " + err.Error()})
		return
	}
	if nonce := readStringClaim(idClaims, "nonce"); nonce != pending.Nonce {
		b.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "nonce mismatch"})
		return
	}

	accessClaims, err := b.validateJWTClaims(tokens.AccessToken, b.resourceAudience)
	if err != nil {
		b.writeJSON(w, http.StatusBadGateway, map[string]string{"error": "access_token validation failed: " + err.Error()})
		return
	}

	sessionID, err := randomToken(32)
	if err != nil {
		b.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// The browser gets only an opaque session cookie; tokens stay on the server.
	savedSession := &browserSession{
		ID:              sessionID,
		User:            userProfileFromClaims(idClaims),
		AccessToken:     tokens.AccessToken,
		AccessTokenExp:  readNumericClaim(accessClaims, "exp"),
		RefreshToken:    tokens.RefreshToken,
		RefreshTokenExp: time.Now().Add(45 * time.Minute),
		Scope:           tokens.Scope,
		CreatedAt:       time.Now(),
	}

	if err := b.sessionManager.RenewToken(r.Context()); err != nil {
		b.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to renew browser session"})
		return
	}
	b.sessionManager.Put(r.Context(), browserSessionKey, savedSession)

	http.Redirect(w, r, b.frontendOrigin, http.StatusFound)
}

func (b *bffApp) handleLogout(w http.ResponseWriter, r *http.Request) {
	storedSession, ok := b.sessionFromRequest(r)
	if !ok {
		b.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}
	if !b.validCSRFRequest(r) {
		b.writeJSON(w, http.StatusForbidden, map[string]string{"error": "csrf validation failed"})
		return
	}

	_ = b.revokeToken(storedSession.AccessToken, "access_token")
	if storedSession.RefreshToken != "" {
		_ = b.revokeToken(storedSession.RefreshToken, "refresh_token")
	}
	if err := b.sessionManager.Destroy(r.Context()); err != nil {
		b.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to destroy browser session"})
		return
	}
	b.writeJSON(w, http.StatusOK, map[string]bool{"loggedOut": true})
}
