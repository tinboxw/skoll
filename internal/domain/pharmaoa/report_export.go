package pharmaoa

import "time"

type ReportExportType string
type ReportExportJobStatus string

const (
	ReportExportBusinessMetrics  ReportExportType = "business_metrics"
	ReportExportSalesTrend       ReportExportType = "sales_trend"
	ReportExportOperationalRisks ReportExportType = "operational_risks"

	ReportExportPending   ReportExportJobStatus = "pending"
	ReportExportRunning   ReportExportJobStatus = "running"
	ReportExportSucceeded ReportExportJobStatus = "succeeded"
	ReportExportFailed    ReportExportJobStatus = "failed"
)

type ReportExportQuery struct {
	From              time.Time `json:"from"`
	To                time.Time `json:"to"`
	Bucket            string    `json:"bucket"`
	QualificationDays int       `json:"qualificationDays"`
}

type ReportExportJobLog struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

type ReportExportJob struct {
	ID             string                `json:"id"`
	ReportType     ReportExportType      `json:"reportType"`
	Status         ReportExportJobStatus `json:"status"`
	Query          ReportExportQuery     `json:"query"`
	OwnerID        string                `json:"ownerId"`
	IdempotencyKey string                `json:"idempotencyKey"`
	FileID         string                `json:"fileId"`
	Filename       string                `json:"filename"`
	ContentType    string                `json:"contentType"`
	Size           int64                 `json:"size"`
	RowCount       int                   `json:"rowCount"`
	Error          string                `json:"error"`
	RetryCount     int                   `json:"retryCount"`
	Logs           []ReportExportJobLog  `json:"logs"`
	CreatedAt      time.Time             `json:"createdAt"`
	StartedAt      *time.Time            `json:"startedAt,omitempty"`
	CompletedAt    *time.Time            `json:"completedAt,omitempty"`
}
