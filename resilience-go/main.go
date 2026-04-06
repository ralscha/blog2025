package main

import "fmt"

func main() {
	fmt.Println("resilience-go-demo")
	fmt.Println("A compact, production-focused failsafe-go tour for a blog post.")

	demoRetryCircuitFallback()
	demoHedgeTimeoutAsync()
	demoHTTPAdapter()
	demoCachePolicy()
	demoRateLimiter()
	demoBulkhead()
	demoAdaptiveLimiter()
	demoAdaptiveThrottler()

	fmt.Println("\nDone.")
}
