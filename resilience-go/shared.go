package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/failsafe-go/failsafe-go"
)

var (
	errUpstreamUnavailable = errors.New("upstream unavailable")
	errInvalidConfig       = errors.New("invalid control-plane config")
)

type rolloutPlan struct {
	Service string
	Region  string
	Source  string
	Note    string
}

type probeResult struct {
	Node    string
	Source  string
	Latency time.Duration
	Note    string
}

type configSnapshot struct {
	Service string
	Assets  []string
	Source  string
}

type planStep struct {
	delay time.Duration
	err   error
	plan  rolloutPlan
}

type scriptedPlanner struct {
	mu    sync.Mutex
	steps []planStep
}

func (p *scriptedPlanner) Fetch(exec failsafe.Execution[rolloutPlan]) (rolloutPlan, error) {
	p.mu.Lock()
	var step planStep
	if len(p.steps) > 0 {
		step = p.steps[0]
		p.steps = p.steps[1:]
	}
	p.mu.Unlock()

	if step.delay > 0 {
		select {
		case <-time.After(step.delay):
		case <-exec.Canceled():
			return rolloutPlan{}, exec.Context().Err()
		}
	}

	if step.err != nil {
		return rolloutPlan{}, step.err
	}

	if step.plan.Source == "" {
		step.plan.Source = fmt.Sprintf("attempt-%d", exec.Attempts())
	}
	return step.plan, nil
}

type replicaQuery struct {
	mu       sync.Mutex
	delays   []time.Duration
	nodes    []string
	scenario string
}

func (q *replicaQuery) Fetch(exec failsafe.Execution[probeResult]) (probeResult, error) {
	q.mu.Lock()
	index := max(exec.Attempts()-1, 0)
	if index >= len(q.delays) {
		index = len(q.delays) - 1
	}
	delay := q.delays[index]
	node := q.nodes[index]
	q.mu.Unlock()

	note := "primary"
	if exec.IsHedge() {
		note = fmt.Sprintf("hedge-%d", exec.Hedges())
	}

	select {
	case <-time.After(delay):
		return probeResult{
			Node:    node,
			Source:  q.scenario,
			Latency: delay,
			Note:    note,
		}, nil
	case <-exec.Canceled():
		return probeResult{}, exec.Context().Err()
	}
}

type memoryCache[R any] struct {
	mu     sync.Mutex
	values map[string]R
}

func newMemoryCache[R any]() *memoryCache[R] {
	return &memoryCache[R]{values: make(map[string]R)}
}

func (c *memoryCache[R]) Get(key string) (R, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	value, ok := c.values[key]
	return value, ok
}

func (c *memoryCache[R]) Set(key string, value R) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

func section(title string) {
	fmt.Printf("\n== %s ==\n", title)
}

func printRolloutResult(label string, plan rolloutPlan, err error) {
	if err != nil {
		fmt.Printf("%s: error=%v\n", label, err)
		return
	}
	if plan.Note == "" {
		fmt.Printf("%s: service=%s region=%s source=%s\n", label, plan.Service, plan.Region, plan.Source)
		return
	}
	fmt.Printf("%s: service=%s region=%s source=%s note=%s\n", label, plan.Service, plan.Region, plan.Source, plan.Note)
}

func ctxWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
