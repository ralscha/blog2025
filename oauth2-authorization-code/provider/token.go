package main

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"
)

func (a *app) handleToken(w http.ResponseWriter, r *http.Request) {
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

	switch r.Form.Get("grant_type") {
	case "authorization_code":
		a.exchangeAuthorizationCode(w, r, client)
	case "refresh_token":
		a.exchangeRefreshToken(w, r, client)
	default:
		a.writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "supported grants are authorization_code and refresh_token")
	}
}

func (a *app) exchangeAuthorizationCode(w http.ResponseWriter, r *http.Request, client Client) {
	codeValue := r.Form.Get("code")
	redirectURI := r.Form.Get("redirect_uri")
	codeVerifier := r.Form.Get("code_verifier")
	if codeValue == "" || redirectURI == "" {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "code and redirect_uri are required")
		return
	}

	a.mu.Lock()
	code, ok := a.authCodes[codeValue]
	if ok {
		delete(a.authCodes, codeValue)
	}
	a.mu.Unlock()

	if !ok {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "authorization code not found or already used")
		return
	}
	if time.Now().After(code.ExpiresAt) {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "authorization code expired")
		return
	}
	if code.ClientID != client.ID || code.RedirectURI != redirectURI {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "authorization code was not issued for this client")
		return
	}
	if code.CodeChallenge != "" && (codeVerifier == "" || computePKCEChallenge(codeVerifier) != code.CodeChallenge) {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "code_verifier does not match the stored PKCE challenge")
		return
	}

	response, err := a.issueTokens(client.ID, code.User, code.Scopes, code.Nonce)
	if err != nil {
		a.writeOAuthError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	a.writeJSON(w, http.StatusOK, response)
}

func (a *app) exchangeRefreshToken(w http.ResponseWriter, r *http.Request, client Client) {
	refreshValue := r.Form.Get("refresh_token")
	if refreshValue == "" {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "refresh_token is required")
		return
	}

	a.mu.Lock()
	grant, ok := a.refreshTokens[refreshValue]
	if ok {
		delete(a.refreshTokens, refreshValue)
	}
	a.mu.Unlock()

	if !ok {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "refresh token not found or already used")
		return
	}
	if time.Now().After(grant.ExpiresAt) {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "refresh token expired")
		return
	}
	if grant.ClientID != client.ID {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "refresh token does not belong to this client")
		return
	}

	requestedScopes := parseScopes(r.Form.Get("scope"))
	if len(requestedScopes) > 0 && !isSubset(requestedScopes, grant.Scopes) {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_scope", "refresh token cannot request scopes outside the original grant")
		return
	}
	if len(requestedScopes) == 0 {
		requestedScopes = grant.Scopes
	}

	response, err := a.issueTokens(client.ID, grant.User, requestedScopes, "")
	if err != nil {
		a.writeOAuthError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	a.writeJSON(w, http.StatusOK, response)
}

func (a *app) issueTokens(clientID string, user User, scopes []string, nonce string) (TokenResponse, error) {
	// Every token exchange returns a fresh access token and optionally rotates refresh access.
	now := time.Now().UTC()
	accessJTI, err := randomToken(24)
	if err != nil {
		return TokenResponse{}, err
	}
	idJTI, err := randomToken(24)
	if err != nil {
		return TokenResponse{}, err
	}

	accessClaims := map[string]any{
		"iss":                a.cfg.Issuer,
		"sub":                user.Subject,
		"aud":                a.cfg.ResourceAudience,
		"exp":                now.Add(a.cfg.AccessTokenTTL).Unix(),
		"iat":                now.Unix(),
		"nbf":                now.Unix(),
		"jti":                accessJTI,
		"scope":              strings.Join(scopes, " "),
		"client_id":          clientID,
		"preferred_username": user.Username,
		"name":               user.Name,
		"email":              user.Email,
		"roles":              user.Roles,
		"token_use":          "access_token",
	}

	idClaims := map[string]any{
		"iss":                a.cfg.Issuer,
		"sub":                user.Subject,
		"aud":                clientID,
		"exp":                now.Add(a.cfg.AccessTokenTTL).Unix(),
		"iat":                now.Unix(),
		"jti":                idJTI,
		"preferred_username": user.Username,
		"name":               user.Name,
		"email":              user.Email,
		"roles":              user.Roles,
		"token_use":          "id_token",
	}
	if nonce != "" {
		idClaims["nonce"] = nonce
	}

	accessToken, err := a.signer.Sign(accessClaims)
	if err != nil {
		return TokenResponse{}, err
	}
	idToken, err := a.signer.Sign(idClaims)
	if err != nil {
		return TokenResponse{}, err
	}

	response := TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(a.cfg.AccessTokenTTL.Seconds()),
		Scope:       strings.Join(scopes, " "),
		IDToken:     idToken,
	}

	if slices.Contains(scopes, "offline_access") {
		refreshToken, err := randomToken(48)
		if err != nil {
			return TokenResponse{}, err
		}
		a.mu.Lock()
		a.refreshTokens[refreshToken] = RefreshGrant{Value: refreshToken, ClientID: clientID, Scopes: scopes, User: user, ExpiresAt: now.Add(a.cfg.RefreshTokenTTL)}
		a.mu.Unlock()
		response.RefreshToken = refreshToken
	}

	return response, nil
}

func (a *app) authenticateClient(r *http.Request) (Client, error) {
	if clientID, clientSecret, ok := r.BasicAuth(); ok {
		client, exists := a.clients[clientID]
		if !exists {
			return Client{}, fmt.Errorf("unknown client_id")
		}
		if client.Public || client.Secret != clientSecret {
			return Client{}, fmt.Errorf("client authentication failed")
		}
		return client, nil
	}

	clientID := r.Form.Get("client_id")
	if clientID == "" {
		return Client{}, fmt.Errorf("client_id is required")
	}
	clientSecret := r.Form.Get("client_secret")
	client, exists := a.clients[clientID]
	if !exists {
		return Client{}, fmt.Errorf("unknown client_id")
	}

	if client.Public {
		if clientSecret != "" {
			return Client{}, fmt.Errorf("public clients must not send client_secret")
		}
		return client, nil
	}

	if client.Secret != clientSecret {
		return Client{}, fmt.Errorf("client authentication failed")
	}
	return client, nil
}
