package gormrepo

import (
	"time"

	jobsvc "github.com/tinboxw/skoll/internal/service/job"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"gorm.io/gorm"
)

func newWorkflowTestService(repo workflowsvc.Repository, db *gorm.DB) workflowsvc.Service {
	return workflowsvc.NewService(repo, workflowsvc.Options{
		Jobs:       jobsvc.NewService(NewJobStore(db), func() time.Time { return time.Now().UTC() }),
		UnitOfWork: storesql.NewUnitOfWorkWithDB(db),
		Now:        func() time.Time { return time.Now().UTC() },
	})
}
