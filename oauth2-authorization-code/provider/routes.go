package main

import (
	"encoding/json"
	"html"
	"log"
	"net/http"
	"net/url"
	"time"
)

func (a *app) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleIndex)
	mux.HandleFunc("/healthz", a.handleHealth)
	mux.HandleFunc("/.well-known/openid-configuration", a.handleDiscovery)
	mux.HandleFunc("/jwks.json", a.handleJWKS)
	mux.HandleFunc("/login", a.handleLogin)
	mux.HandleFunc("/authorize", a.handleAuthorize)
	mux.HandleFunc("/token", a.handleToken)
	mux.HandleFunc("/userinfo", a.handleUserInfo)
	mux.HandleFunc("/logout", a.handleLogout)
	mux.HandleFunc("/revoke", a.handleRevocation)
	mux.HandleFunc("/introspect", a.handleIntrospection)

	return a.withLogging(a.withCORS(mux))
}

func (a *app) handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.renderLoginPage(w, r, "")
	case http.MethodPost:
		a.completeLogin(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *app) renderLoginPage(w http.ResponseWriter, r *http.Request, errorMessage string) {
	returnTo := r.URL.Query().Get("return_to")
	if returnTo == "" {
		returnTo = "/authorize"
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte("<!doctype html><html><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><title>Demo Provider Login</title><style>body{margin:0;font-family:Segoe UI,Aptos,sans-serif;background:linear-gradient(135deg,#f0f5f8,#dde7ef);color:#102033}.shell{min-height:100vh;display:grid;place-items:center;padding:24px}.card{width:min(560px,100%);background:#ffffffd9;border:1px solid #c9d5e2;border-radius:24px;box-shadow:0 24px 60px rgba(16,32,51,.12);padding:28px}.eyebrow{font-size:.8rem;text-transform:uppercase;letter-spacing:.12em;color:#4d647c;margin:0 0 12px}h1{margin:0 0 12px;font-size:2rem}.muted{color:#4d647c;line-height:1.5}.grid{display:grid;gap:12px;margin-top:20px}.user-button{border:0;border-radius:18px;padding:16px 18px;text-align:left;background:linear-gradient(135deg,#114b9b,#3e73bf);color:#fff;font:inherit}.user-button.secondary{background:linear-gradient(135deg,#d66a2b,#e38f54)}.pill{display:inline-block;margin-top:8px;padding:6px 10px;border-radius:999px;background:#eff4fb;color:#114b9b;font-size:.85rem}.error{margin-top:16px;border-radius:16px;background:#fbe9e7;color:#8a2419;padding:12px 14px;border:1px solid #f1c1ba}code{font-family:Consolas,monospace;background:#f4f7fb;padding:2px 6px;border-radius:6px}</style></head><body><main class=\"shell\"><section class=\"card\"><p class=\"eyebrow\">Interactive Provider Login</p><h1>Choose a hardcoded user</h1><p class=\"muted\">This tiny provider page exists purely to compare <code>prompt=login</code> and <code>prompt=none</code>. It creates a provider session cookie and sends the browser back to the original authorization request.</p>"))
	if errorMessage != "" {
		_, _ = w.Write([]byte("<div class=\"error\">" + html.EscapeString(errorMessage) + "</div>"))
	}
	_, _ = w.Write([]byte("<form method=\"post\" class=\"grid\"><input type=\"hidden\" name=\"return_to\" value=\"" + html.EscapeString(returnTo) + "\"><button class=\"user-button\" type=\"submit\" name=\"user\" value=\"alice\">Alice Admin<span class=\"pill\">roles: reader, admin</span></button><button class=\"user-button secondary\" type=\"submit\" name=\"user\" value=\"bob\">Bob Builder<span class=\"pill\">roles: reader</span></button></form></section></main></body></html>"))
}

func (a *app) completeLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.renderLoginPage(w, r, "The login form submission could not be parsed.")
		return
	}

	username := r.Form.Get("user")
	user, ok := a.users[username]
	if !ok {
		query := url.Values{}
		query.Set("return_to", r.Form.Get("return_to"))
		r.URL.RawQuery = query.Encode()
		a.renderLoginPage(w, r, "Pick one of the demo users to continue.")
		return
	}

	returnTo := r.Form.Get("return_to")
	if returnTo == "" {
		returnTo = "/authorize"
	}

	sessionID, err := a.createProviderSession(user)
	if err != nil {
		query := url.Values{}
		query.Set("return_to", returnTo)
		r.URL.RawQuery = query.Encode()
		a.renderLoginPage(w, r, "The provider could not create a session.")
		return
	}
	a.setProviderSessionCookie(w, sessionID)

	redirect, err := url.Parse(returnTo)
	if err != nil || redirect.Host != "" || redirect.Scheme != "" {
		a.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "return_to must be a relative path")
		return
	}
	values := redirect.Query()
	values.Del("prompt")
	values.Del("login_hint")
	redirect.RawQuery = values.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

func (a *app) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(started))
	})
}

func (a *app) withCORS(next http.Handler) http.Handler {
	allowed := map[string]bool{
		a.cfg.PKCEOrigin: true,
		a.cfg.BFFOrigin:  true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *app) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<html><body><h1>Demo OIDC Provider</h1><p>Available endpoints:</p><ul><li><a href="/.well-known/openid-configuration">discovery</a></li><li><a href="/jwks.json">jwks</a></li><li><a href="/logout">logout</a></li></ul></body></html>`))
}

func (a *app) handleHealth(w http.ResponseWriter, r *http.Request) {
	a.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "issuer": a.cfg.Issuer})
}

func (a *app) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	a.writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                a.cfg.Issuer,
		"authorization_endpoint":                a.cfg.Issuer + "/authorize",
		"token_endpoint":                        a.cfg.Issuer + "/token",
		"userinfo_endpoint":                     a.cfg.Issuer + "/userinfo",
		"jwks_uri":                              a.cfg.Issuer + "/jwks.json",
		"revocation_endpoint":                   a.cfg.Issuer + "/revoke",
		"introspection_endpoint":                a.cfg.Issuer + "/introspect",
		"end_session_endpoint":                  a.cfg.Issuer + "/logout",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email", "roles", "offline_access", "api.read"},
		"token_endpoint_auth_methods_supported": []string{"none", "client_secret_basic", "client_secret_post"},
		"claims_supported":                      []string{"sub", "name", "email", "preferred_username", "roles"},
		"code_challenge_methods_supported":      []string{"S256"},
	})
}

func (a *app) handleJWKS(w http.ResponseWriter, r *http.Request) {
	a.writeJSON(w, http.StatusOK, jwksDocument{Keys: []jwk{a.signer.JWK()}})
}

func (a *app) writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func (a *app) writeOAuthError(w http.ResponseWriter, status int, code string, description string) {
	a.writeJSON(w, status, map[string]string{
		"error":             code,
		"error_description": description,
	})
}

func (a *app) writeAuthorizeError(w http.ResponseWriter, redirectURI string, state string, code string, description string) {
	redirect, err := url.Parse(redirectURI)
	if err != nil {
		a.writeOAuthError(w, http.StatusBadRequest, code, description)
		return
	}
	values := redirect.Query()
	values.Set("error", code)
	values.Set("error_description", description)
	if state != "" {
		values.Set("state", state)
	}
	redirect.RawQuery = values.Encode()
	w.Header().Set("Location", redirect.String())
	w.WriteHeader(http.StatusFound)
}
