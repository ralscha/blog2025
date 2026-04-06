package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/failsafe-go/failsafe-go/fallback"
	"github.com/failsafe-go/failsafe-go/retrypolicy"
)

func demoRetryCircuitFallback() {
	section("Retry + circuit breaker + fallback")

	breaker := circuitbreaker.NewBuilder[rolloutPlan]().
		HandleErrors(errUpstreamUnavailable).
		WithFailureThreshold(2).
		WithSuccessThreshold(1).
		WithDelay(120 * time.Millisecond).
		OnStateChanged(func(event circuitbreaker.StateChangedEvent) {
			fmt.Printf("  breaker state: %s -> %s\n", event.OldState, event.NewState)
		}).
		OnOpen(func(event circuitbreaker.StateChangedEvent) {
			fmt.Printf("  breaker opened after %d upstream failures\n", event.Metrics().Failures())
		}).
		OnHalfOpen(func(event circuitbreaker.StateChangedEvent) {
			fmt.Println("  breaker is probing the planner again")
		}).
		OnClose(func(event circuitbreaker.StateChangedEvent) {
			fmt.Println("  breaker closed after a healthy probe")
		}).
		Build()

	retryPolicy := retrypolicy.NewBuilder[rolloutPlan]().
		HandleIf(func(_ rolloutPlan, err error) bool {
			return err != nil
		}).
		AbortOnErrors(errInvalidConfig, circuitbreaker.ErrOpen).
		WithMaxAttempts(3).
		WithBackoff(20*time.Millisecond, 80*time.Millisecond).
		WithJitter(5 * time.Millisecond).
		OnRetryScheduled(func(event failsafe.ExecutionScheduledEvent[rolloutPlan]) {
			fmt.Printf("  retry scheduled: attempt=%d delay=%s\n", event.Attempts()+1, event.Delay)
		}).
		OnRetry(func(event failsafe.ExecutionEvent[rolloutPlan]) {
			fmt.Printf("  retrying after: %v\n", event.LastError())
		}).
		OnAbort(func(event failsafe.ExecutionEvent[rolloutPlan]) {
			fmt.Printf("  retry aborted on: %v\n", event.LastError())
		}).
		Build()

	fallbackPolicy := fallback.NewBuilderWithFunc(func(exec failsafe.Execution[rolloutPlan]) (rolloutPlan, error) {
		note := "served a cached rollout plan after retries were exhausted"
		if errors.Is(exec.LastError(), circuitbreaker.ErrOpen) {
			note = "served a cached rollout plan because the breaker is open"
		}
		return rolloutPlan{
			Service: "checkout-api",
			Region:  "us-east-1",
			Source:  "degraded-cache",
			Note:    note,
		}, nil
	}).
		HandleErrors(retrypolicy.ErrExceeded, circuitbreaker.ErrOpen).
		OnFallbackExecuted(func(event failsafe.ExecutionDoneEvent[rolloutPlan]) {
			fmt.Printf("  fallback served: %s\n", event.Result.Source)
		}).
		Build()

	ctx, cancel := ctxWithTimeout(time.Second)
	defer cancel()

	executor := failsafe.With(fallbackPolicy, retryPolicy, breaker).
		WithContext(ctx).
		OnDone(func(event failsafe.ExecutionDoneEvent[rolloutPlan]) {
			if event.Error != nil {
				fmt.Printf("  done in %s after %d attempts with error=%v\n", event.ElapsedTime().Round(time.Millisecond), event.Attempts(), event.Error)
				return
			}
			fmt.Printf("  done in %s after %d attempts with source=%s\n", event.ElapsedTime().Round(time.Millisecond), event.Attempts(), event.Result.Source)
		})

	first := &scriptedPlanner{steps: []planStep{
		{err: errUpstreamUnavailable},
		{plan: rolloutPlan{Service: "checkout-api", Region: "us-east-1", Source: "planner-v2", Note: "recovered after one retry"}},
	}}
	plan, err := executor.GetWithExecution(first.Fetch)
	printRolloutResult("  initial rollout", plan, err)

	second := &scriptedPlanner{steps: []planStep{
		{err: errUpstreamUnavailable},
		{err: errUpstreamUnavailable},
	}}
	plan, err = executor.GetWithExecution(second.Fetch)
	printRolloutResult("  sustained outage", plan, err)

	third := &scriptedPlanner{steps: []planStep{{plan: rolloutPlan{Service: "checkout-api", Region: "us-east-1", Source: "planner-v2"}}}}
	plan, err = executor.GetWithExecution(third.Fetch)
	printRolloutResult("  immediate follow-up", plan, err)

	time.Sleep(140 * time.Millisecond)

	fourth := &scriptedPlanner{steps: []planStep{
		{plan: rolloutPlan{Service: "checkout-api", Region: "us-east-1", Source: "planner-v2", Note: "half-open probe succeeded"}},
	}}
	plan, err = executor.GetWithExecution(fourth.Fetch)
	printRolloutResult("  recovery rollout", plan, err)

	fatal := &scriptedPlanner{steps: []planStep{{err: errInvalidConfig}}}
	_, err = executor.GetWithExecution(fatal.Fetch)
	fmt.Printf("  invalid rollout rejected: %v\n", err)
}
