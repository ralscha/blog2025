package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type UserData struct {
	UserID    string  `json:"userId"`
	Balance   float64 `json:"balance"`
	LastLogin string  `json:"lastLogin"`
}

type SecureDataResponse struct {
	Message           string   `json:"message"`
	ClientCertificate string   `json:"clientCertificate"`
	Authenticated     bool     `json:"authenticated"`
	Timestamp         int64    `json:"timestamp"`
	Data              UserData `json:"data"`
}

type UpdateDataResponse struct {
	Message           string         `json:"message"`
	ClientCertificate string         `json:"clientCertificate"`
	ReceivedData      map[string]any `json:"receivedData"`
	Timestamp         int64          `json:"timestamp"`
}

func main() {
	caCert, err := os.ReadFile("ca-cert.pem")
	if err != nil {
		log.Fatal("Error reading CA certificate:", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		log.Fatal("Failed to parse CA certificate")
	}

	serverCert, err := tls.LoadX509KeyPair("server-cert.pem", "server-key.pem")
	if err != nil {
		log.Fatal("Error loading server certificate:", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert, /* tls.VerifyClientCertIfGiven, */
		MinVersion:   tls.VersionTLS13,
	}

	mux := http.NewServeMux()

	server := &http.Server{
		Addr:         ":8443",
		TLSConfig:    tlsConfig,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		Handler:      mux,
	}

	mux.HandleFunc("GET /api/public/health", healthHandler)
	mux.HandleFunc("GET /api/secure/data", secureDataHandler)
	mux.HandleFunc("POST /api/secure/update", updateDataHandler)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		fmt.Println("Starting server on https://localhost:8443")
		fmt.Println("Press Ctrl+C to stop")
		if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	} else {
		log.Println("Server shutdown complete")
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:  "UP",
		Message: "Public endpoint - no authentication required",
	}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

func secureDataHandler(w http.ResponseWriter, r *http.Request) {
	if len(r.TLS.PeerCertificates) == 0 {
		http.Error(w, "No client certificate", http.StatusUnauthorized)
		return
	}

	clientCert := r.TLS.PeerCertificates[0]
	cn := getCN(clientCert.Subject.String())

	if !strings.EqualFold(cn, "demo-client") {
		http.Error(w, "Certificate not authorized", http.StatusUnauthorized)
		return
	}

	userData := UserData{
		UserID:    "12345",
		Balance:   1500.75,
		LastLogin: "2025-08-12T10:30:00Z",
	}

	response := SecureDataResponse{
		Message:           "Access granted to secure endpoint",
		ClientCertificate: cn,
		Authenticated:     true,
		Timestamp:         time.Now().UnixMilli(),
		Data:              userData,
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

func updateDataHandler(w http.ResponseWriter, r *http.Request) {
	if len(r.TLS.PeerCertificates) == 0 {
		http.Error(w, "No client certificate", http.StatusUnauthorized)
		return
	}

	clientCert := r.TLS.PeerCertificates[0]
	cn := getCN(clientCert.Subject.String())

	if !strings.EqualFold(cn, "demo-client") {
		http.Error(w, "Certificate not authorized", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1048576) // 1MB limit

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var receivedData map[string]any
	if err := json.Unmarshal(body, &receivedData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response := UpdateDataResponse{
		Message:           "Data updated successfully",
		ClientCertificate: cn,
		ReceivedData:      receivedData,
		Timestamp:         time.Now().UnixMilli(),
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

func getCN(subject string) string {
	parts := strings.Split(subject, ",")
	for _, part := range parts {
		if strings.HasPrefix(strings.TrimSpace(part), "CN=") {
			return strings.TrimPrefix(strings.TrimSpace(part), "CN=")
		}
	}
	return ""
}
