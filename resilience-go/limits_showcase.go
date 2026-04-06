package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/adaptivelimiter"
	"github.com/failsafe-go/failsafe-go/adaptivethrottler"
	"github.com/failsafe-go/failsafe-go/bulkhead"
	"github.com/failsafe-go/failsafe-go/ratelimiter"
)

func demoRateLimiter() {
	section("Rate limiter")

	limiter := ratelimiter.NewBurstyBuilder[string](2, 120*time.Millisecond).
		OnRateLimitExceeded(func(event failsafe.ExecutionEvent[string]) {
			fmt.Printf("  rate limited at attempt %d\n", event.Attempts())
		}).
		Build()

	executor := failsafe.With(limiter)
	for attempt := 1; attempt <= 3; attempt++ {
		result, err := executor.Get(func() (string, error) {
			return fmt.Sprintf("request-%d", attempt), nil
		})
		if err != nil {
			fmt.Printf("  request %d rejected: %v\n", attempt, err)
			continue
		}
		fmt.Printf("  request %d accepted with %s\n", attempt, result)
	}

	time.Sleep(130 * time.Millisecond)
	result, err := executor.Get(func() (string, error) {
		return "request-after-refill", nil
	})
	if err != nil {
		fmt.Printf("  refill request failed: %v\n", err)
	} else {
		fmt.Printf("  refill request accepted with %s\n", result)
	}
}

func demoBulkhead() {
	section("Bulkhead")

	gate := bulkhead.NewBuilder[string](1).
		OnFull(func(event failsafe.ExecutionEvent[string]) {
			fmt.Println("  bulkhead is full")
		}).
		Build()

	if err := gate.AcquirePermit(context.Background()); err != nil {
		fmt.Printf("  failed to prefill bulkhead: %v\n", err)
		return
	}

	_, err := failsafe.With(gate).Get(func() (string, error) {
		return "worker-slot-1", nil
	})
	fmt.Printf("  while saturated: %v\n", err)

	gate.ReleasePermit()

	result, err := failsafe.With(gate).Get(func() (string, error) {
		return "worker-slot-1", nil
	})
	if err != nil {
		fmt.Printf("  recovered bulkhead failed: %v\n", err)
	} else {
		fmt.Printf("  recovered bulkhead accepted %s\n", result)
	}
}

func demoAdaptiveLimiter() {
	section("Adaptive limiter")

	limiter := adaptivelimiter.NewBuilder[string]().
		WithLimits(1, 3, 1).
		WithRecentWindow(time.Second, 2*time.Second, 50).
		OnLimitExceeded(func(event failsafe.ExecutionEvent[string]) {
			fmt.Printf("  adaptive limiter rejected attempt %d\n", event.Attempts())
		}).
		Build()

	heldPermit, err := limiter.AcquirePermit(context.Background())
	if err != nil {
		fmt.Printf("  failed to acquire warm-up permit: %v\n", err)
		return
	}

	fmt.Printf("  limit=%d inflight=%d queued=%d\n", limiter.Limit(), limiter.Inflight(), limiter.Queued())

	_, err = failsafe.With(limiter).Get(func() (string, error) {
		return "background-sync", nil
	})
	fmt.Printf("  saturated execution: %v\n", err)

	heldPermit.Drop()

	result, err := failsafe.With(limiter).Get(func() (string, error) {
		return "background-sync", nil
	})
	if err != nil {
		fmt.Printf("  post-release execution failed: %v\n", err)
	} else {
		fmt.Printf("  post-release execution accepted %s\n", result)
	}
}

func demoAdaptiveThrottler() {
	section("Adaptive throttler")

	throttler := adaptivethrottler.NewBuilder[int]().
		HandleResult(503).
		WithFailureRateThreshold(0.2, 1, time.Minute).
		WithMaxRejectionRate(1.0).
		Build()

	executor := failsafe.With(throttler)
	for attempt := 1; attempt <= 8; attempt++ {
		result, err := executor.Get(func() (int, error) {
			return 503, nil
		})
		if errors.Is(err, adaptivethrottler.ErrExceeded) {
			fmt.Printf("  attempt %d rejected with rejection rate %.2f\n", attempt, throttler.RejectionRate())
			return
		}
		fmt.Printf("  attempt %d recorded result %d, rejection rate %.2f\n", attempt, result, throttler.RejectionRate())
	}

	fmt.Printf("  throttler never rejected within the sample window; final rate %.2f\n", throttler.RejectionRate())
}
