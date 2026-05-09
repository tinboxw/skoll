package main

import "github.com/tinboxw/skoll/internal/bootstrap"

func newServerRunner() (*bootstrap.Runner, error) {
	return bootstrap.NewRunnerFromEnv()
}
