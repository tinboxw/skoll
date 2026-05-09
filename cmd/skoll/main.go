package main

import "log"

func main() {
	ctx, stop := withShutdownSignalContext()
	defer stop()

	runner, err := newServerRunner()
	if err != nil {
		log.Fatalf("bootstrap config error: %v", err)
	}

	if err := runner.Run(ctx); err != nil {
		log.Fatalf("runner error: %v", err)
	}
}
