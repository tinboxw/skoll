package service

import (
	"sync/atomic"
	"time"

	"github.com/tinboxw/skoll/internal/domain"
)

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now().UTC()
}

type SystemService struct {
	ready     atomic.Bool
	startedAt time.Time
	version   string
	clock     Clock

	healthCacheAt  atomic.Int64
	healthCacheVal atomic.Value
	healthCacheTTL time.Duration
}

func NewSystemService(version string, clock Clock) *SystemService {
	if clock == nil {
		clock = realClock{}
	}

	return &SystemService{
		startedAt:      clock.Now(),
		version:        version,
		clock:          clock,
		healthCacheTTL: 200 * time.Millisecond,
	}
}

func (s *SystemService) Health() domain.Health {
	now := s.clock.Now()
	cachedAtUnixNano := s.healthCacheAt.Load()
	if cachedAtUnixNano != 0 {
		cachedAt := time.Unix(0, cachedAtUnixNano)
		if now.Sub(cachedAt) <= s.healthCacheTTL {
			if cached := s.healthCacheVal.Load(); cached != nil {
				return cached.(domain.Health)
			}
		}
	}

	fresh := domain.Health{
		Status:  "ok",
		Version: s.version,
		Uptime:  now.Sub(s.startedAt),
	}

	s.healthCacheVal.Store(fresh)
	s.healthCacheAt.Store(now.UnixNano())

	return fresh
}

func (s *SystemService) Readiness() domain.Readiness {
	if !s.ready.Load() {
		return domain.ReadinessNotReady
	}

	return domain.ReadinessReady
}

func (s *SystemService) SetReady(v bool) {
	s.ready.Store(v)
}
