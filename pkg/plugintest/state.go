package plugintest

import (
	"context"
	"errors"
	"sync"
	"time"
)

type TransactionalValue[T any] struct {
	mu    sync.RWMutex
	value T
	clone func(T) T
}

func NewTransactionalValue[T any](initial T, clone func(T) T) (*TransactionalValue[T], error) {
	if clone == nil {
		return nil, errors.New("transactional fixture clone function is required")
	}
	return &TransactionalValue[T]{value: clone(initial), clone: clone}, nil
}

func (s *TransactionalValue[T]) Within(ctx context.Context, mutate func(*T) error) error {
	if s == nil || s.clone == nil {
		return errors.New("transactional fixture is not configured")
	}
	if mutate == nil {
		return errors.New("transactional fixture mutation is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.clone(s.value)
	if err := mutate(&next); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.value = s.clone(next)
	return nil
}

func (s *TransactionalValue[T]) Snapshot() (T, error) {
	var zero T
	if s == nil || s.clone == nil {
		return zero, errors.New("transactional fixture is not configured")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clone(s.value), nil
}

func Eventually(ctx context.Context, interval time.Duration, check func() (bool, error)) error {
	if check == nil {
		return errors.New("eventual fixture check is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if interval <= 0 {
		interval = 10 * time.Millisecond
	}
	for {
		satisfied, err := check()
		if err != nil {
			return err
		}
		if satisfied {
			return nil
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
}
