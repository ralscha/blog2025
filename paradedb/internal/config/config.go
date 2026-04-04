package config

import (
	"os"
	"path/filepath"
)

const (
	defaultDatabaseURL  = "postgres://postgres:postgres@localhost:5432/paradedb_demo?sslmode=disable"
	defaultDownloadsDir = "data/downloads"
	defaultTopicsPath   = "data/manifest/gutenberg-topics.txt"
)

func DatabaseURL() string {
	return envOrDefault("DATABASE_URL", defaultDatabaseURL)
}

func DownloadsDir() string {
	return filepath.Clean(envOrDefault("PARADEDB_DOWNLOADS_DIR", defaultDownloadsDir))
}

func TopicsPath() string {
	return filepath.Clean(envOrDefault("PARADEDB_TOPICS_FILE", defaultTopicsPath))
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
