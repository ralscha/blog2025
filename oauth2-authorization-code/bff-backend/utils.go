package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func readStringClaim(claims map[string]any, key string) string {
	value, _ := claims[key].(string)
	return value
}

func readNumericClaim(claims map[string]any, key string) int64 {
	value, _ := claims[key].(float64)
	return int64(value)
}

func readStringSliceClaim(claims map[string]any, key string) []string {
	raw, ok := claims[key].([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		if stringValue, ok := item.(string); ok {
			result = append(result, stringValue)
		}
	}
	return result
}

func decodeJSONResponse(response *http.Response, target any) error {
	return json.NewDecoder(response.Body).Decode(target)
}

func newFormRequest(method string, requestURL string, values url.Values) (*http.Request, error) {
	request, err := http.NewRequest(method, requestURL, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return request, nil
}

func (b *bffApp) postOAuthForm(
	requestURL string,
	clientID string,
	clientSecret string,
	values url.Values,
	target any,
) (int, error) {
	request, err := newFormRequest(http.MethodPost, requestURL, values)
	if err != nil {
		return 0, err
	}
	if clientID != "" {
		request.SetBasicAuth(clientID, clientSecret)
	}

	response, err := b.httpClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer closeResponseBody(response.Body)

	if target != nil {
		if err := decodeJSONResponse(response, target); err != nil && !errors.Is(err, io.EOF) {
			return response.StatusCode, err
		}
	}

	return response.StatusCode, nil
}

func closeResponseBody(body io.Closer) {
	if err := body.Close(); err != nil {
		log.Printf("close response body: %v", err)
	}
}

func bearerToken(header string) string {
	token := strings.TrimPrefix(header, "Bearer ")
	if token == "" || token == header {
		return ""
	}
	return token
}

func computePKCEChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func randomToken(length int) (string, error) {
	buffer := make([]byte, length)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
