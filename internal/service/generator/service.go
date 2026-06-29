package generator

import (
	"context"
)

type Service interface {
	DryRun(ctx context.Context, in DryRunInput) (*DryRunResult, error)
}
