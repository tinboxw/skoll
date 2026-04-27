package releasegov

import (
	"testing"
	"time"
)

func TestServiceScorecard(t *testing.T) {
	svc := NewService()
	now := time.Unix(1710000000, 0).UTC()

	_, err := svc.SubmitEvidence(EvidenceInput{
		Milestone:        "E8-step1",
		GoTestPassed:     true,
		GoRacePassed:     true,
		ReadmeSynced:     true,
		BenchmarkNsPerOp: 3200,
		BaselineNsPerOp:  3000,
	}, now)
	if err != nil {
		t.Fatalf("submit evidence failed: %v", err)
	}

	card := svc.Scorecard("E8-step1", 0.10)
	if !card.QualityGatePassed || !card.ReleaseReady {
		t.Fatalf("expected passing scorecard, got %+v", card)
	}

	_, err = svc.SubmitEvidence(EvidenceInput{
		Milestone:        "E8-step1",
		GoTestPassed:     true,
		GoRacePassed:     true,
		ReadmeSynced:     true,
		BenchmarkNsPerOp: 3800,
		BaselineNsPerOp:  3000,
	}, now.Add(1*time.Hour))
	if err != nil {
		t.Fatalf("submit evidence failed: %v", err)
	}

	card = svc.Scorecard("E8-step1", 0.10)
	if card.QualityGatePassed {
		t.Fatalf("expected failing scorecard due to regression, got %+v", card)
	}
	if len(card.FailedChecks) == 0 {
		t.Fatalf("expected failed checks populated")
	}
}
