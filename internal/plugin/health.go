package plugin

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

type HealthStatus string

const (
	HealthStatusHealthy       HealthStatus = "healthy"
	HealthStatusUnhealthy     HealthStatus = "unhealthy"
	HealthStatusNotApplicable HealthStatus = "not_applicable"
)

type HealthReport struct {
	PluginID      string       `json:"pluginId"`
	Status        HealthStatus `json:"status"`
	Code          string       `json:"code"`
	CheckedAt     time.Time    `json:"checkedAt"`
	LatencyMillis int64        `json:"latencyMillis"`
	HTTPStatus    int          `json:"httpStatus,omitempty"`
}

func (r HealthReport) Ready() bool {
	return r.Status == HealthStatusHealthy || r.Status == HealthStatusNotApplicable
}

type ReadinessReport struct {
	Ready     bool           `json:"ready"`
	CheckedAt time.Time      `json:"checkedAt"`
	Plugins   []HealthReport `json:"plugins"`
}

type HealthChecker interface {
	Check(ctx context.Context, info Info) HealthReport
}

type HealthProvider interface {
	CheckPluginHealth(ctx context.Context, pluginID string) (HealthReport, error)
	PluginReadiness(ctx context.Context) ReadinessReport
}

type HTTPHealthChecker struct {
	client *http.Client
	now    func() time.Time
}

func NewHTTPHealthChecker(timeout time.Duration) *HTTPHealthChecker {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &HTTPHealthChecker{
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		now: time.Now,
	}
}

func (c *HTTPHealthChecker) Check(ctx context.Context, info Info) HealthReport {
	now := time.Now
	if c != nil && c.now != nil {
		now = c.now
	}
	started := now().UTC()
	report := HealthReport{
		PluginID:  strings.TrimSpace(info.ID),
		Status:    HealthStatusUnhealthy,
		Code:      "health_unavailable",
		CheckedAt: started,
	}
	finish := func() HealthReport {
		report.LatencyMillis = max(now().UTC().Sub(started).Milliseconds(), 0)
		return report
	}

	if info.State != StateEnabled {
		report.Code = "plugin_not_enabled"
		return finish()
	}
	baseURL := strings.TrimSpace(info.ServiceBaseURL)
	healthURL := strings.TrimSpace(info.ServiceHealthURL)
	if baseURL == "" && healthURL == "" {
		report.Status = HealthStatusNotApplicable
		report.Code = "health_not_applicable"
		return finish()
	}
	if baseURL == "" || healthURL == "" {
		report.Code = "health_not_configured"
		return finish()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		report.Code = "health_invalid_url"
		return finish()
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Skoll-Plugin-Health/1")

	client := (*http.Client)(nil)
	if c != nil {
		client = c.client
	}
	if client == nil {
		report.Code = "health_checker_unavailable"
		return finish()
	}
	resp, err := client.Do(req)
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			report.Code = "health_cancelled"
		case errors.Is(err, context.DeadlineExceeded):
			report.Code = "health_timeout"
		default:
			report.Code = "health_unreachable"
		}
		return finish()
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))

	report.HTTPStatus = resp.StatusCode
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		report.Status = HealthStatusHealthy
		report.Code = "health_ok"
		return finish()
	}
	report.Code = "health_http_status"
	return finish()
}
