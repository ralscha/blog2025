package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	realtimeTopic  = "iss-position"
	realtimeEvent  = "iss-update"
	issPositionURL = "https://api.wheretheiss.at/v1/satellites/25544"
	interval       = 10 * time.Second
	timeout        = 8 * time.Second
)

type issPositionResponse struct {
	Timestamp int64   `json:"timestamp"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type broadcastPayload struct {
	Source      string  `json:"source"`
	RequestedAt string  `json:"requestedAt"`
	Timestamp   int64   `json:"timestamp"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

type broadcastMessage struct {
	Topic   string           `json:"topic"`
	Event   string           `json:"event"`
	Private bool             `json:"private"`
	Payload broadcastPayload `json:"payload"`
}

type broadcastRequest struct {
	Messages []broadcastMessage `json:"messages"`
}

func main() {
	logger := log.New(os.Stdout, "iss-broadcast: ", log.LstdFlags)

	supabaseURL, err := envOrError("SUPABASE_URL")
	if err != nil {
		logger.Fatal(err)
	}

	anonKey, err := envOrError("VITE_SUPABASE_ANON_KEY")
	if err != nil {
		logger.Fatal(err)
	}

	broadcastURL, err := buildBroadcastURL(supabaseURL)
	if err != nil {
		logger.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := &http.Client{}

	if err := publishOnce(ctx, client, broadcastURL, anonKey, logger); err != nil {
		logger.Printf("initial publish failed: %v", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Printf("publishing ISS updates to %s every %s", broadcastURL, interval)

	for {
		select {
		case <-ctx.Done():
			logger.Println("shutting down")
			return
		case <-ticker.C:
			if err := publishOnce(ctx, client, broadcastURL, anonKey, logger); err != nil {
				logger.Printf("publish failed: %v", err)
			}
		}
	}
}

func publishOnce(parent context.Context, client *http.Client, broadcastURL, anonKey string, logger *log.Logger) error {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	payload, err := fetchISSPosition(ctx, client)
	if err != nil {
		return err
	}

	if err := sendBroadcast(ctx, client, broadcastURL, anonKey, payload); err != nil {
		return err
	}

	logger.Printf(
		"broadcast sent lat=%.4f lon=%.4f timestamp=%d",
		payload.Latitude,
		payload.Longitude,
		payload.Timestamp,
	)

	return nil
}

func fetchISSPosition(ctx context.Context, client *http.Client) (broadcastPayload, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, issPositionURL, nil)
	if err != nil {
		return broadcastPayload{}, fmt.Errorf("build ISS request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return broadcastPayload{}, fmt.Errorf("fetch ISS position: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return broadcastPayload{}, fmt.Errorf("ISS API returned status %d", response.StatusCode)
	}

	var issData issPositionResponse
	if err := json.NewDecoder(response.Body).Decode(&issData); err != nil {
		return broadcastPayload{}, fmt.Errorf("decode ISS response: %w", err)
	}

	if issData.Timestamp <= 0 || issData.Latitude < -90 || issData.Latitude > 90 || issData.Longitude < -180 || issData.Longitude > 180 {
		return broadcastPayload{}, errors.New("ISS API returned invalid position data")
	}

	return broadcastPayload{
		Source:      "wheretheiss.at",
		RequestedAt: time.Now().UTC().Format(time.RFC3339),
		Timestamp:   issData.Timestamp,
		Latitude:    issData.Latitude,
		Longitude:   issData.Longitude,
	}, nil
}

func sendBroadcast(ctx context.Context, client *http.Client, broadcastURL, anonKey string, payload broadcastPayload) error {
	requestBody := broadcastRequest{
		Messages: []broadcastMessage{
			{
				Topic:   realtimeTopic,
				Event:   realtimeEvent,
				Private: false,
				Payload: payload,
			},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("encode broadcast payload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, broadcastURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build broadcast request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("apikey", anonKey)
	request.Header.Set("Authorization", "Bearer "+anonKey)

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("send broadcast: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		var details bytes.Buffer
		_, _ = details.ReadFrom(response.Body)
		return fmt.Errorf("broadcast endpoint returned status %d: %s", response.StatusCode, strings.TrimSpace(details.String()))
	}

	return nil
}

func buildBroadcastURL(rawSupabaseURL string) (string, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(rawSupabaseURL), "/")
	if trimmed == "" {
		return "", errors.New("SUPABASE_URL is empty")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("parse SUPABASE_URL: %w", err)
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("SUPABASE_URL must be an absolute URL")
	}

	return trimmed + "/realtime/v1/api/broadcast", nil
}

func envOrError(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("missing required environment variable %s", key)
	}

	return value, nil
}
