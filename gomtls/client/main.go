package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type UpdateRequest struct {
	Action    string `json:"action"`
	Value     string `json:"value"`
	Timestamp int64  `json:"timestamp"`
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

	clientCert, err := tls.LoadX509KeyPair("client-cert.pem", "client-key.pem")
	if err != nil {
		log.Fatal("Error loading client certificate:", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS13,
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	fmt.Println("Testing mTLS connection with client certificate...")

	testPublicEndpoint(client)
	testSecureGetEndpoint(client)
	testSecurePostEndpoint(client)

	fmt.Println("\nTesting without client certificate...")
	tlsConfigNoCert := &tls.Config{
		RootCAs:    caCertPool,
		MinVersion: tls.VersionTLS13,
	}

	transportNoCert := &http.Transport{
		TLSClientConfig: tlsConfigNoCert,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
	}

	clientNoCert := &http.Client{
		Transport: transportNoCert,
		Timeout:   30 * time.Second,
	}

	testPublicEndpoint(clientNoCert)
	testSecureGetEndpoint(clientNoCert)
}

func testPublicEndpoint(client *http.Client) {
	fmt.Println("1. Testing public endpoint (no authentication required):")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://localhost:8443/api/public/health", nil)
	if err != nil {
		fmt.Printf("   Error creating request: %v\n", err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("   Error reading response: %v\n", err)
		return
	}

	fmt.Printf("   Status: %d\n", resp.StatusCode)
	fmt.Printf("   Response: %s\n", string(body))
}

func testSecureGetEndpoint(client *http.Client) {
	fmt.Println("2. Testing secure GET endpoint:")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://localhost:8443/api/secure/data", nil)
	if err != nil {
		fmt.Printf("   Error creating request: %v\n", err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("   Error reading response: %v\n", err)
		return
	}

	fmt.Printf("   Status: %d\n", resp.StatusCode)
	if resp.StatusCode == 200 {
		fmt.Printf("   Response: %s\n", string(body))
	} else {
		fmt.Printf("   Error response: %s\n", string(body))
	}
}

func testSecurePostEndpoint(client *http.Client) {
	fmt.Println("3. Testing secure POST endpoint:")
	updateReq := UpdateRequest{
		Action:    "update_balance",
		Value:     "2000.00",
		Timestamp: time.Now().UnixMilli(),
	}

	jsonData, err := json.Marshal(updateReq)
	if err != nil {
		fmt.Printf("   Error marshaling JSON: %v\n", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://localhost:8443/api/secure/update", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("   Error creating request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("   Error reading response: %v\n", err)
		return
	}

	fmt.Printf("   Status: %d\n", resp.StatusCode)
	if resp.StatusCode == 200 {
		fmt.Printf("   Response: %s\n", string(body))
	} else {
		fmt.Printf("   Error response: %s\n", string(body))
	}
}
