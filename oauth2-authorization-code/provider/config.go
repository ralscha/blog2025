package main

import "time"

type Config struct {
	Issuer                     string
	Port                       string
	PKCERedirectURI            string
	BFFRedirectURI             string
	PKCEOrigin                 string
	BFFOrigin                  string
	ResourceAudience           string
	ProviderSessionCookieName  string
	CodeTTL                    time.Duration
	AccessTokenTTL             time.Duration
	RefreshTokenTTL            time.Duration
	ProviderSessionTTL         time.Duration
	ConfidentialID             string
	ConfidentialSecret         string
	PublicClientID             string
	ResourceServerClientID     string
	ResourceServerClientSecret string
}

func loadConfig() Config {
	return Config{
		Issuer:                     "http://localhost:8080",
		Port:                       "8080",
		PKCERedirectURI:            "http://localhost:4200/callback",
		BFFRedirectURI:             "http://localhost:8082/auth/callback",
		PKCEOrigin:                 "http://localhost:4200",
		BFFOrigin:                  "http://localhost:4201",
		ResourceAudience:           "pkce-api",
		ProviderSessionCookieName:  "provider_session",
		CodeTTL:                    2 * time.Minute,
		AccessTokenTTL:             5 * time.Minute,
		RefreshTokenTTL:            45 * time.Minute,
		ProviderSessionTTL:         30 * time.Minute,
		ConfidentialID:             "bff-client",
		ConfidentialSecret:         "bff-secret",
		PublicClientID:             "pkce-spa",
		ResourceServerClientID:     "resource-server",
		ResourceServerClientSecret: "resource-secret",
	}
}
