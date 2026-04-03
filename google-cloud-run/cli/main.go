package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type temperatureResponse struct {
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	TemperatureC int       `json:"temperatureC"`
	GeneratedAt  time.Time `json:"generatedAt"`
	Revision     string    `json:"revision"`
}

func main() {
	if err := loadDotEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "load .env: %v\n", err)
		os.Exit(1)
	}

	baseURL := flag.String("url", envOrDefault("TEMPERATURE_SERVICE_URL", "http://localhost:8080"), "Base URL for the temperature service")
	lat := flag.Float64("lat", 37.7749, "Latitude")
	lng := flag.Float64("lng", -122.4194, "Longitude")
	apiKey := flag.String("api-key", envOrDefault("TEMPERATURE_SERVICE_API_KEY", ""), "API key sent as X-API-Key")
	timeout := flag.Duration("timeout", 8*time.Second, "HTTP timeout")
	flag.Parse()

	endpoint, err := buildEndpoint(*baseURL, *lat, *lng)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid request: %v\n", err)
		os.Exit(1)
	}

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create request: %v\n", err)
		os.Exit(1)
	}
	if *apiKey != "" {
		req.Header.Set("X-API-Key", *apiKey)
	}

	client := &http.Client{Timeout: *timeout}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "service returned %s\n", resp.Status)
		os.Exit(1)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read response: %v\n", err)
		os.Exit(1)
	}

	var result temperatureResponse
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "decode response: %v\n", err)
		os.Exit(1)
	}
	if err := validateTemperatureResponse(result); err != nil {
		fmt.Fprintf(os.Stderr, "unexpected response payload: %v\n", err)
		fmt.Fprintf(os.Stderr, "response body: %s\n", strings.TrimSpace(string(body)))
		os.Exit(1)
	}

	fmt.Printf("Latitude: %.4f\n", result.Latitude)
	fmt.Printf("Longitude: %.4f\n", result.Longitude)
	fmt.Printf("Temperature: %d C\n", result.TemperatureC)
	fmt.Printf("Generated: %s\n", result.GeneratedAt.Format(time.RFC3339))
	fmt.Printf("Revision: %s\n", result.Revision)
}

func loadDotEnv() error {
	for _, path := range []string{".env", "cli/.env"} {
		if _, err := os.Stat(path); err == nil {
			return godotenv.Load(path)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func buildEndpoint(rawBaseURL string, lat, lng float64) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawBaseURL))
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("base URL must include scheme and host")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/api/temperature"
	query := parsed.Query()
	query.Set("lat", fmt.Sprintf("%.6f", lat))
	query.Set("lng", fmt.Sprintf("%.6f", lng))
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func validateTemperatureResponse(result temperatureResponse) error {
	if result.GeneratedAt.IsZero() {
		return fmt.Errorf("missing generatedAt")
	}
	if result.Revision == "" {
		return fmt.Errorf("missing revision")
	}
	if result.Latitude < -90 || result.Latitude > 90 {
		return fmt.Errorf("latitude out of range: %.4f", result.Latitude)
	}
	if result.Longitude < -180 || result.Longitude > 180 {
		return fmt.Errorf("longitude out of range: %.4f", result.Longitude)
	}
	return nil
}
