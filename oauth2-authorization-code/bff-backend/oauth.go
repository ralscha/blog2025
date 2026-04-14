package main

import (
	"errors"
	"net/url"
	"time"

	"golang.org/x/oauth2"
)

func (b *bffApp) exchangeCode(code string, codeVerifier string) (oauthTokenResponse, error) {
	values := url.Values{}
	values.Set("grant_type", "authorization_code")
	values.Set("code", code)
	values.Set("redirect_uri", b.redirectURI)
	values.Set("code_verifier", codeVerifier)
	requestTrace := traceRequest{
		Method:           "POST",
		URL:              b.oauthConfig.Endpoint.TokenURL,
		Headers:          map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Form:             map[string][]string(values),
		ClientID:         b.clientID,
		ClientAuthMethod: "client_secret_basic",
		Notes:            "BFF exchanges the authorization code server-side and binds it with PKCE.",
	}
	token, err := b.oauthConfig.Exchange(b.oidcContext, code, oauth2.VerifierOption(codeVerifier))
	if err != nil {
		b.traceLogger.Write("token_exchange", requestTrace, nil, err)
		return oauthTokenResponse{}, err
	}
	tokens, err := oauthTokenResponseFromToken(token)
	if err != nil {
		b.traceLogger.Write("token_exchange", requestTrace, nil, err)
		return oauthTokenResponse{}, err
	}
	b.traceLogger.Write("token_exchange", requestTrace, &traceResponse{StatusCode: 200, Body: tokens}, nil)
	return tokens, nil
}

func (b *bffApp) refreshTokens(refreshToken string) (oauthTokenResponse, error) {
	values := url.Values{}
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", refreshToken)
	requestTrace := traceRequest{
		Method:           "POST",
		URL:              b.oauthConfig.Endpoint.TokenURL,
		Headers:          map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Form:             map[string][]string(values),
		ClientID:         b.clientID,
		ClientAuthMethod: "client_secret_basic",
		Notes:            "BFF refreshes tokens without exposing the refresh token to the browser.",
	}
	tokenSource := b.oauthConfig.TokenSource(b.oidcContext, &oauth2.Token{
		RefreshToken: refreshToken,
		Expiry:       time.Now().Add(-time.Minute),
	})
	token, err := tokenSource.Token()
	if err != nil {
		b.traceLogger.Write("token_refresh", requestTrace, nil, err)
		return oauthTokenResponse{}, err
	}
	tokens, err := oauthTokenResponseFromToken(token)
	if err != nil {
		b.traceLogger.Write("token_refresh", requestTrace, nil, err)
		return oauthTokenResponse{}, err
	}
	b.traceLogger.Write("token_refresh", requestTrace, &traceResponse{StatusCode: 200, Body: tokens}, nil)
	return tokens, nil
}

func oauthTokenResponseFromToken(token *oauth2.Token) (oauthTokenResponse, error) {
	if token == nil {
		return oauthTokenResponse{}, errors.New("token exchange returned no token")
	}
	idToken, _ := token.Extra("id_token").(string)
	if idToken == "" {
		return oauthTokenResponse{}, errors.New("token exchange did not return an id_token")
	}
	scope, _ := token.Extra("scope").(string)
	expiresIn := int64(0)
	if !token.Expiry.IsZero() {
		expiresIn = max(int64(time.Until(token.Expiry).Seconds()), 0)
	}
	return oauthTokenResponse{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		ExpiresIn:    expiresIn,
		RefreshToken: token.RefreshToken,
		Scope:        scope,
		IDToken:      idToken,
	}, nil
}

func (b *bffApp) revokeToken(token string, tokenTypeHint string) error {
	if token == "" {
		return nil
	}
	values := url.Values{}
	values.Set("token", token)
	values.Set("token_type_hint", tokenTypeHint)
	requestTrace := traceRequest{
		Method:           "POST",
		URL:              b.revokeURL,
		Headers:          map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Form:             map[string][]string(values),
		ClientID:         b.clientID,
		ClientAuthMethod: "client_secret_basic",
		TokenTypeHint:    tokenTypeHint,
	}
	var responseBody any
	statusCode, err := b.postOAuthForm(b.revokeURL, b.clientID, b.clientSecret, values, &responseBody)
	if err != nil {
		b.traceLogger.Write("token_revoke", requestTrace, nil, err)
		return err
	}
	b.traceLogger.Write("token_revoke", requestTrace, &traceResponse{StatusCode: statusCode, Body: responseBody}, nil)
	if statusCode >= 400 {
		return errors.New("revocation endpoint rejected the token")
	}
	return nil
}
