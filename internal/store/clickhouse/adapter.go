package clickhouse

import (
	"fmt"
	"strings"

	auditrepo "github.com/tinboxw/skoll/internal/repository/audit"
)

type Adapter struct {
	dsn     string
	audit   auditrepo.AuditRepository
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
func (a *Adapter) AuditRepository() auditrepo.AuditRepository {
	return a.audit
}
func (a *Adapter) MetricsStore() *MetricsStore {
	return a.metrics
}
