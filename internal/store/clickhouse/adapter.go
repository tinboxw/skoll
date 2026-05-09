package clickhouse

import (
	"fmt"
	"strings"

	"github.com/tinboxw/skoll/internal/repository"
)

type Adapter struct {
	dsn     string
	audit   repository.AuditRepository
	metrics *MetricsStore
}

func NewAdapter(dsn string) (*Adapter, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("clickhouse dsn is required")
	}
	return &Adapter{
		dsn:     dsn,
		audit:   NewAuditStore(),
		metrics: NewMetricsStore(),
	}, nil
}

func (a *Adapter) DSN() string { return a.dsn }
func (a *Adapter) AuditRepository() repository.AuditRepository {
	return a.audit
}
func (a *Adapter) MetricsStore() *MetricsStore {
	return a.metrics
}
