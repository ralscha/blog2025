package main

import "time"

type User struct {
	Username string   `json:"preferred_username"`
	Subject  string   `json:"sub"`
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
}

type Client struct {
	ID           string
	Secret       string
	RedirectURIs []string
	Public       bool
	RequirePKCE  bool
	Scopes       []string
}

type AuthorizationCode struct {
	Value               string
	ClientID            string
	RedirectURI         string
	Scopes              []string
	User                User
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
	ExpiresAt           time.Time
	Used                bool
}

type RefreshGrant struct {
	Value     string
	ClientID  string
	Scopes    []string
	User      User
	ExpiresAt time.Time
	Used      bool
}

type ProviderSession struct {
	ID        string
	User      User
	ExpiresAt time.Time
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token"`
	IssuedToken  string `json:"issued_token_type,omitempty"`
}

type jwksDocument struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}
