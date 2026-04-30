package bootstrap

import (
	"context"
	"log"
	"os"
	"time"
)

type shutdownServer interface {
	SetReady(bool)
	Shutdown(context.Context) error
}

func performGracefulStop(srv shutdownServer, drainTime, shutdownTimeout time.Duration, sigCh <-chan os.Signal) error {
	srv.SetReady(false)
	if drainTime > 0 {
		log.Printf("readiness switched to not_ready, draining for %s", drainTime)
		if interrupted := waitForDrain(drainTime, sigCh); interrupted {
			log.Printf("drain interrupted by a second termination signal")
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func waitForDrain(drainTime time.Duration, sigCh <-chan os.Signal) bool {
	timer := time.NewTimer(drainTime)
	defer timer.Stop()

	select {
	case <-timer.C:
		return false
	case <-sigCh:
		return true
	}
}
