package persistent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tinboxw/skoll/internal/module/jobscheduler"
	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
)

const (
	jobKindClaims         = "claims"
	jobKindRetryPolicies  = "retry_policies"
	jobKindRetrySchedules = "retry_schedules"
	jobKindDeadLetters    = "dead_letters"
)

// JobModel persists the core job record.
type JobModel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement:false"`
	Name      string    `gorm:"size:255;not null"`
	Schedule  string    `gorm:"size:255"`
	Enabled   bool      `gorm:"not null;default:true"`
	CreatedAt time.Time `gorm:"not null"`
}

func (JobModel) TableName() string { return "skoll_jobs" }

// JobExecutionModel persists per-run execution records.
type JobExecutionModel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement:false"`
	JobID     int64     `gorm:"index;not null"`
	Status    string    `gorm:"size:32;not null"`
	StartedAt time.Time `gorm:""`
	EndedAt   time.Time `gorm:""`
	Message   string    `gorm:"size:1024"`
}

func (JobExecutionModel) TableName() string { return "skoll_job_executions" }

// JobStateModel stores the opaque JSON aggregate for a single state kind:
// claims / retry_policies / retry_schedules / dead_letters.
type JobStateModel struct {
	Kind    string `gorm:"primaryKey;size:32"`
	Payload string `gorm:"type:text;not null"`
}

func (JobStateModel) TableName() string { return "skoll_job_state" }

// JobCounterModel stores the next id counters.
type JobCounterModel struct {
	ID              string `gorm:"primaryKey;size:32"` // always "job"
	NextJobID       int64  `gorm:"not null;default:1"`
	NextExecutionID int64  `gorm:"not null;default:1"`
}

func (JobCounterModel) TableName() string { return "skoll_job_counter" }

const jobCounterRowID = "job"

// JobRepository implements contracts.JobRepository using a write-through
// pattern over jobscheduler.NewService(). Each mutation commits the affected
// aggregate (jobs, executions, or state-kind JSON) plus the next-id
// counters in the same logical step.
type JobRepository struct {
	db    *gorm.DB
	mu    sync.Mutex
	inner *jobscheduler.Service
}

// NewJobRepository constructs a SQL-backed job repository, hydrating state
// from the database. Tables must already be migrated.
func NewJobRepository(database *gorm.DB) (*JobRepository, error) {
	if database == nil {
		return nil, fmt.Errorf("persistent: nil db")
	}
	r := &JobRepository{db: database, inner: jobscheduler.NewService()}
	if err := r.hydrate(); err != nil {
		return nil, err
	}
	return r, nil
}

// Models returns the gorm models managed by this repository.
func (r *JobRepository) Models() []any {
	return []any{
		&JobModel{},
		&JobExecutionModel{},
		&JobStateModel{},
		&JobCounterModel{},
	}
}

func (r *JobRepository) ctx() context.Context { return context.Background() }

func (r *JobRepository) hydrate() error {
	var (
		jobs     []JobModel
		execs    []JobExecutionModel
		stateRow []JobStateModel
		counter  JobCounterModel
	)
	if err := r.db.WithContext(r.ctx()).Find(&jobs).Error; err != nil {
		return fmt.Errorf("hydrate jobs: %w", err)
	}
	if err := r.db.WithContext(r.ctx()).Order("id asc").Find(&execs).Error; err != nil {
		return fmt.Errorf("hydrate executions: %w", err)
	}
	if err := r.db.WithContext(r.ctx()).Find(&stateRow).Error; err != nil {
		return fmt.Errorf("hydrate job state: %w", err)
	}
	_ = r.db.WithContext(r.ctx()).First(&counter, "id = ?", jobCounterRowID).Error

	snap := jobscheduler.JobStateSnapshot{
		NextJobID:       counter.NextJobID,
		NextExecutionID: counter.NextExecutionID,
		RetryPolicies:   map[int64]jobscheduler.RetryPolicy{},
		RetrySchedules:  map[string][]jobscheduler.RetrySchedule{},
	}
	for _, j := range jobs {
		snap.Jobs = append(snap.Jobs, jobscheduler.Job{
			ID:        j.ID,
			Name:      j.Name,
			Schedule:  j.Schedule,
			Enabled:   j.Enabled,
			CreatedAt: j.CreatedAt,
		})
	}
	for _, e := range execs {
		snap.Executions = append(snap.Executions, jobscheduler.Execution{
			ID:        e.ID,
			JobID:     e.JobID,
			Status:    e.Status,
			StartedAt: e.StartedAt,
			EndedAt:   e.EndedAt,
			Message:   e.Message,
		})
	}
	for _, row := range stateRow {
		switch row.Kind {
		case jobKindClaims:
			var claims []jobscheduler.DispatchClaimImport
			if err := json.Unmarshal([]byte(row.Payload), &claims); err != nil {
				return fmt.Errorf("decode claims: %w", err)
			}
			snap.Claims = claims
		case jobKindRetryPolicies:
			var pol map[int64]jobscheduler.RetryPolicy
			if err := json.Unmarshal([]byte(row.Payload), &pol); err != nil {
				return fmt.Errorf("decode retry policies: %w", err)
			}
			snap.RetryPolicies = pol
		case jobKindRetrySchedules:
			var sch map[string][]jobscheduler.RetrySchedule
			if err := json.Unmarshal([]byte(row.Payload), &sch); err != nil {
				return fmt.Errorf("decode retry schedules: %w", err)
			}
			snap.RetrySchedules = sch
		case jobKindDeadLetters:
			var dls []jobscheduler.DeadLetter
			if err := json.Unmarshal([]byte(row.Payload), &dls); err != nil {
				return fmt.Errorf("decode dead letters: %w", err)
			}
			snap.DeadLetters = dls
		}
	}
	r.inner.ImportSnapshot(snap)
	return nil
}

// --- write-through helpers --------------------------------------------------

func (r *JobRepository) saveCounter() {
	nj, ne := r.inner.ExportNextIDs()
	row := JobCounterModel{ID: jobCounterRowID, NextJobID: nj, NextExecutionID: ne}
	if err := r.db.WithContext(r.ctx()).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"next_job_id", "next_execution_id"}),
	}).Create(&row).Error; err != nil {
		panic(fmt.Errorf("persist job counter: %w", err))
	}
}

func (r *JobRepository) saveStateKind(kind string, value any) {
	payload, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Errorf("encode job state %s: %w", kind, err))
	}
	row := JobStateModel{Kind: kind, Payload: string(payload)}
	if err := r.db.WithContext(r.ctx()).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "kind"}},
		DoUpdates: clause.AssignmentColumns([]string{"payload"}),
	}).Create(&row).Error; err != nil {
		panic(fmt.Errorf("persist job state %s: %w", kind, err))
	}
}

func (r *JobRepository) saveClaims() { r.saveStateKind(jobKindClaims, r.inner.ExportClaims()) }
func (r *JobRepository) saveRetryPolicies() {
	r.saveStateKind(jobKindRetryPolicies, r.inner.ExportRetryPolicies())
}
func (r *JobRepository) saveRetrySchedules() {
	r.saveStateKind(jobKindRetrySchedules, r.inner.ExportRetrySchedules())
}
func (r *JobRepository) saveDeadLetters() {
	r.saveStateKind(jobKindDeadLetters, r.inner.ExportDeadLetters())
}

func (r *JobRepository) saveJob(j jobscheduler.Job) {
	row := JobModel{
		ID: j.ID, Name: j.Name, Schedule: j.Schedule, Enabled: j.Enabled, CreatedAt: j.CreatedAt,
	}
	if err := r.db.WithContext(r.ctx()).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "schedule", "enabled", "created_at"}),
	}).Create(&row).Error; err != nil {
		panic(fmt.Errorf("persist job: %w", err))
	}
}

func (r *JobRepository) saveExecution(e jobscheduler.Execution) {
	row := JobExecutionModel{
		ID: e.ID, JobID: e.JobID, Status: e.Status,
		StartedAt: e.StartedAt, EndedAt: e.EndedAt, Message: e.Message,
	}
	if err := r.db.WithContext(r.ctx()).Create(&row).Error; err != nil {
		panic(fmt.Errorf("persist execution: %w", err))
	}
}

// --- contracts.JobRepository -----------------------------------------------

func (r *JobRepository) Create(name, schedule string) jobscheduler.Job {
	r.mu.Lock()
	defer r.mu.Unlock()
	j := r.inner.Create(name, schedule)
	r.saveJob(j)
	r.saveCounter()
	return j
}

func (r *JobRepository) Get(id int64) (jobscheduler.Job, error) { return r.inner.Get(id) }
func (r *JobRepository) List() []jobscheduler.Job               { return r.inner.List() }

func (r *JobRepository) Run(jobID int64) (jobscheduler.Execution, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, err := r.inner.Run(jobID)
	if err != nil {
		return e, err
	}
	r.saveExecution(e)
	r.saveCounter()
	return e, nil
}

func (r *JobRepository) History(jobID int64, limit int) []jobscheduler.Execution {
	return r.inner.History(jobID, limit)
}

func (r *JobRepository) ClaimRun(jobID int64, executionKey, instanceID string, now time.Time) (jobscheduler.DispatchClaim, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, err := r.inner.ClaimRun(jobID, executionKey, instanceID, now)
	if err != nil {
		return c, err
	}
	r.saveClaims()
	r.saveCounter()
	return c, nil
}

func (r *JobRepository) RenewClaimLease(executionKey, instanceID string, leaseTTLSeconds int64, now time.Time) (jobscheduler.DispatchClaim, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, err := r.inner.RenewClaimLease(executionKey, instanceID, leaseTTLSeconds, now)
	if err != nil {
		return c, err
	}
	r.saveClaims()
	return c, nil
}

func (r *JobRepository) ClaimStatus(executionKey string) jobscheduler.DispatchClaim {
	return r.inner.ClaimStatus(executionKey)
}

func (r *JobRepository) SetRetryPolicy(jobID int64, policy jobscheduler.RetryPolicy) (jobscheduler.RetryPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, err := r.inner.SetRetryPolicy(jobID, policy)
	if err != nil {
		return p, err
	}
	r.saveRetryPolicies()
	return p, nil
}

func (r *JobRepository) GetRetryPolicy(jobID int64) (jobscheduler.RetryPolicy, error) {
	return r.inner.GetRetryPolicy(jobID)
}

func (r *JobRepository) ScheduleRetry(jobID int64, executionKey string, attempt int, now time.Time) (jobscheduler.RetrySchedule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, err := r.inner.ScheduleRetry(jobID, executionKey, attempt, now)
	if err != nil {
		return s, err
	}
	r.saveRetrySchedules()
	return s, nil
}

func (r *JobRepository) MarkDeadLetter(jobID int64, executionKey, reason string, retryCount int, now time.Time) (jobscheduler.DeadLetter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, err := r.inner.MarkDeadLetter(jobID, executionKey, reason, retryCount, now)
	if err != nil {
		return d, err
	}
	r.saveDeadLetters()
	return d, nil
}

func (r *JobRepository) ListDeadLetters(limit int) []jobscheduler.DeadLetter {
	return r.inner.ListDeadLetters(limit)
}

func (r *JobRepository) ReplayDeadLetter(executionKey, operator string, now time.Time) (jobscheduler.DeadLetter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, err := r.inner.ReplayDeadLetter(executionKey, operator, now)
	if err != nil {
		return d, err
	}
	r.saveDeadLetters()
	return d, nil
}

func (r *JobRepository) ReliabilitySnapshot(now time.Time) jobscheduler.ReliabilityMetrics {
	return r.inner.ReliabilitySnapshot(now)
}

var _ contracts.JobRepository = (*JobRepository)(nil)
