package jobscheduler

import "time"

// DispatchClaimImport is the persistence-friendly view of a dispatch claim.
type DispatchClaimImport struct {
	ExecutionKey      string
	JobID             int64
	InstanceID        string
	ExecutionID       int64
	ClaimedAt         time.Time
	LeaseUntil        time.Time
	LeaseRenewalCount int64
}

// JobStateSnapshot bundles the persistent state of jobscheduler.Service.
type JobStateSnapshot struct {
	NextJobID       int64
	NextExecutionID int64
	Jobs            []Job
	Executions      []Execution // each carries its JobID
	Claims          []DispatchClaimImport
	RetryPolicies   map[int64]RetryPolicy
	RetrySchedules  map[string][]RetrySchedule
	DeadLetters     []DeadLetter
}

// ImportSnapshot replaces in-memory state with the supplied snapshot.
func (s *Service) ImportSnapshot(snap JobStateSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobs = make(map[int64]Job, len(snap.Jobs))
	for _, j := range snap.Jobs {
		s.jobs[j.ID] = j
	}

	s.executionItems = make(map[int64][]Execution)
	for _, e := range snap.Executions {
		s.executionItems[e.JobID] = append(s.executionItems[e.JobID], e)
	}

	s.claims = make(map[string]dispatchClaimRecord, len(snap.Claims))
	for _, c := range snap.Claims {
		s.claims[c.ExecutionKey] = dispatchClaimRecord{
			jobID:       c.JobID,
			instanceID:  c.InstanceID,
			executionID: c.ExecutionID,
			claimedAt:   c.ClaimedAt,
			leaseUntil:  c.LeaseUntil,
			renewCount:  c.LeaseRenewalCount,
		}
	}

	s.retryPolicies = make(map[int64]RetryPolicy, len(snap.RetryPolicies))
	for k, v := range snap.RetryPolicies {
		s.retryPolicies[k] = v
	}

	s.retrySchedules = make(map[string][]RetrySchedule, len(snap.RetrySchedules))
	for k, v := range snap.RetrySchedules {
		s.retrySchedules[k] = append([]RetrySchedule(nil), v...)
	}

	s.deadLetters = make(map[string]DeadLetter, len(snap.DeadLetters))
	for _, d := range snap.DeadLetters {
		s.deadLetters[d.ExecutionKey] = d
	}

	if snap.NextJobID > s.nextJobID {
		s.nextJobID = snap.NextJobID
	}
	if snap.NextExecutionID > s.nextExecution {
		s.nextExecution = snap.NextExecutionID
	}
}

// ExportClaims returns claims in persistence-friendly form.
func (s *Service) ExportClaims() []DispatchClaimImport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]DispatchClaimImport, 0, len(s.claims))
	for k, rec := range s.claims {
		out = append(out, DispatchClaimImport{
			ExecutionKey:      k,
			JobID:             rec.jobID,
			InstanceID:        rec.instanceID,
			ExecutionID:       rec.executionID,
			ClaimedAt:         rec.claimedAt,
			LeaseUntil:        rec.leaseUntil,
			LeaseRenewalCount: rec.renewCount,
		})
	}
	return out
}

// ExportRetryPolicies returns a copy of the retry-policy map.
func (s *Service) ExportRetryPolicies() map[int64]RetryPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[int64]RetryPolicy, len(s.retryPolicies))
	for k, v := range s.retryPolicies {
		out[k] = v
	}
	return out
}

// ExportRetrySchedules returns a deep copy of the retry-schedule map.
func (s *Service) ExportRetrySchedules() map[string][]RetrySchedule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string][]RetrySchedule, len(s.retrySchedules))
	for k, v := range s.retrySchedules {
		out[k] = append([]RetrySchedule(nil), v...)
	}
	return out
}

// ExportDeadLetters returns a copy of all dead-letter records.
func (s *Service) ExportDeadLetters() []DeadLetter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]DeadLetter, 0, len(s.deadLetters))
	for _, d := range s.deadLetters {
		out = append(out, d)
	}
	return out
}

// ExportNextIDs returns the current job-id and execution-id counters.
func (s *Service) ExportNextIDs() (nextJobID, nextExecutionID int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nextJobID, s.nextExecution
}
