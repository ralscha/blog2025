package main

import (
	"errors"
	"fmt"
	"maps"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
)

func (b *bffApp) validateJWTClaims(token string, audience string) (map[string]any, error) {
	verifier, err := b.verifierForAudience(audience)
	if err != nil {
		return nil, err
	}
	if _, err := verifier.Verify(b.oidcContext, token); err != nil {
		return nil, err
	}
	return parseVerifiedJWTClaims(token)
}
func (b *bffApp) verifierForAudience(audience string) (*oidc.IDTokenVerifier, error) {
	switch audience {
	case b.clientID:
		return b.idTokenVerifier, nil
	case b.resourceAudience:
		return b.accessTokenVerifier, nil
	default:
		return nil, fmt.Errorf("unsupported audience %q", audience)
	}
}

func parseVerifiedJWTClaims(token string) (map[string]any, error) {
	parsedToken, _, err := jwt.NewParser().ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("verified token did not contain map claims")
	}
	result := make(map[string]any, len(claims))
	maps.Copy(result, claims)
	return result, nil
}
