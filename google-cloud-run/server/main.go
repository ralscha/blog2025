package main

import (
	"encoding/json"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type temperatureResponse struct {
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	TemperatureC int       `json:"temperatureC"`
	GeneratedAt  time.Time `json:"generatedAt"`
	Revision     string    `json:"revision"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type requestError struct {
	message string
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/api/temperature", handleTemperature)

	addr := ":" + portFromEnv()
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func handleIndex(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":        "random-temperature-service",
		"description": "Returns a random temperature for a latitude/longitude.",
		"endpoint":    "/api/temperature?lat=37.7749&lng=-122.4194",
	})
}
func handleTemperature(w http.ResponseWriter, r *http.Request) {
	lat, err := parseCoordinate(r, "lat", -90, 90)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	lng, err := parseCoordinate(r, "lng", -180, 180)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, temperatureResponse{
		Latitude:     lat,
		Longitude:    lng,
		TemperatureC: randomTemperature(lat),
		GeneratedAt:  time.Now().UTC(),
		Revision:     revision(),
	})
}
func parseCoordinate(r *http.Request, key string, minValue, maxValue float64) (float64, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return 0, &requestError{message: key + " is required"}
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, &requestError{message: key + " must be a number"}
	}
	if parsed < minValue || parsed > maxValue {
		return 0, &requestError{message: key + " is out of range"}
	}
	return parsed, nil
}

func randomTemperature(latitude float64) int {
	latitudeFactor := math.Abs(latitude) / 90.0
	maxTemp := int(math.Round(38 - (latitudeFactor * 43)))
	minTemp := min(int(math.Round(24-(latitudeFactor*49))), maxTemp)
	return rand.IntN(maxTemp-minTemp+1) + minTemp
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func portFromEnv() string {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		return "8080"
	}
	if _, err := strconv.Atoi(port); err != nil {
		return "8080"
	}
	return port
}

func revision() string {
	value := strings.TrimSpace(os.Getenv("K_REVISION"))
	if value == "" {
		return "local"
	}
	return value
}

func (e *requestError) Error() string {
	return e.message
}
