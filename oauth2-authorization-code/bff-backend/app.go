package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const browserSessionKey = "browserSession"

type bffApp struct {
	issuer                    string
	revokeURL                 string
	frontendOrigin            string
	pkceFrontendOrigin        string
	redirectURI               string
	clientID                  string
	clientSecret              string
	resourceAPIURL            string
	resourceAudience          string
	introspectionURL          string
	introspectionClientID     string
	introspectionClientSecret string
	oauthConfig               *oauth2.Config
	oidcContext               context.Context
	httpClient                *http.Client
	idTokenVerifier           *oidc.IDTokenVerifier
	accessTokenVerifier       *oidc.IDTokenVerifier
	sessionManager            *scs.SessionManager
	traceLogger               *httpTraceLogger
	mu                        sync.Mutex
	pending                   map[string]pendingLogin
}

type pendingLogin struct {
	User         string
	Nonce        string
	CodeVerifier string
	ExpiresAt    time.Time
}

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token"`
}

type browserSession struct {
	ID              string
	User            userProfile
	AccessToken     string
	AccessTokenExp  int64
	RefreshToken    string
	RefreshTokenExp time.Time
	Scope           string
	CreatedAt       time.Time
	refreshing      bool
}

type userProfile struct {
	Sub               string
	PreferredUsername string
	Name              string
	Email             string
	Roles             []string
}

type introspectionResponse struct {
	Active   bool   `json:"active"`
	TokenUse string `json:"token_use"`
	ClientID string `json:"client_id"`
	Scope    string `json:"scope"`
}

func newBFFApp() (*bffApp, error) {
	gob.Register(&browserSession{})
	gob.Register(userProfile{})

	traceLogger, err := newHTTPTraceLoggerFromEnv()
	if err != nil {
		return nil, err
	}

	issuer := "http://localhost:8080"
	redirectURI := "http://localhost:8082/auth/callback"
	clientID := "bff-client"
	clientSecret := "bff-secret"
	resourceAudience := "pkce-api"
	httpClient := &http.Client{Timeout: 10 * time.Second}
	oidcContext := oidc.ClientContext(context.Background(), httpClient)
	provider, err := newProviderWithRetry(oidcContext, issuer, 20, 250*time.Millisecond)
	if err != nil {
		return nil, err
	}
	oauthConfig := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{"openid", "profile", "email", "roles", "offline_access", "api.read"},
	}
	sessionManager := scs.New()
	sessionManager.Store = memstore.New()
	sessionManager.Cookie.Name = "oidc_bff_session"
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	sessionManager.Cookie.Path = "/"
	sessionManager.Cookie.Persist = true
	// Secure must be true in production (HTTPS). Set to false here because the
	// local demo runs over plain HTTP.
	sessionManager.Cookie.Secure = false
	sessionManager.Lifetime = 45 * time.Minute

	b := &bffApp{
		issuer:                    issuer,
		revokeURL:                 issuer + "/revoke",
		frontendOrigin:            "http://localhost:4201",
		pkceFrontendOrigin:        "http://localhost:4200",
		redirectURI:               redirectURI,
		clientID:                  clientID,
		clientSecret:              clientSecret,
		resourceAPIURL:            "http://localhost:8082/api/profile",
		resourceAudience:          resourceAudience,
		introspectionURL:          issuer + "/introspect",
		introspectionClientID:     "resource-server",
		introspectionClientSecret: "resource-secret",
		oauthConfig:               oauthConfig,
		oidcContext:               oidcContext,
		httpClient:                httpClient,
		idTokenVerifier:           provider.Verifier(&oidc.Config{ClientID: clientID}),
		accessTokenVerifier:       provider.Verifier(&oidc.Config{ClientID: resourceAudience}),
		sessionManager:            sessionManager,
		traceLogger:               traceLogger,
		pending:                   map[string]pendingLogin{},
	}
	go b.cleanPendingLogins()
	return b, nil
}

func newProviderWithRetry(ctx context.Context, issuer string, attempts int, delay time.Duration) (*oidc.Provider, error) {
	var lastErr error

	for attempt := 1; attempt <= attempts; attempt++ {
		provider, err := oidc.NewProvider(ctx, issuer)
		if err == nil {
			return provider, nil
		}

		lastErr = err
		if attempt == attempts {
			break
		}

		time.Sleep(delay)
	}

	return nil, fmt.Errorf("discover oidc provider after %d attempts: %w", attempts, lastErr)
}
