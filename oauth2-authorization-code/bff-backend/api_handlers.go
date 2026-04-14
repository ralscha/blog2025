package main

import (
	"context"
	"errors"
	"net/http"
	"time"
)

func (b *bffApp) handleSession(w http.ResponseWriter, r *http.Request) {
	storedSession, ok := b.sessionFromRequest(r)
	if !ok {
		b.writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}

	if err := b.ensureFreshAccessToken(r.Context(), storedSession); err != nil {
		b.writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	b.writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"user": map[string]any{
			"sub":                storedSession.User.Sub,
			"preferred_username": storedSession.User.PreferredUsername,
			"name":               storedSession.User.Name,
			"email":              storedSession.User.Email,
			"roles":              storedSession.User.Roles,
		},
		"session": map[string]any{
			"browserHasTokens":      false,
			"tokenStorage":          "server-side session",
			"accessTokenExpiresAt":  storedSession.AccessTokenExp,
			"refreshTokenAvailable": storedSession.RefreshToken != "",
			"scope":                 storedSession.Scope,
		},
	})
}

func (b *bffApp) handleData(w http.ResponseWriter, r *http.Request) {
	storedSession, ok := b.sessionFromRequest(r)
	if !ok {
		b.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}

	if err := b.ensureFreshAccessToken(r.Context(), storedSession); err != nil {
		b.writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	requestTrace := traceRequest{
		Method:  http.MethodGet,
		URL:     b.resourceAPIURL,
		Headers: map[string]string{"Authorization": "Bearer [server-side token]"},
		Notes:   "BFF invokes the colocated resource API with the server-held access token.",
	}
	payload, err := b.resourceProfilePayload(storedSession.AccessToken)
	if err != nil {
		b.traceLogger.Write("resource_api", requestTrace, nil, err)
		b.writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	b.traceLogger.Write("resource_api", requestTrace, &traceResponse{StatusCode: http.StatusOK, Body: payload}, nil)

	b.writeJSON(w, http.StatusOK, map[string]any{
		"proxied":         true,
		"browserHasToken": false,
		"resourceData":    payload,
	})
}

func (b *bffApp) sessionFromRequest(r *http.Request) (*browserSession, bool) {
	storedSession, ok := b.sessionManager.Get(r.Context(), browserSessionKey).(*browserSession)
	if !ok || storedSession == nil {
		return nil, false
	}
	return storedSession, ok
}

func (b *bffApp) validCSRFRequest(r *http.Request) bool {
	// This demo trusts Fetch Metadata and intentionally rejects browsers that do not send it.
	switch r.Header.Get("Sec-Fetch-Site") {
	case "same-origin", "same-site", "none":
		return true
	default:
		return false
	}
}

func (b *bffApp) ensureFreshAccessToken(ctx context.Context, storedSession *browserSession) error {
	// Refresh just before expiry so the browser session stays opaque and long-lived.
	// Claim the refresh slot under lock to prevent a concurrent double-refresh that
	// would burn the single-use refresh token on the first call and fail the second.
	b.mu.Lock()
	if time.Now().Unix() < storedSession.AccessTokenExp-30 {
		b.mu.Unlock()
		return nil
	}
	if storedSession.RefreshToken == "" {
		b.mu.Unlock()
		return errors.New("session has no refresh_token")
	}
	if storedSession.refreshing {
		b.mu.Unlock()
		return nil
	}
	storedSession.refreshing = true
	b.mu.Unlock()

	updated, err := b.refreshTokens(storedSession.RefreshToken)
	b.mu.Lock()
	storedSession.refreshing = false
	b.mu.Unlock()
	if err != nil {
		return err
	}

	accessClaims, err := b.validateJWTClaims(updated.AccessToken, b.resourceAudience)
	if err != nil {
		return err
	}
	idClaims, err := b.validateJWTClaims(updated.IDToken, b.clientID)
	if err != nil {
		return err
	}

	b.mu.Lock()
	storedSession.AccessToken = updated.AccessToken
	storedSession.AccessTokenExp = readNumericClaim(accessClaims, "exp")
	storedSession.RefreshToken = updated.RefreshToken
	storedSession.User = userProfileFromClaims(idClaims)
	storedSession.Scope = updated.Scope
	b.mu.Unlock()
	b.sessionManager.Put(ctx, browserSessionKey, storedSession)
	return nil
}

func userProfileFromClaims(claims map[string]any) userProfile {
	return userProfile{
		Sub:               readStringClaim(claims, "sub"),
		PreferredUsername: readStringClaim(claims, "preferred_username"),
		Name:              readStringClaim(claims, "name"),
		Email:             readStringClaim(claims, "email"),
		Roles:             readStringSliceClaim(claims, "roles"),
	}
}
