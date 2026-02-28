package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/sync/errgroup"
)

func demoErrgroup(tasks []task) {
	var g errgroup.Group

	for _, currentTask := range tasks {
		g.Go(func() error {
			err := runTask(currentTask)
			return err
		})
	}

	if err := g.Wait(); err != nil {
		log.Printf("failed with first error: %v", err)
	}
}

func demoErrgroupWithContext(tasks []task) {
	baseCtx := context.Background()
	g, ctx := errgroup.WithContext(baseCtx)
	g.SetLimit(2)
	for _, currentTask := range tasks {
		g.Go(func() error {
			return runTaskWithCtx(ctx, currentTask)
		})
	}

	if err := g.Wait(); err != nil {
		log.Printf("failed with first error: %v", err)
	}
}

func demoErrgroupFailFastCollect(tasks []task) {
	baseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	g, ctx := errgroup.WithContext(baseCtx)
	g.SetLimit(2)

	errCh := make(chan error, len(tasks))

	for _, currentTask := range tasks {
		g.Go(func() error {
			err := runTaskWithCtx(ctx, currentTask)
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
			}
			return err
		})
	}

	firstErr := g.Wait()
	close(errCh)

	if firstErr != nil {
		fmt.Printf("failed with first error: %v\n", firstErr)
	}

	for err := range errCh {
		fmt.Println(err)
	}

}

func demoErrgroupCollectAll(tasks []task) {
	baseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var g errgroup.Group
	g.SetLimit(2)

	errCh := make(chan error, len(tasks))

	for _, currentTask := range tasks {
		g.Go(func() error {
			err := runTaskWithCtx(baseCtx, currentTask)
			if err != nil {
				errCh <- err
			}
			return nil
		})
	}

	_ = g.Wait()
	close(errCh)

	var collected []error
	for err := range errCh {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			collected = append(collected, err)
		}
	}

	if len(collected) > 0 {
		log.Printf("collected %d error(s): %v", len(collected), errors.Join(collected...))
		return
	}

}

func demoErrgroupCollectAllWithResults(tasks []task) {
	baseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var g errgroup.Group
	g.SetLimit(2)

	resultCh := make(chan fetchResult, len(tasks))

	for _, currentTask := range tasks {
		g.Go(func() error {
			data, err := runTaskWithResultCtx(baseCtx, currentTask)
			resultCh <- fetchResult{
				taskName: currentTask.name,
				data:     data,
				err:      err,
			}
			return nil
		})
	}

	_ = g.Wait()
	close(resultCh)

	for r := range resultCh {
		if r.err != nil {
			log.Printf("task %s failed with error: %v", r.taskName, r.err)
		} else {
			log.Printf("task %s succeeded with data: %s", r.taskName, r.data)
		}
	}
}
