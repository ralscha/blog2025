package main

import (
	"context"
	"fmt"

	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/cachepolicy"
)

func demoCachePolicy() {
	section("Cache policy + context keys")

	cache := newMemoryCache[configSnapshot]()
	controlPlaneCalls := 0

	cacheExecutor := failsafe.With(cachepolicy.NewBuilder[configSnapshot](cache).
		CacheIf(func(result configSnapshot, err error) bool {
			return err == nil && len(result.Assets) > 0
		}).
		OnCacheMiss(func(event failsafe.ExecutionEvent[configSnapshot]) {
			fmt.Printf("  cache miss on attempt %d\n", event.Attempts())
		}).
		OnResultCached(func(event failsafe.ExecutionEvent[configSnapshot]) {
			fmt.Printf("  cached snapshot from %s\n", event.LastResult().Source)
		}).
		OnCacheHit(func(event failsafe.ExecutionDoneEvent[configSnapshot]) {
			fmt.Printf("  cache hit for %s\n", event.Result.Service)
		}).
		Build())

	ctx := cachepolicy.ContextWithCacheKey(context.Background(), "snapshot:checkout-api")
	loader := func(exec failsafe.Execution[configSnapshot]) (configSnapshot, error) {
		controlPlaneCalls++
		return configSnapshot{
			Service: "checkout-api",
			Assets:  []string{"feature-flags", "routing-rules", "slo-budgets"},
			Source:  fmt.Sprintf("control-plane call %d", controlPlaneCalls),
		}, nil
	}

	first, err := cacheExecutor.WithContext(ctx).GetWithExecution(loader)
	if err != nil {
		fmt.Printf("  first snapshot error: %v\n", err)
	} else {
		fmt.Printf("  first snapshot source: %s\n", first.Source)
	}

	second, err := cacheExecutor.WithContext(ctx).GetWithExecution(loader)
	if err != nil {
		fmt.Printf("  second snapshot error: %v\n", err)
	} else {
		fmt.Printf("  second snapshot source: %s\n", second.Source)
	}

	fmt.Printf("  supplier was called %d time(s)\n", controlPlaneCalls)
}
