package jobscheduler

import (
	"errors"
	"sort"
	"sync"
	"time"
)

var ErrJobNotFound = errors.New("job not found")

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

type Service struct {
	mu             sync.RWMutex
	nextJobID      int64
	nextExecution  int64
	jobs           map[int64]Job
	executionItems map[int64][]Execution
}

func NewService() *Service {
	return &Service{
		nextJobID:      1,
		nextExecution:  1,
		jobs:           make(map[int64]Job),
		executionItems: make(map[int64][]Execution),
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
