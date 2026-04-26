package main

import (
	"log"

	"github.com/tinboxw/skoll/internal/bootstrap"
)

func main() {
	if err := bootstrap.Run(); err != nil {
		log.Fatalf("startup failed: %v", err)
	}
}
