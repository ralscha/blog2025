package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type httpTraceLogger struct {
	mu   sync.Mutex
	file *os.File
}

type httpTraceEntry struct {
	Timestamp string         `json:"timestamp"`
	Name      string         `json:"name"`
	Request   traceRequest   `json:"request"`
	Response  *traceResponse `json:"response,omitempty"`
	Error     string         `json:"error,omitempty"`
}

type traceRequest struct {
	Method           string              `json:"method"`
	URL              string              `json:"url"`
	Headers          map[string]string   `json:"headers,omitempty"`
	Form             map[string][]string `json:"form,omitempty"`
	ClientID         string              `json:"clientId,omitempty"`
	ClientAuthMethod string              `json:"clientAuthMethod,omitempty"`
	TokenTypeHint    string              `json:"tokenTypeHint,omitempty"`
	Notes            string              `json:"notes,omitempty"`
}

type traceResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       any               `json:"body,omitempty"`
}

func newHTTPTraceLoggerFromEnv() (*httpTraceLogger, error) {
	path := os.Getenv("OIDC_BFF_TRACE_FILE")
	if path == "" {
		return nil, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}

	return &httpTraceLogger{file: file}, nil
}

func (l *httpTraceLogger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

func (l *httpTraceLogger) Write(name string, request traceRequest, response *traceResponse, err error) {
	if l == nil || l.file == nil {
		return
	}

	entry := httpTraceEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Name:      name,
		Request:   request,
		Response:  response,
	}
	if err != nil {
		entry.Error = err.Error()
	}

	encoded, marshalErr := json.Marshal(entry)
	if marshalErr != nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.file.Write(append(encoded, '\n'))
}
