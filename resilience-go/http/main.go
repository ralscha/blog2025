package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/failsafe-go/failsafe-go/failsafehttp"
	"github.com/failsafe-go/failsafe-go/timeout"
)

func main() {
	timeoutPolicy := timeout.NewBuilder[*http.Response](150 * time.Millisecond).Build()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(250 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("inventory refreshed"))
		case <-r.Context().Done():
			fmt.Printf("handler canceled: %v\n", r.Context().Err())
		}
	})

	protected := failsafehttp.NewHandler(handler, timeoutPolicy)
	server := httptest.NewServer(protected)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		fmt.Printf("request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("read failed: %v\n", err)
		return
	}

	fmt.Printf("status=%d body=%q\n", resp.StatusCode, strings.TrimSpace(string(body)))
}
