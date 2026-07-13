package pharmaoa

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type InventoryAlertType string
type InventoryAlertStatus string
type InventoryAlertJobStatus string

const (
	InventoryAlertNearExpiry InventoryAlertType = "near_expiry"
	InventoryAlertLowStock   InventoryAlertType = "low_stock"
	InventoryAlertOverStock  InventoryAlertType = "over_stock"

	InventoryAlertActive   InventoryAlertStatus = "active"
	InventoryAlertResolved InventoryAlertStatus = "resolved"

	InventoryAlertJobPending   InventoryAlertJobStatus = "pending"
	InventoryAlertJobRunning   InventoryAlertJobStatus = "running"
	InventoryAlertJobSucceeded InventoryAlertJobStatus = "succeeded"
	InventoryAlertJobFailed    InventoryAlertJobStatus = "failed"
)

type InventoryAlertPolicy struct {
	NearExpiryDays     int    `json:"nearExpiryDays"`
	LowStockThreshold  int    `json:"lowStockThreshold"`
	OverStockThreshold int    `json:"overStockThreshold"`
	RecipientID        string `json:"recipientId"`
}

func (p InventoryAlertPolicy) Validate() error {
	if p.NearExpiryDays <= 0 || p.LowStockThreshold < 0 || p.OverStockThreshold <= 0 || p.LowStockThreshold >= p.OverStockThreshold || strings.TrimSpace(p.RecipientID) == "" {
		return fmt.Errorf("inventory alert policy is invalid")
	}
	return nil
}

type InventoryAlert struct {
	ID             shared.ID            `json:"id"`
	Type           InventoryAlertType   `json:"type"`
	Status         InventoryAlertStatus `json:"status"`
	BalanceID      string               `json:"balanceId"`
	ProductID      string               `json:"productId"`
	WarehouseID    string               `json:"warehouseId"`
	AreaID         string               `json:"areaId"`
	LocationID     string               `json:"locationId"`
	BatchID        string               `json:"batchId"`
	Quantity       int                  `json:"quantity"`
	Threshold      int                  `json:"threshold"`
	ExpiresAt      time.Time            `json:"expiresAt,omitempty"`
	RecipientID    string               `json:"recipientId"`
	NotificationID string               `json:"notificationId"`
	TargetPath     string               `json:"targetPath"`
	FirstSeenAt    time.Time            `json:"firstSeenAt"`
	LastSeenAt     time.Time            `json:"lastSeenAt"`
	ResolvedAt     *time.Time           `json:"resolvedAt,omitempty"`
}

type InventoryAlertJobLog struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

type InventoryAlertJob struct {
	ID            shared.ID               `json:"id"`
	Status        InventoryAlertJobStatus `json:"status"`
	Policy        InventoryAlertPolicy    `json:"policy"`
	RetryCount    int                     `json:"retryCount"`
	MatchedCount  int                     `json:"matchedCount"`
	CreatedCount  int                     `json:"createdCount"`
	ResolvedCount int                     `json:"resolvedCount"`
	Error         string                  `json:"error,omitempty"`
	Logs          []InventoryAlertJobLog  `json:"logs"`
	CreatedAt     time.Time               `json:"createdAt"`
	StartedAt     *time.Time              `json:"startedAt,omitempty"`
	CompletedAt   *time.Time              `json:"completedAt,omitempty"`
}

func NewInventoryAlertJob(id shared.ID, policy InventoryAlertPolicy, now time.Time) (*InventoryAlertJob, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("inventory alert job id is required")
	}
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	now = now.UTC()
	return &InventoryAlertJob{ID: id, Status: InventoryAlertJobPending, Policy: policy, Logs: []InventoryAlertJobLog{{Level: "info", Message: "job queued", CreatedAt: now}}, CreatedAt: now}, nil
}

func (j *InventoryAlertJob) Start(now time.Time) error {
	if j == nil || (j.Status != InventoryAlertJobPending && j.Status != InventoryAlertJobFailed) {
		return fmt.Errorf("inventory alert job cannot start")
	}
	now = now.UTC()
	j.Status = InventoryAlertJobRunning
	j.Error = ""
	j.StartedAt = &now
	j.CompletedAt = nil
	j.Logs = append(j.Logs, InventoryAlertJobLog{Level: "info", Message: "scan started", CreatedAt: now})
	return nil
}

func (j *InventoryAlertJob) Succeed(matched, created, resolved int, now time.Time) error {
	if j == nil || j.Status != InventoryAlertJobRunning {
		return fmt.Errorf("inventory alert job is not running")
	}
	now = now.UTC()
	j.Status = InventoryAlertJobSucceeded
	j.MatchedCount = matched
	j.CreatedCount = created
	j.ResolvedCount = resolved
	j.CompletedAt = &now
	j.Logs = append(j.Logs, InventoryAlertJobLog{Level: "info", Message: "scan completed", CreatedAt: now})
	return nil
}

func (j *InventoryAlertJob) Fail(err error, now time.Time) {
	if j == nil {
		return
	}
	now = now.UTC()
	j.Status = InventoryAlertJobFailed
	if err != nil {
		j.Error = err.Error()
	}
	j.CompletedAt = &now
	j.Logs = append(j.Logs, InventoryAlertJobLog{Level: "error", Message: j.Error, CreatedAt: now})
}
