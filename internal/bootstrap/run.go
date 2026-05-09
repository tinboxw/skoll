package bootstrap

import "context"

// Runner holds process-level runtime dependencies and startup configuration.
type Runner struct{}

// NewRunnerFromEnv builds a runner from environment configuration.
func NewRunnerFromEnv() (*Runner, error) {
	return &Runner{}, nil
}

// Run starts the runtime and blocks until context cancellation.
func (r *Runner) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}
