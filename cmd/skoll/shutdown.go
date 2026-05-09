package main

import (
	"context"
	"os/signal"
	"syscall"
)

func withShutdownSignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}
