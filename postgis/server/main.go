package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"starbucks/internal/demo"
)

type apiServer struct {
	pool      *pgxpool.Pool
	staticDir string
	mux       *http.ServeMux
}

type nearbyResponse struct {
	Stores []demo.NearbyStore `json:"stores"`
}

type clustersResponse struct {
	CountryCode string               `json:"countryCode"`
	Clusters    []demo.ClusterResult `json:"clusters"`
}

type corridorResponse struct {
	Stores []demo.CorridorStore `json:"stores"`
}

type countriesResponse struct {
	Countries []demo.CountrySummary `json:"countries"`
}

type geofencesResponse struct {
	Geofences []demo.GeofenceArea `json:"geofences"`
}

type geofenceLiveResponse struct {
	Geofence   demo.GeofenceArea          `json:"geofence"`
	Trucks     []demo.TruckGeofenceStatus `json:"trucks"`
	Events     []demo.TruckGeofenceEvent  `json:"events"`
	ServerTime time.Time                  `json:"serverTime"`
}

func main() {
	var addr string
	var staticDir string

	flag.StringVar(&addr, "addr", ":8080", "HTTP listen address")
	flag.StringVar(&staticDir, "static", "web/dist", "directory with Vite build output")
	flag.Parse()

	ctx := context.Background()
	pool, err := demo.Open(ctx)
	check(err)
	defer pool.Close()

	check(demo.EnsureSchema(ctx, pool))
	check(demo.EnsureGeofencingSchema(ctx, pool))
	check(demo.SeedHomeDepotGeofence(ctx, pool))

	api := newAPIServer(pool, staticDir)
	log.Printf("listening on http://localhost%s", addr)
	check(http.ListenAndServe(addr, api.routes()))
}

func newAPIServer(pool *pgxpool.Pool, staticDir string) *apiServer {
	server := &apiServer{
		pool:      pool,
		staticDir: staticDir,
		mux:       http.NewServeMux(),
	}

	server.mux.HandleFunc("/api/countries", server.handleCountries)
	server.mux.HandleFunc("/api/nearby", server.handleNearby)
	server.mux.HandleFunc("/api/clusters", server.handleClusters)
	server.mux.HandleFunc("/api/corridor", server.handleCorridor)
	server.mux.HandleFunc("/api/geofences", server.handleGeofences)
	server.mux.HandleFunc("/api/geofence/live", server.handleGeofenceLive)

	if dirExists(staticDir) {
		server.mux.Handle("/", spaHandler(staticDir))
	} else {
		server.mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte("Starbucks PostGIS API is running. Build the Vite app in web/ to serve the UI here."))
		})
	}

	return server
}

func (s *apiServer) routes() http.Handler {
	return loggingMiddleware(s.mux)
}

func (s *apiServer) handleCountries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	limit := int32(queryInt(r, "limit", 30))
	countries, err := demo.ListCountries(r.Context(), s.pool, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, countriesResponse{Countries: countries})
}

func (s *apiServer) handleNearby(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	latitude, err := queryFloatRequired(r, "lat")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	longitude, err := queryFloatRequired(r, "lon")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	radius := queryFloat(r, "radius", 1500)
	limit := int32(queryInt(r, "limit", 12))

	stores, err := demo.Nearby(r.Context(), s.pool, latitude, longitude, radius, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, nearbyResponse{Stores: stores})
}

func (s *apiServer) handleClusters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	countryCode := strings.TrimSpace(r.URL.Query().Get("country"))
	if countryCode == "" {
		writeError(w, http.StatusBadRequest, errors.New("missing country query parameter"))
		return
	}
	k := int32(queryInt(r, "k", 5))

	clusters, err := demo.ClusterByCountry(r.Context(), s.pool, countryCode, k)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, clustersResponse{CountryCode: countryCode, Clusters: clusters})
}

func (s *apiServer) handleCorridor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	fromLatitude, err := queryFloatRequired(r, "fromLat")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	fromLongitude, err := queryFloatRequired(r, "fromLon")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	toLatitude, err := queryFloatRequired(r, "toLat")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	toLongitude, err := queryFloatRequired(r, "toLon")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	distance := queryFloat(r, "distance", 600)
	limit := int32(queryInt(r, "limit", 15))

	stores, err := demo.StoresAlongCorridor(
		r.Context(),
		s.pool,
		fromLatitude,
		fromLongitude,
		toLatitude,
		toLongitude,
		distance,
		limit,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, corridorResponse{Stores: stores})
}

func (s *apiServer) handleGeofences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	geofences, err := demo.ListGeofences(r.Context(), s.pool)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, geofencesResponse{Geofences: geofences})
}

func (s *apiServer) handleGeofenceLive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	geofenceID := strings.TrimSpace(r.URL.Query().Get("geofence"))
	if geofenceID == "" {
		geofenceID = demo.HomeDepotGeofenceID
	}

	eventLimit := int32(queryInt(r, "eventLimit", 12))

	geofence, err := demo.GetGeofence(r.Context(), s.pool, geofenceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	trucks, err := demo.ListTruckGeofenceStatuses(r.Context(), s.pool, geofenceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	events, err := demo.ListRecentTruckGeofenceEvents(r.Context(), s.pool, geofenceID, eventLimit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, geofenceLiveResponse{
		Geofence:   geofence,
		Trucks:     trucks,
		Events:     events,
		ServerTime: time.Now().UTC(),
	})
}

func spaHandler(staticDir string) http.Handler {
	fileServer := http.FileServer(http.Dir(staticDir))
	indexPath := filepath.Join(staticDir, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		requestedPath := filepath.Join(staticDir, filepath.Clean(r.URL.Path))
		if r.URL.Path == "/" || (!hasFileExtension(r.URL.Path) && !fileExists(requestedPath)) {
			http.ServeFile(w, r, indexPath)
			return
		}

		fileServer.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func queryFloatRequired(r *http.Request, key string) (float64, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return 0, fmt.Errorf("missing %s query parameter", key)
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s query parameter", key)
	}
	return parsed, nil
}

func queryFloat(r *http.Request, key string, fallback float64) float64 {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func queryInt(r *http.Request, key string, fallback int) int {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func hasFileExtension(path string) bool {
	return filepath.Ext(path) != ""
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
