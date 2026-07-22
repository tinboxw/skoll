package bootstrap

import (
	"context"
	"sync"

	"github.com/tinboxw/skoll/internal/plugin"
)

type readyTestServiceLauncher struct{}

func (l *readyTestServiceLauncher) Start(ctx context.Context, _ plugin.Info) (plugin.ServiceHandle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &readyTestServiceHandle{done: make(chan error, 1)}, nil
}

type readyTestServiceHandle struct {
	done chan error
	once sync.Once
}

func (h *readyTestServiceHandle) Done() <-chan error { return h.done }

func (h *readyTestServiceHandle) Stop(context.Context) error {
	h.finish()
	return nil
}

func (h *readyTestServiceHandle) ForceStop() error {
	h.finish()
	return nil
}

func (h *readyTestServiceHandle) finish() {
	h.once.Do(func() {
		h.done <- nil
		close(h.done)
	})
}
