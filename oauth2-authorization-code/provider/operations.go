package main

import (
	"net/http"
	"strings"
	"time"
)

func (a *app) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		a.writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "missing bearer token"})
		return
	}

	claims, ok := a.validateActiveAccessToken(token)
	if !ok || !strings.Contains(readStringClaim(claims, "scope"), "openid") {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		a.writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "token validation failed"})
		return
	}

	a.writeJSON(w, http.StatusOK, map[string]any{
		"sub":                readStringClaim(claims, "sub"),
		"name":               readStringClaim(claims, "name"),
		"email":              readStringClaim(claims, "email"),
		"preferred_username": readStringClaim(claims, "preferred_username"),
		"roles":              readStringSliceClaim(claims, "roles"),
	})
}

func (a *app) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	_ = r.ParseForm()
	postLogoutRedirectURI := r.Form.Get("post_logout_redirect_uri")
	if postLogoutRedirectURI != "" && !a.isAllowedPostLogoutRedirectURI(postLogoutRedirectURI) {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "post_logout_redirect_uri is not allowed")
		return
	}

	if cookie, err := r.Cookie(a.cfg.ProviderSessionCookieName); err == nil {
		a.mu.Lock()
		delete(a.providerSessions, cookie.Value)
		a.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: a.cfg.ProviderSessionCookieName, Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})

	if postLogoutRedirectURI != "" {
		http.Redirect(w, r, postLogoutRedirectURI, http.StatusFound)
		return
	}
	a.writeJSON(w, http.StatusOK, map[string]bool{"logged_out": true})
}

func (a *app) handleRevocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "expected application/x-www-form-urlencoded body")
		return
	}
	client, err := a.authenticateClient(r)
	if err != nil {
		a.writeOAuthError(w, http.StatusUnauthorized, "invalid_client", err.Error())
		return
	}
	token := r.Form.Get("token")
	if token == "" {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "token is required")
		return
	}

	if strings.Count(token, ".") == 2 {
		claims, err := verifyJWT(token, a.signer.publicKey)
		if err == nil && readStringClaim(claims, "client_id") == client.ID {
			if jti := readStringClaim(claims, "jti"); jti != "" {
				exp, _ := readNumericClaim(claims, "exp")
				tokenExp := time.Unix(exp, 0)
				a.mu.Lock()
				a.revokedTokenIDs[jti] = tokenExp
				a.mu.Unlock()
			}
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	a.mu.Lock()
	grant, ok := a.refreshTokens[token]
	if ok && grant.ClientID == client.ID {
		delete(a.refreshTokens, token)
	}
	a.mu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (a *app) handleIntrospection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "expected application/x-www-form-urlencoded body")
		return
	}
	if _, err := a.authenticateClient(r); err != nil {
		a.writeOAuthError(w, http.StatusUnauthorized, "invalid_client", err.Error())
		return
	}
	token := r.Form.Get("token")
	if token == "" {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "token is required")
		return
	}

	if strings.Count(token, ".") == 2 {
		claims, ok := a.validateActiveAccessToken(token)
		if !ok {
			a.writeJSON(w, http.StatusOK, map[string]any{"active": false})
			return
		}
		a.writeJSON(w, http.StatusOK, map[string]any{
			"active":    true,
			"token_use": readStringClaim(claims, "token_use"),
			"scope":     readStringClaim(claims, "scope"),
			"client_id": readStringClaim(claims, "client_id"),
			"username":  readStringClaim(claims, "preferred_username"),
			"sub":       readStringClaim(claims, "sub"),
			"aud":       readStringClaim(claims, "aud"),
			"exp":       mustNumericClaim(claims, "exp"),
			"iat":       mustNumericClaim(claims, "iat"),
			"iss":       readStringClaim(claims, "iss"),
			"roles":     readStringSliceClaim(claims, "roles"),
		})
		return
	}

	a.mu.Lock()
	grant, ok := a.refreshTokens[token]
	a.mu.Unlock()
	if !ok || time.Now().After(grant.ExpiresAt) {
		a.writeJSON(w, http.StatusOK, map[string]any{"active": false})
		return
	}
	a.writeJSON(w, http.StatusOK, map[string]any{
		"active":    true,
		"token_use": "refresh_token",
		"client_id": grant.ClientID,
		"username":  grant.User.Username,
		"sub":       grant.User.Subject,
		"scope":     strings.Join(grant.Scopes, " "),
		"exp":       grant.ExpiresAt.Unix(),
	})
}

func (a *app) validateActiveAccessToken(token string) (map[string]any, bool) {
	claims, err := verifyJWT(token, a.signer.publicKey)
	if err != nil {
		return nil, false
	}
	if !validTimeClaims(claims) || readStringClaim(claims, "iss") != a.cfg.Issuer || readStringClaim(claims, "token_use") != "access_token" {
		return nil, false
	}
	if a.isTokenIDRevoked(readStringClaim(claims, "jti")) {
		return nil, false
	}
	return claims, true
}

func (a *app) isTokenIDRevoked(jti string) bool {
	if jti == "" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	exp, ok := a.revokedTokenIDs[jti]
	if !ok {
		return false
	}
	// Prune the entry once the token has expired naturally; it cannot be used anyway.
	if time.Now().After(exp) {
		delete(a.revokedTokenIDs, jti)
		return false
	}
	return true
}

func (a *app) isAllowedPostLogoutRedirectURI(value string) bool {
	allowed := map[string]bool{
		a.cfg.PKCEOrigin:       true,
		a.cfg.PKCEOrigin + "/": true,
		a.cfg.BFFOrigin:        true,
		a.cfg.BFFOrigin + "/":  true,
	}
	return allowed[value]
}
