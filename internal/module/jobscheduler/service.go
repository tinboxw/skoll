package jobscheduler

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

var ErrJobNotFound = errors.New("job not found")
var ErrExecutionKeyRequired = errors.New("execution key required")
var ErrClaimNotFound = errors.New("dispatch claim not found")
var ErrClaimLeaseOwnerMismatch = errors.New("dispatch claim lease owner mismatch")
var ErrInvalidLeaseTTL = errors.New("invalid lease ttl")

type Job struct {
	ID        int64
	Name      string
	Schedule  string
	Enabled   bool
	CreatedAt time.Time
}

type Execution struct {
	ID        int64
	JobID     int64
	Status    string
	StartedAt time.Time
	EndedAt   time.Time
	Message   string
}

type DispatchClaim struct {
	ExecutionKey      string `json:"execution_key"`
	JobID             int64  `json:"job_id"`
	InstanceID        string `json:"instance_id,omitempty"`
	ExecutionID       int64  `json:"execution_id,omitempty"`
	Claimed           bool   `json:"claimed"`
	DuplicateBlocked  bool   `json:"duplicate_blocked"`
	ClaimedAtUnixSec  int64  `json:"claimed_at_unix_sec,omitempty"`
	LeaseUntilUnixSec int64  `json:"lease_until_unix_sec,omitempty"`
	LeaseRenewalCount int64  `json:"lease_renewal_count,omitempty"`
	Message           string `json:"message,omitempty"`
}

type dispatchClaimRecord struct {
	jobID       int64
	instanceID  string
	executionID int64
	claimedAt   time.Time
	leaseUntil  time.Time
	renewCount  int64
}

type Service struct {
	mu             sync.RWMutex
	nextJobID      int64
	nextExecution  int64
	jobs           map[int64]Job
	executionItems map[int64][]Execution
	claims         map[string]dispatchClaimRecord
}

func NewService() *Service {
	return &Service{
		nextJobID:      1,
		nextExecution:  1,
		jobs:           make(map[int64]Job),
		executionItems: make(map[int64][]Execution),
		claims:         make(map[string]dispatchClaimRecord),
	}
}

func (s *Service) Create(name, schedule string) Job {
	s.mu.Lock()
	defer s.mu.Unlock()

	job := Job{
		ID:        s.nextJobID,
		Name:      name,
		Schedule:  schedule,
		Enabled:   true,
		CreatedAt: time.Now().UTC(),
	}
	s.nextJobID++
	s.jobs[job.ID] = job
	return job
}

func (s *Service) Get(id int64) (Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	if !ok {
		return Job{}, ErrJobNotFound
	}
	return job, nil
}

func (s *Service) List() []Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Job, 0, len(s.jobs))
	for _, item := range s.jobs {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) Run(jobID int64) (Execution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[jobID]; !ok {
		return Execution{}, ErrJobNotFound
	}

	start := time.Now().UTC()
	exec := Execution{
		ID:        s.nextExecution,
		JobID:     jobID,
		Status:    "success",
		StartedAt: start,
		EndedAt:   time.Now().UTC(),
		Message:   "run completed",
	}
	s.nextExecution++
	s.executionItems[jobID] = append(s.executionItems[jobID], exec)
	return exec, nil
}

func (s *Service) History(jobID int64, limit int) []Execution {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := s.executionItems[jobID]
	if len(items) == 0 || limit <= 0 {
		return nil
	}
	if limit > len(items) {
		limit = len(items)
	}
	start := len(items) - limit
	out := make([]Execution, limit)
	copy(out, items[start:])
	return out
}

func (s *Service) ClaimRun(jobID int64, executionKey, instanceID string, now time.Time) (DispatchClaim, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.jobs[jobID]; !ok {
		return DispatchClaim{}, ErrJobNotFound
	}
	if executionKey == "" {
		return DispatchClaim{}, ErrExecutionKeyRequired
	}

	if rec, exists := s.claims[executionKey]; exists {
		if rec.jobID == jobID && rec.instanceID == instanceID {
			return toDispatchClaim(executionKey, rec, true, false, "idempotent claim"), nil
		}
		return toDispatchClaim(executionKey, rec, false, true, "duplicate execution key blocked"), nil
	}

	rec := dispatchClaimRecord{
		jobID:       jobID,
		instanceID:  instanceID,
		executionID: s.nextExecution,
		claimedAt:   now.UTC(),
		leaseUntil:  now.UTC().Add(30 * time.Second),
	}
	s.nextExecution++
	s.claims[executionKey] = rec

	return toDispatchClaim(executionKey, rec, true, false, "claim accepted"), nil
}

func (s *Service) RenewClaimLease(executionKey, instanceID string, leaseTTLSeconds int64, now time.Time) (DispatchClaim, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	executionKey = strings.TrimSpace(executionKey)
	if executionKey == "" {
		return DispatchClaim{}, ErrExecutionKeyRequired
	}
	if leaseTTLSeconds <= 0 {
		return DispatchClaim{}, ErrInvalidLeaseTTL
	}
	rec, ok := s.claims[executionKey]
	if !ok {
		return DispatchClaim{}, ErrClaimNotFound
	}
	if strings.TrimSpace(instanceID) == "" || rec.instanceID != strings.TrimSpace(instanceID) {
		return DispatchClaim{}, ErrClaimLeaseOwnerMismatch
	}
	rec.leaseUntil = now.UTC().Add(time.Duration(leaseTTLSeconds) * time.Second)
	rec.renewCount++
	s.claims[executionKey] = rec
	return toDispatchClaim(executionKey, rec, true, false, "lease renewed"), nil
}

func (s *Service) ClaimStatus(executionKey string) DispatchClaim {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.claims[executionKey]
	if !ok {
		return DispatchClaim{ExecutionKey: executionKey, Claimed: false, DuplicateBlocked: false, Message: "not found"}
	}
	return toDispatchClaim(executionKey, rec, true, false, "claimed")
}

func toDispatchClaim(executionKey string, rec dispatchClaimRecord, claimed, duplicate bool, message string) DispatchClaim {
	out := DispatchClaim{
		ExecutionKey:     executionKey,
		JobID:            rec.jobID,
		InstanceID:       rec.instanceID,
		ExecutionID:      rec.executionID,
		Claimed:          claimed,
		DuplicateBlocked: duplicate,
		Message:          message,
	}
	if !rec.claimedAt.IsZero() {
		out.ClaimedAtUnixSec = rec.claimedAt.Unix()
	}
	if !rec.leaseUntil.IsZero() {
		out.LeaseUntilUnixSec = rec.leaseUntil.Unix()
	}
	out.LeaseRenewalCount = rec.renewCount
	return out
}
