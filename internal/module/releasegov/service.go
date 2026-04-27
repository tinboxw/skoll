package releasegov

import (
	"errors"
	"math"
	"strings"
	"sync"
	"time"
)

var ErrMilestoneRequired = errors.New("milestone is required")

type EvidenceInput struct {
	Milestone           string  `json:"milestone"`
	GoTestPassed        bool    `json:"go_test_passed"`
	GoRacePassed        bool    `json:"go_race_passed"`
	ReadmeSynced        bool    `json:"readme_synced"`
	BenchmarkNsPerOp    float64 `json:"benchmark_ns_per_op"`
	BaselineNsPerOp     float64 `json:"baseline_ns_per_op"`
	BenchmarkCommand    string  `json:"benchmark_command,omitempty"`
	EvidenceDescription string  `json:"evidence_description,omitempty"`
}

type Evidence struct {
	EvidenceInput
	CollectedAtUnixSec int64 `json:"collected_at_unix_sec"`
}

type Scorecard struct {
	Milestone             string   `json:"milestone"`
	EvidenceCount         int      `json:"evidence_count"`
	QualityGatePassed     bool     `json:"quality_gate_passed"`
	ReleaseReady          bool     `json:"release_ready"`
	PerformanceRegression float64  `json:"performance_regression_ratio"`
	AllowedRegression     float64  `json:"allowed_regression_ratio"`
	FailedChecks          []string `json:"failed_checks,omitempty"`
	LastUpdatedUnixSec    int64    `json:"last_updated_unix_sec,omitempty"`
}

type Service struct {
	mu       sync.RWMutex
	evidence map[string][]Evidence
}

func NewService() *Service {
	return &Service{evidence: make(map[string][]Evidence)}
}

func (s *Service) SubmitEvidence(input EvidenceInput, now time.Time) (Evidence, error) {
	milestone := strings.TrimSpace(strings.ToLower(input.Milestone))
	if milestone == "" {
		return Evidence{}, ErrMilestoneRequired
	}

	input.Milestone = milestone
	input.BenchmarkCommand = strings.TrimSpace(input.BenchmarkCommand)
	input.EvidenceDescription = strings.TrimSpace(input.EvidenceDescription)

	evidence := Evidence{EvidenceInput: input, CollectedAtUnixSec: now.UTC().Unix()}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.evidence[milestone] = append(s.evidence[milestone], evidence)
	return evidence, nil
}

func (s *Service) Scorecard(milestone string, allowedRegression float64) Scorecard {
	key := strings.TrimSpace(strings.ToLower(milestone))
	if allowedRegression <= 0 {
		allowedRegression = 0.10
	}

	s.mu.RLock()
	items := s.evidence[key]
	s.mu.RUnlock()

	card := Scorecard{
		Milestone:         key,
		EvidenceCount:     len(items),
		AllowedRegression: allowedRegression,
		QualityGatePassed: false,
		ReleaseReady:      false,
		FailedChecks:      []string{},
	}
	if len(items) == 0 {
		card.FailedChecks = append(card.FailedChecks, "missing_evidence")
		return card
	}

	latest := items[len(items)-1]
	card.LastUpdatedUnixSec = latest.CollectedAtUnixSec
	failed := make([]string, 0, 4)
	if !latest.GoTestPassed {
		failed = append(failed, "go_test_failed")
	}
	if !latest.GoRacePassed {
		failed = append(failed, "go_race_failed")
	}
	if !latest.ReadmeSynced {
		failed = append(failed, "readme_not_synced")
	}

	regression := 0.0
	if latest.BaselineNsPerOp > 0 && latest.BenchmarkNsPerOp > 0 {
		regression = (latest.BenchmarkNsPerOp - latest.BaselineNsPerOp) / latest.BaselineNsPerOp
		regression = math.Round(regression*10000) / 10000
		if regression > allowedRegression {
			failed = append(failed, "performance_regression_exceeded")
		}
	}

	card.PerformanceRegression = regression
	card.FailedChecks = failed
	card.QualityGatePassed = len(failed) == 0
	card.ReleaseReady = card.QualityGatePassed
	return card
}
