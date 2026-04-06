package main

import (
	"fmt"
	"time"

	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/budget"
	"github.com/failsafe-go/failsafe-go/fallback"
	"github.com/failsafe-go/failsafe-go/hedgepolicy"
	"github.com/failsafe-go/failsafe-go/timeout"
)

func demoHedgeTimeoutAsync() {
	section("Timeout + hedging + budget + async")

	hedgeBudget := budget.NewBuilder().WithMaxRate(1.0).WithMinConcurrency(0).Build()

	hedgePolicy := hedgepolicy.NewBuilderWithDelay[probeResult](30 * time.Millisecond).
		WithMaxHedges(1).
		WithBudget(hedgeBudget).
		OnHedge(func(event failsafe.ExecutionEvent[probeResult]) {
			fmt.Printf("  hedge launched: attempts=%d hedges=%d\n", event.Attempts(), event.Hedges())
		}).
		Build()

	timeoutPolicy := timeout.NewBuilder[probeResult](180 * time.Millisecond).
		OnTimeoutExceeded(func(event failsafe.ExecutionDoneEvent[probeResult]) {
			fmt.Printf("  timeout fired after %s\n", event.ElapsedTime().Round(time.Millisecond))
		}).
		Build()

	asyncExecutor := failsafe.With(timeoutPolicy, hedgePolicy).
		OnDone(func(event failsafe.ExecutionDoneEvent[probeResult]) {
			fmt.Printf("  replica probe completed in %s\n", event.ElapsedTime().Round(time.Millisecond))
		})

	query := &replicaQuery{
		delays:   []time.Duration{200 * time.Millisecond, 50 * time.Millisecond, 40 * time.Millisecond},
		nodes:    []string{"edge-eu-west", "edge-us-east", "edge-ap-south"},
		scenario: "read-model probe",
	}

	asyncResult := asyncExecutor.GetAsyncWithExecution(query.Fetch)
	<-asyncResult.Done()
	reading, err := asyncResult.Get()
	if err != nil {
		fmt.Printf("  probe error: %v\n", err)
	} else {
		fmt.Printf("  winner: node=%s source=%s latency=%s note=%s\n", reading.Node, reading.Source, reading.Latency, reading.Note)
	}

	guardedBudget := budget.NewBuilder().
		WithMaxRate(0.4).
		WithMinConcurrency(0).
		OnBudgetExceeded(func(event budget.ExceededEvent) {
			fmt.Printf("  hedge budget blocked another %s\n", event.ExecutionType)
		}).
		Build()

	guardedFallback := fallback.NewBuilderWithFunc(func(exec failsafe.Execution[probeResult]) (probeResult, error) {
		return probeResult{
			Node:    "control-plane-cache",
			Source:  "fallback store",
			Latency: 30 * time.Second,
			Note:    fmt.Sprintf("hedges capped: %v", exec.LastError()),
		}, nil
	}).
		HandleErrors(budget.ErrExceeded).
		OnFallbackExecuted(func(event failsafe.ExecutionDoneEvent[probeResult]) {
			fmt.Println("  budget fallback preserved backend capacity")
		}).
		Build()

	guardedHedge := hedgepolicy.NewBuilderWithDelay[probeResult](30 * time.Millisecond).
		WithMaxHedges(2).
		WithBudget(guardedBudget).
		OnHedge(func(event failsafe.ExecutionEvent[probeResult]) {
			fmt.Printf("  guarded hedge launched: attempts=%d hedges=%d\n", event.Attempts(), event.Hedges())
		}).
		Build()

	guardedQuery := &replicaQuery{
		delays:   []time.Duration{200 * time.Millisecond, 90 * time.Millisecond, 40 * time.Millisecond},
		nodes:    []string{"edge-primary", "edge-secondary", "edge-tertiary"},
		scenario: "budget-guarded read",
	}

	reading, err = failsafe.With(guardedFallback, guardedHedge).GetWithExecution(guardedQuery.Fetch)
	if err != nil {
		fmt.Printf("  guarded hedge failed: %v\n", err)
	} else {
		fmt.Printf("  guarded hedge result: node=%s note=%s\n", reading.Node, reading.Note)
	}

	timeoutFallback := fallback.NewBuilderWithFunc(func(exec failsafe.Execution[probeResult]) (probeResult, error) {
		return probeResult{
			Node:    "archive-store",
			Source:  "fallback store",
			Latency: 5 * time.Minute,
			Note:    fmt.Sprintf("served because %v", exec.LastError()),
		}, nil
	}).
		HandleErrors(timeout.ErrExceeded).
		OnFallbackExecuted(func(event failsafe.ExecutionDoneEvent[probeResult]) {
			fmt.Println("  timeout fallback served archival data")
		}).
		Build()

	slowExecutor := failsafe.With(timeoutFallback, timeout.NewBuilder[probeResult](60*time.Millisecond).
		OnTimeoutExceeded(func(event failsafe.ExecutionDoneEvent[probeResult]) {
			fmt.Println("  strict timeout cut off the slow archive probe")
		}).
		Build())

	slowQuery := &replicaQuery{
		delays:   []time.Duration{200 * time.Millisecond},
		nodes:    []string{"archive-cold-path"},
		scenario: "cold-path read",
	}

	reading, err = slowExecutor.GetWithExecution(slowQuery.Fetch)
	if err != nil {
		fmt.Printf("  timeout path failed: %v\n", err)
	} else {
		fmt.Printf("  timeout path result: node=%s note=%s\n", reading.Node, reading.Note)
	}
}
