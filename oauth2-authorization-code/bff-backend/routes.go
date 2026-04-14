package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func (b *bffApp) routes() http.Handler {
	router := chi.NewRouter()
	router.Use(b.withLogging)
	router.Use(b.withCORS)
	router.Use(b.sessionManager.LoadAndSave)
	router.Get("/healthz", b.handleHealth)
	router.Get("/auth/login", b.handleLogin)
	router.Get("/auth/callback", b.handleCallback)
	router.Post("/auth/logout", b.handleLogout)
	router.Get("/api/profile", b.handleProfile)
	router.Get("/api/session", b.handleSession)
	router.Get("/api/data", b.handleData)
	return router
}

func (b *bffApp) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(started))
	})
}

func (b *bffApp) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); b.allowOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			if origin == b.frontendOrigin {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (b *bffApp) handleHealth(w http.ResponseWriter, r *http.Request) {
	b.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (b *bffApp) allowOrigin(origin string) bool {
	return origin == b.frontendOrigin || origin == b.pkceFrontendOrigin
}

func (b *bffApp) writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}
