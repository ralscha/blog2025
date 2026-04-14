package main

import (
	"sync"
	"time"
)

type app struct {
	cfg              Config
	signer           *Signer
	users            map[string]User
	clients          map[string]Client
	authCodes        map[string]AuthorizationCode
	refreshTokens    map[string]RefreshGrant
	providerSessions map[string]ProviderSession
	revokedTokenIDs  map[string]time.Time
	mu               sync.Mutex
}

func newApp() (*app, error) {
	signer, err := newSigner()
	if err != nil {
		return nil, err
	}

	cfg := loadConfig()
	allowedScopes := []string{"openid", "profile", "email", "roles", "offline_access", "api.read"}

	return &app{
		cfg:    cfg,
		signer: signer,
		users: map[string]User{
			"alice": {
				Username: "alice",
				Subject:  "user-alice",
				Name:     "Alice Admin",
				Email:    "alice@example.test",
				Roles:    []string{"reader", "admin"},
			},
			"bob": {
				Username: "bob",
				Subject:  "user-bob",
				Name:     "Bob Builder",
				Email:    "bob@example.test",
				Roles:    []string{"reader"},
			},
		},
		clients: map[string]Client{
			cfg.PublicClientID: {
				ID:           cfg.PublicClientID,
				Public:       true,
				RequirePKCE:  true,
				RedirectURIs: []string{cfg.PKCERedirectURI},
				Scopes:       allowedScopes,
			},
			cfg.ConfidentialID: {
				ID:           cfg.ConfidentialID,
				Secret:       cfg.ConfidentialSecret,
				RedirectURIs: []string{cfg.BFFRedirectURI},
				Scopes:       allowedScopes,
			},
			cfg.ResourceServerClientID: {
				ID:           cfg.ResourceServerClientID,
				Secret:       cfg.ResourceServerClientSecret,
				RedirectURIs: nil,
				Scopes:       []string{"introspect"},
			},
		},
		authCodes:        map[string]AuthorizationCode{},
		refreshTokens:    map[string]RefreshGrant{},
		providerSessions: map[string]ProviderSession{},
		revokedTokenIDs:  map[string]time.Time{},
	}, nil
}
