package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	errTransient   = errors.New("transient dependency error")
	errPermanent   = errors.New("permanent dependency error")
	errCircuitOpen = errors.New("circuit open")
)

type task struct {
	name string
	d    time.Duration
	fail bool
}

type fetchResult struct {
	taskName string
	data     string
	err      error
}

func sampleTasks() []task {
	return []task{
		{name: "profile", d: 80 * time.Millisecond, fail: false},
		{name: "orders", d: 120 * time.Millisecond, fail: true},
		{name: "inventory", d: 200 * time.Millisecond, fail: false},
	}
}

func sampleTasksMultipleFailures() []task {
	return []task{
		{name: "profile", d: 80 * time.Millisecond, fail: true},
		{name: "orders", d: 120 * time.Millisecond, fail: true},
		{name: "inventory", d: 200 * time.Millisecond, fail: true},
	}
}

func runTaskWithCtx(ctx context.Context, t task) error {
	fmt.Printf("running %s...\n", t.name)
	select {
	case <-time.After(t.d):
		if t.fail {
			fmt.Printf("%s failed\n", t.name)
			return fmt.Errorf("%s failed", t.name)
		}
		fmt.Printf("%s succeeded\n", t.name)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func runTaskWithResult(t task) (string, error) {
	fmt.Printf("running %s...\n", t.name)
	time.Sleep(t.d)
	if t.fail {
		fmt.Printf("%s failed\n", t.name)
		return "", fmt.Errorf("%s failed", t.name)
	}
	fmt.Printf("%s succeeded\n", t.name)
	return fmt.Sprintf("data for %s", t.name), nil
}

func runTaskWithResultCtx(ctx context.Context, t task) (string, error) {
	fmt.Printf("running %s...\n", t.name)
	select {
	case <-time.After(t.d):
		if t.fail {
			fmt.Printf("%s failed\n", t.name)
			return "", fmt.Errorf("%s failed", t.name)
		}
		fmt.Printf("%s succeeded\n", t.name)
		return fmt.Sprintf("data for %s", t.name), nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func runTask(t task) error {
	fmt.Printf("running %s...\n", t.name)
	time.Sleep(t.d)
	if t.fail {
		fmt.Printf("%s failed\n", t.name)
		return fmt.Errorf("%s failed", t.name)
	}
	fmt.Printf("%s succeeded\n", t.name)
	return nil
}
