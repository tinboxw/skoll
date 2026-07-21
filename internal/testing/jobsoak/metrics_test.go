package jobsoak

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestMeasureRejectsFailuresAndDuplicateSideEffects(t *testing.T) {
	scenario := Measure("threshold proof", 2, Thresholds{
		MaxP95Milliseconds: 1000, MaxP99Milliseconds: 1000,
		MaxHeapGrowthBytes: 64 << 20, MaxGoroutineGrowth: 4,
		MaxFailures: 0, MaxDuplicateSideEffects: 0,
	}, func(index int) (Observation, error) {
		if index == 0 {
			return Observation{DuplicateSideEffects: 1}, errors.New("expected failure")
		}
		return Observation{}, nil
	})
	if scenario.Passed || scenario.Failures != 1 || scenario.DuplicateSideEffects != 1 || len(scenario.Alerts) < 2 {
		t.Fatalf("threshold breach was not observable: %+v", scenario)
	}
}

func TestWriteReportCreatesMachineReadableEvidence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "report.json")
	report := NewReport(true, Scenario{Name: "通过", Iterations: 1, Succeeded: 1, Passed: true})
	if err := WriteReport(path, report); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 || !report.Passed || report.Locale != "zh-CN" || !report.RaceEnabled {
		t.Fatalf("unexpected report: %+v body=%q", report, body)
	}
}
