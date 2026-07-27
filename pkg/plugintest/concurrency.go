package plugintest

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type ParallelResult[T any] struct {
	Index int
	Value T
	Err   error
}

type ParallelReport[T any] struct {
	Results []ParallelResult[T]
}

func RunConcurrent[T any](ctx context.Context, workers int, operation func(context.Context, int) (T, error)) (ParallelReport[T], error) {
	if workers < 1 {
		return ParallelReport[T]{}, errors.New("concurrent fixture requires at least one worker")
	}
	if operation == nil {
		return ParallelReport[T]{}, errors.New("concurrent fixture operation is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	start := make(chan struct{})
	results := make(chan ParallelResult[T], workers)
	var ready sync.WaitGroup
	ready.Add(workers)
	for index := 0; index < workers; index++ {
		go func(worker int) {
			ready.Done()
			select {
			case <-start:
			case <-ctx.Done():
				results <- ParallelResult[T]{Index: worker, Err: ctx.Err()}
				return
			}
			value, err := operation(ctx, worker)
			results <- ParallelResult[T]{Index: worker, Value: value, Err: err}
		}(index)
	}
	ready.Wait()
	close(start)
	report := ParallelReport[T]{Results: make([]ParallelResult[T], workers)}
	for range workers {
		result := <-results
		report.Results[result.Index] = result
	}
	return report, nil
}

func (r ParallelReport[T]) Errors() []error {
	failures := make([]error, 0)
	for _, result := range r.Results {
		if result.Err != nil {
			failures = append(failures, fmt.Errorf("worker %d: %w", result.Index, result.Err))
		}
	}
	return failures
}

func (r ParallelReport[T]) Values() []T {
	values := make([]T, 0, len(r.Results))
	for _, result := range r.Results {
		if result.Err == nil {
			values = append(values, result.Value)
		}
	}
	return values
}

func (r ParallelReport[T]) RequireSuccess() error {
	return errors.Join(r.Errors()...)
}
