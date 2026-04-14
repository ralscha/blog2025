package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	application, err := newApp()
	if err != nil {
		log.Fatalf("failed to initialize provider: %v", err)
	}

	server := &http.Server{
		Addr:              ":" + application.cfg.Port,
		Handler:           application.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("provider listening on %s", application.cfg.Issuer)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("provider failed: %v", err)
	}
}
