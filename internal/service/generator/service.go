package generator

import (
	"context"
)

type Service interface {
	DryRun(ctx context.Context, in DryRunInput) (*DryRunResult, error)
	RecordHistory(ctx context.Context, in RecordHistoryInput) (*GenerationHistory, error)
	GetHistory(ctx context.Context, batchID string) (*GenerationHistory, error)
	PlanRollback(ctx context.Context, in RollbackInput) (*RollbackPlan, error)
}
