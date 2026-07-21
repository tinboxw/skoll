package jobsoak

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"
)

type Thresholds struct {
	MaxP95Milliseconds      float64 `json:"maxP95Milliseconds"`
	MaxP99Milliseconds      float64 `json:"maxP99Milliseconds"`
	MaxHeapGrowthBytes      int64   `json:"maxHeapGrowthBytes"`
	MaxGoroutineGrowth      int     `json:"maxGoroutineGrowth"`
	MaxFailures             int     `json:"maxFailures"`
	MaxDuplicateSideEffects int     `json:"maxDuplicateSideEffects"`
}

type Observation struct {
	Retries              int
	DeadLetters          int
	DuplicateSideEffects int
}

type Scenario struct {
	Name                 string     `json:"name"`
	Iterations           int        `json:"iterations"`
	Succeeded            int        `json:"succeeded"`
	Failures             int        `json:"failures"`
	Retries              int        `json:"retries"`
	DeadLetters          int        `json:"deadLetters"`
	DuplicateSideEffects int        `json:"duplicateSideEffects"`
	P50Milliseconds      float64    `json:"p50Milliseconds"`
	P95Milliseconds      float64    `json:"p95Milliseconds"`
	P99Milliseconds      float64    `json:"p99Milliseconds"`
	HeapGrowthBytes      int64      `json:"heapGrowthBytes"`
	GoroutineGrowth      int        `json:"goroutineGrowth"`
	Thresholds           Thresholds `json:"thresholds"`
	Alerts               []string   `json:"alerts"`
	Passed               bool       `json:"passed"`
}

type Report struct {
	SchemaVersion string            `json:"schemaVersion"`
	Locale        string            `json:"locale"`
	GeneratedAt   time.Time         `json:"generatedAt"`
	GoVersion     string            `json:"goVersion"`
	GOOS          string            `json:"goos"`
	GOARCH        string            `json:"goarch"`
	RaceEnabled   bool              `json:"raceEnabled"`
	Scenarios     []Scenario        `json:"scenarios"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Passed        bool              `json:"passed"`
}

func Measure(name string, iterations int, thresholds Thresholds, operation func(int) (Observation, error)) Scenario {
	if iterations < 1 {
		iterations = 1
	}
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	beforeGoroutines := runtime.NumGoroutine()
	durations := make([]time.Duration, 0, iterations)
	result := Scenario{Name: name, Iterations: iterations, Thresholds: thresholds, Alerts: []string{}}
	for index := 0; index < iterations; index++ {
		startedAt := time.Now()
		observation, err := operation(index)
		durations = append(durations, time.Since(startedAt))
		result.Retries += observation.Retries
		result.DeadLetters += observation.DeadLetters
		result.DuplicateSideEffects += observation.DuplicateSideEffects
		if err != nil {
			result.Failures++
			continue
		}
		result.Succeeded++
	}
	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	result.HeapGrowthBytes = int64(after.HeapAlloc) - int64(before.HeapAlloc)
	result.GoroutineGrowth = runtime.NumGoroutine() - beforeGoroutines
	result.P50Milliseconds = percentileMilliseconds(durations, 0.50)
	result.P95Milliseconds = percentileMilliseconds(durations, 0.95)
	result.P99Milliseconds = percentileMilliseconds(durations, 0.99)
	result.evaluate()
	return result
}

func NewReport(raceEnabled bool, scenarios ...Scenario) Report {
	report := Report{
		SchemaVersion: "skoll.h5-job-soak.v1",
		Locale:        "zh-CN",
		GeneratedAt:   time.Now().UTC(),
		GoVersion:     runtime.Version(),
		GOOS:          runtime.GOOS,
		GOARCH:        runtime.GOARCH,
		RaceEnabled:   raceEnabled,
		Scenarios:     scenarios,
		Passed:        len(scenarios) > 0,
	}
	for _, scenario := range scenarios {
		if !scenario.Passed {
			report.Passed = false
		}
	}
	return report
}

func WriteReport(path string, report Report) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func IterationsFromEnv(name string, fallback int) int {
	if fallback < 1 {
		fallback = 1
	}
	var value int
	if _, err := fmt.Sscanf(os.Getenv(name), "%d", &value); err != nil || value < 1 {
		return fallback
	}
	return value
}

func BoolFromEnv(name string) bool {
	return os.Getenv(name) == "1" || os.Getenv(name) == "true"
}

func (s *Scenario) evaluate() {
	if s.P95Milliseconds > s.Thresholds.MaxP95Milliseconds {
		s.Alerts = append(s.Alerts, fmt.Sprintf("P95 %.3fms exceeds %.3fms", s.P95Milliseconds, s.Thresholds.MaxP95Milliseconds))
	}
	if s.P99Milliseconds > s.Thresholds.MaxP99Milliseconds {
		s.Alerts = append(s.Alerts, fmt.Sprintf("P99 %.3fms exceeds %.3fms", s.P99Milliseconds, s.Thresholds.MaxP99Milliseconds))
	}
	if s.HeapGrowthBytes > s.Thresholds.MaxHeapGrowthBytes {
		s.Alerts = append(s.Alerts, fmt.Sprintf("heap growth %d exceeds %d bytes", s.HeapGrowthBytes, s.Thresholds.MaxHeapGrowthBytes))
	}
	if s.GoroutineGrowth > s.Thresholds.MaxGoroutineGrowth {
		s.Alerts = append(s.Alerts, fmt.Sprintf("goroutine growth %d exceeds %d", s.GoroutineGrowth, s.Thresholds.MaxGoroutineGrowth))
	}
	if s.Failures > s.Thresholds.MaxFailures {
		s.Alerts = append(s.Alerts, fmt.Sprintf("failures %d exceeds %d", s.Failures, s.Thresholds.MaxFailures))
	}
	if s.DuplicateSideEffects > s.Thresholds.MaxDuplicateSideEffects {
		s.Alerts = append(s.Alerts, fmt.Sprintf("duplicate side effects %d exceeds %d", s.DuplicateSideEffects, s.Thresholds.MaxDuplicateSideEffects))
	}
	s.Passed = len(s.Alerts) == 0
}

func percentileMilliseconds(values []time.Duration, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	ordered := append([]time.Duration(nil), values...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	index := int(float64(len(ordered)-1) * percentile)
	return float64(ordered[index]) / float64(time.Millisecond)
}
