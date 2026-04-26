package domain

import "time"

type Readiness string

const (
	ReadinessReady    Readiness = "ready"
	ReadinessNotReady Readiness = "not_ready"
)

type Health struct {
	Status  string
	Version string
	Uptime  time.Duration
}
