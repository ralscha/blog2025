package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/failsafehttp"
)

func demoHTTPAdapter() {
	section("HTTP adapter")

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := attempts.Add(1)
		if attempt <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("control plane warming up"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("configuration applied"))
	}))
	defer server.Close()

	retryPolicy := failsafehttp.NewRetryPolicyBuilder().
		WithBackoff(20*time.Millisecond, 80*time.Millisecond).
		OnRetryScheduled(func(event failsafe.ExecutionScheduledEvent[*http.Response]) {
			status := 0
			if resp := event.LastResult(); resp != nil {
				status = resp.StatusCode
			}
			fmt.Printf("  retrying outbound request: status=%d next-attempt=%d delay=%s\n", status, event.Attempts()+1, event.Delay)
		}).
		Build()

	client := &http.Client{
		Transport: failsafehttp.NewRoundTripper(nil, retryPolicy),
	}

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		fmt.Printf("  request build failed: %v\n", err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("  outbound request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("  response read failed: %v\n", err)
		return
	}

	fmt.Printf("  final HTTP status=%d body=%q attempts=%d\n", resp.StatusCode, string(body), attempts.Load())
}
