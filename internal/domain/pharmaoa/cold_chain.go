package pharmaoa

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type ColdChainAnomalyStatus string
type ColdChainRisk string
type ColdChainJobStatus string

const (
	ColdChainAnomalyActive   ColdChainAnomalyStatus = "active"
	ColdChainAnomalyResolved ColdChainAnomalyStatus = "resolved"

	ColdChainRiskMedium ColdChainRisk = "medium"
	ColdChainRiskHigh   ColdChainRisk = "high"

	ColdChainJobPending   ColdChainJobStatus = "pending"
	ColdChainJobRunning   ColdChainJobStatus = "running"
	ColdChainJobSucceeded ColdChainJobStatus = "succeeded"
	ColdChainJobFailed    ColdChainJobStatus = "failed"
)

type ColdChainRecord struct {
	ID                 shared.ID `json:"id"`
	BalanceID          string    `json:"balanceId"`
	ProductID          string    `json:"productId"`
	BatchID            string    `json:"batchId"`
	BatchNo            string    `json:"batchNo"`
	WarehouseID        string    `json:"warehouseId"`
	AreaID             string    `json:"areaId"`
	LocationID         string    `json:"locationId"`
	TemperatureCelsius float64   `json:"temperatureCelsius"`
	HumidityPercent    float64   `json:"humidityPercent"`
	MinCelsius         float64   `json:"minCelsius"`
	MaxCelsius         float64   `json:"maxCelsius"`
	Source             string    `json:"source"`
	RecordedBy         string    `json:"recordedBy"`
	RecordedAt         time.Time `json:"recordedAt"`
	CreatedAt          time.Time `json:"createdAt"`
}

type ColdChainRecordInput struct {
	BalanceID          string
	ProductID          string
	BatchID            string
	BatchNo            string
	WarehouseID        string
	AreaID             string
	LocationID         string
	TemperatureCelsius float64
	HumidityPercent    float64
	MinCelsius         float64
	MaxCelsius         float64
	Source             string
	RecordedBy         string
	RecordedAt         time.Time
}

func NewColdChainRecord(id shared.ID, in ColdChainRecordInput, now time.Time) (*ColdChainRecord, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("cold-chain record id is required")
	}
	for field, value := range map[string]string{
		"balanceId": in.BalanceID, "productId": in.ProductID, "batchId": in.BatchID,
		"batchNo": in.BatchNo, "warehouseId": in.WarehouseID, "areaId": in.AreaID,
		"locationId": in.LocationID, "source": in.Source, "recordedBy": in.RecordedBy,
	} {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("%s is required", field)
		}
	}
	if in.RecordedAt.IsZero() {
		return nil, fmt.Errorf("recordedAt is required")
	}
	if math.IsNaN(in.TemperatureCelsius) || math.IsInf(in.TemperatureCelsius, 0) || in.TemperatureCelsius < -100 || in.TemperatureCelsius > 100 {
		return nil, fmt.Errorf("temperatureCelsius must be between -100 and 100")
	}
	if math.IsNaN(in.HumidityPercent) || math.IsInf(in.HumidityPercent, 0) || in.HumidityPercent < 0 || in.HumidityPercent > 100 {
		return nil, fmt.Errorf("humidityPercent must be between 0 and 100")
	}
	if in.MinCelsius > in.MaxCelsius {
		return nil, fmt.Errorf("minCelsius cannot exceed maxCelsius")
	}
	return &ColdChainRecord{
		ID: id, BalanceID: strings.TrimSpace(in.BalanceID), ProductID: strings.TrimSpace(in.ProductID),
		BatchID: strings.TrimSpace(in.BatchID), BatchNo: strings.TrimSpace(in.BatchNo), WarehouseID: strings.TrimSpace(in.WarehouseID),
		AreaID: strings.TrimSpace(in.AreaID), LocationID: strings.TrimSpace(in.LocationID), TemperatureCelsius: in.TemperatureCelsius,
		HumidityPercent: in.HumidityPercent, MinCelsius: in.MinCelsius, MaxCelsius: in.MaxCelsius, Source: strings.TrimSpace(in.Source),
		RecordedBy: strings.TrimSpace(in.RecordedBy), RecordedAt: in.RecordedAt.UTC(), CreatedAt: now.UTC(),
	}, nil
}

type ColdChainScanPolicy struct {
	MinHumidityPercent float64 `json:"minHumidityPercent"`
	MaxHumidityPercent float64 `json:"maxHumidityPercent"`
	RecipientID        string  `json:"recipientId"`
}

func (p ColdChainScanPolicy) Validate() error {
	if math.IsNaN(p.MinHumidityPercent) || math.IsNaN(p.MaxHumidityPercent) || p.MinHumidityPercent < 0 || p.MaxHumidityPercent > 100 || p.MinHumidityPercent > p.MaxHumidityPercent {
		return fmt.Errorf("cold-chain humidity policy is invalid")
	}
	if strings.TrimSpace(p.RecipientID) == "" {
		return fmt.Errorf("cold-chain recipientId is required")
	}
	return nil
}

type ColdChainAnomaly struct {
	ID                 shared.ID              `json:"id"`
	Status             ColdChainAnomalyStatus `json:"status"`
	Risk               ColdChainRisk          `json:"risk"`
	RecordID           string                 `json:"recordId"`
	BalanceID          string                 `json:"balanceId"`
	ProductID          string                 `json:"productId"`
	BatchID            string                 `json:"batchId"`
	BatchNo            string                 `json:"batchNo"`
	WarehouseID        string                 `json:"warehouseId"`
	AreaID             string                 `json:"areaId"`
	LocationID         string                 `json:"locationId"`
	TemperatureCelsius float64                `json:"temperatureCelsius"`
	HumidityPercent    float64                `json:"humidityPercent"`
	MinCelsius         float64                `json:"minCelsius"`
	MaxCelsius         float64                `json:"maxCelsius"`
	MinHumidityPercent float64                `json:"minHumidityPercent"`
	MaxHumidityPercent float64                `json:"maxHumidityPercent"`
	Reasons            []string               `json:"reasons"`
	RecipientID        string                 `json:"recipientId"`
	NotificationID     string                 `json:"notificationId"`
	TargetPath         string                 `json:"targetPath"`
	FirstSeenAt        time.Time              `json:"firstSeenAt"`
	LastSeenAt         time.Time              `json:"lastSeenAt"`
	ResolvedAt         *time.Time             `json:"resolvedAt,omitempty"`
}

type ColdChainJobLog struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

type ColdChainJob struct {
	ID            shared.ID           `json:"id"`
	Status        ColdChainJobStatus  `json:"status"`
	Policy        ColdChainScanPolicy `json:"policy"`
	InitiatedBy   string              `json:"initiatedBy"`
	LastRunBy     string              `json:"lastRunBy"`
	RetryCount    int                 `json:"retryCount"`
	MatchedCount  int                 `json:"matchedCount"`
	CreatedCount  int                 `json:"createdCount"`
	ResolvedCount int                 `json:"resolvedCount"`
	Error         string              `json:"error,omitempty"`
	Logs          []ColdChainJobLog   `json:"logs"`
	CreatedAt     time.Time           `json:"createdAt"`
	StartedAt     *time.Time          `json:"startedAt,omitempty"`
	CompletedAt   *time.Time          `json:"completedAt,omitempty"`
}

func NewColdChainJob(id shared.ID, policy ColdChainScanPolicy, actorID string, now time.Time) (*ColdChainJob, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("cold-chain job id is required")
	}
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return nil, fmt.Errorf("cold-chain job actor is required")
	}
	now = now.UTC()
	return &ColdChainJob{ID: id, Status: ColdChainJobPending, Policy: policy, InitiatedBy: actorID, LastRunBy: actorID, Logs: []ColdChainJobLog{{Level: "info", Message: "job queued", CreatedAt: now}}, CreatedAt: now}, nil
}

func (j *ColdChainJob) Start(actorID string, now time.Time) error {
	if j == nil || (j.Status != ColdChainJobPending && j.Status != ColdChainJobFailed) {
		return fmt.Errorf("cold-chain job cannot start")
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return fmt.Errorf("cold-chain job actor is required")
	}
	now = now.UTC()
	j.Status = ColdChainJobRunning
	j.LastRunBy = actorID
	j.Error = ""
	j.StartedAt = &now
	j.CompletedAt = nil
	j.Logs = append(j.Logs, ColdChainJobLog{Level: "info", Message: "scan started", CreatedAt: now})
	return nil
}

func (j *ColdChainJob) Succeed(matched, created, resolved int, now time.Time) error {
	if j == nil || j.Status != ColdChainJobRunning {
		return fmt.Errorf("cold-chain job is not running")
	}
	now = now.UTC()
	j.Status = ColdChainJobSucceeded
	j.MatchedCount = matched
	j.CreatedCount = created
	j.ResolvedCount = resolved
	j.CompletedAt = &now
	j.Logs = append(j.Logs, ColdChainJobLog{Level: "info", Message: "scan completed", CreatedAt: now})
	return nil
}

func (j *ColdChainJob) Fail(err error, now time.Time) {
	if j == nil {
		return
	}
	now = now.UTC()
	j.Status = ColdChainJobFailed
	if err != nil {
		j.Error = err.Error()
	}
	j.CompletedAt = &now
	j.Logs = append(j.Logs, ColdChainJobLog{Level: "error", Message: j.Error, CreatedAt: now})
}
