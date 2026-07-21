package pluginsdk

import (
	"context"
	"time"
)

type AuditResult string

const (
	AuditResultSuccess AuditResult = "success"
	AuditResultFailure AuditResult = "failure"
	AuditResultDenied  AuditResult = "denied"
)

type AuditRisk string

const (
	AuditRiskLow      AuditRisk = "low"
	AuditRiskMedium   AuditRisk = "medium"
	AuditRiskHigh     AuditRisk = "high"
	AuditRiskCritical AuditRisk = "critical"
)

type AuditEntry struct {
	Action     string
	Resource   string
	ResourceID string
	Result     AuditResult
	Risk       AuditRisk
	Detail     map[string]any
}

type AuditReceipt struct {
	ID         string
	OccurredAt time.Time
}

type AuditService interface {
	Record(ctx context.Context, entry AuditEntry) (AuditReceipt, error)
}
