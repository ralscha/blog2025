package main

import (
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

func (b *bffApp) handleProfile(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		b.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
		return
	}

	payload, err := b.resourceProfilePayload(token)
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		b.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	b.writeJSON(w, http.StatusOK, payload)
}

func (b *bffApp) resourceProfilePayload(token string) (map[string]any, error) {
	claims, err := b.validateResourceAccessToken(token)
	if err != nil {
		return nil, err
	}

	roles := readStringSliceClaim(claims, "roles")
	message := "Reader data: you can inspect profile information."
	if slices.Contains(roles, "admin") {
		message = "Admin data: you can inspect deployment switches and role-gated data."
	}

	return map[string]any{
		"subject": readStringClaim(claims, "sub"),
		"name":    readStringClaim(claims, "name"),
		"email":   readStringClaim(claims, "email"),
		"roles":   roles,
		"scope":   readStringClaim(claims, "scope"),
		"message": message,
		"token": map[string]any{
			"issuer":         readStringClaim(claims, "iss"),
			"audience":       readStringClaim(claims, "aud"),
			"clientId":       readStringClaim(claims, "client_id"),
			"expiresAtEpoch": readNumericClaim(claims, "exp"),
		},
	}, nil
}

func (b *bffApp) validateResourceAccessToken(token string) (map[string]any, error) {
	claims, err := b.validateJWTClaims(token, b.resourceAudience)
	if err != nil {
		return nil, err
	}
	if readStringClaim(claims, "token_use") != "access_token" {
		return nil, errors.New("token_use must be access_token")
	}
	if !strings.Contains(readStringClaim(claims, "scope"), "api.read") {
		return nil, errors.New("missing api.read scope")
	}
	if err := b.introspectAccessToken(token); err != nil {
		return nil, err
	}
	return claims, nil
}

func (b *bffApp) introspectAccessToken(token string) error {
	values := url.Values{}
	values.Set("token", token)
	values.Set("token_type_hint", "access_token")

	var payload introspectionResponse
	statusCode, err := b.postOAuthForm(
		b.introspectionURL,
		b.introspectionClientID,
		b.introspectionClientSecret,
		values,
		&payload,
	)
	if err != nil {
		return err
	}
	if statusCode >= http.StatusBadRequest {
		return errors.New("introspection endpoint rejected the access token")
	}
	if !payload.Active {
		return errors.New("token is inactive or revoked")
	}
	if payload.TokenUse != "access_token" {
		return errors.New("introspection returned the wrong token_use")
	}
	if !strings.Contains(payload.Scope, "api.read") {
		return errors.New("introspection reported a missing api.read scope")
	}
	return nil
}
