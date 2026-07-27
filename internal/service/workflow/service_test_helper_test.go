package workflow

import (
	"time"

	"github.com/tinboxw/skoll/internal/service/job"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
)

func newTestService(repo Repository) Service {
	return NewService(repo, Options{
		Jobs:       job.NewService(job.NewMemoryRepository(), func() time.Time { return time.Now().UTC() }),
		UnitOfWork: storesql.NewUnitOfWork(),
		Now:        func() time.Time { return time.Now().UTC() },
	})
}
