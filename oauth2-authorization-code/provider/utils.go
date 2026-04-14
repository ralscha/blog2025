package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"slices"
	"strings"
	"time"
)

func parseScopes(value string) []string {
	if value == "" {
		return nil
	}
	seen := map[string]bool{}
	result := make([]string, 0)
	for part := range strings.FieldsSeq(value) {
		if !seen[part] {
			seen[part] = true
			result = append(result, part)
		}
	}
	return result
}

func allScopesAllowed(requested []string, allowed []string) bool {
	for _, scope := range requested {
		if !slices.Contains(allowed, scope) {
			return false
		}
	}
	return true
}

func isSubset(requested []string, allowed []string) bool {
	for _, scope := range requested {
		if !slices.Contains(allowed, scope) {
			return false
		}
	}
	return true
}

func randomToken(length int) (string, error) {
	buffer := make([]byte, length)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func computePKCEChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func validTimeClaims(claims map[string]any) bool {
	now := time.Now().Unix()
	exp, ok := readNumericClaim(claims, "exp")
	if !ok || exp < now {
		return false
	}
	if nbf, ok := readNumericClaim(claims, "nbf"); ok && nbf > now {
		return false
	}
	return true
}

func readNumericClaim(claims map[string]any, key string) (int64, bool) {
	value, ok := claims[key]
	if !ok {
		return 0, false
	}
	floatValue, ok := value.(float64)
	if !ok {
		return 0, false
	}
	return int64(floatValue), true
}

func mustNumericClaim(claims map[string]any, key string) int64 {
	value, _ := readNumericClaim(claims, key)
	return value
}

func readStringClaim(claims map[string]any, key string) string {
	value, _ := claims[key].(string)
	return value
}

func readStringSliceClaim(claims map[string]any, key string) []string {
	raw, ok := claims[key].([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(raw))
	for _, value := range raw {
		if stringValue, ok := value.(string); ok {
			result = append(result, stringValue)
		}
	}
	return result
}

func bearerToken(header string) string {
	token := strings.TrimPrefix(header, "Bearer ")
	if token == "" || token == header {
		return ""
	}
	return token
}
