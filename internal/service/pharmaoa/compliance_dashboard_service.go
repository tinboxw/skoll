package pharmaoa

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
)

type ComplianceRisk string
type ComplianceSource string

const (
	ComplianceRiskHigh   ComplianceRisk = "high"
	ComplianceRiskMedium ComplianceRisk = "medium"

	ComplianceSourceQualification    ComplianceSource = "qualification"
	ComplianceSourceQualityComplaint ComplianceSource = "quality_complaint"
	ComplianceSourceDrugRecall       ComplianceSource = "drug_recall"
	ComplianceSourceColdChain        ComplianceSource = "cold_chain"
)

type ComplianceQualificationReader interface {
	List(ctx context.Context, in QualificationListInput) ([]QualificationRecord, error)
}

type ComplianceComplaintReader interface {
	List(ctx context.Context, in QualityComplaintListInput) ([]*domainpharma.QualityComplaint, error)
}

type ComplianceRecallReader interface {
	List(ctx context.Context, in DrugRecallListInput) ([]*domainpharma.DrugRecall, error)
}

type ComplianceColdChainReader interface {
	ListAnomalies(ctx context.Context, activeOnly bool) ([]*domainpharma.ColdChainAnomaly, error)
}

type ComplianceDashboardService interface {
	Get(ctx context.Context, in ComplianceDashboardInput) (*ComplianceDashboardSnapshot, error)
	Export(ctx context.Context, in ComplianceDashboardInput) ([]byte, error)
}

type ComplianceDashboardInput struct {
	Keyword string
	Risk    ComplianceRisk
	Source  ComplianceSource
	Limit   int
	ActorID string
}

type ComplianceTraceEntry struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type ComplianceRiskItem struct {
	ID             string                 `json:"id"`
	Source         ComplianceSource       `json:"source"`
	SourceID       string                 `json:"sourceId"`
	Risk           ComplianceRisk         `json:"risk"`
	Status         string                 `json:"status"`
	Reference      string                 `json:"reference"`
	Title          string                 `json:"title"`
	Subject        string                 `json:"subject"`
	BatchID        string                 `json:"batchId,omitempty"`
	BatchNo        string                 `json:"batchNo,omitempty"`
	TargetPath     string                 `json:"targetPath"`
	ObservedAt     time.Time              `json:"observedAt"`
	PendingActions int                    `json:"pendingActions"`
	TotalActions   int                    `json:"totalActions"`
	Trace          []ComplianceTraceEntry `json:"trace"`
}

type ComplianceDashboardSummary struct {
	Total          int `json:"total"`
	High           int `json:"high"`
	Medium         int `json:"medium"`
	Qualifications int `json:"qualifications"`
	Complaints     int `json:"complaints"`
	Recalls        int `json:"recalls"`
	ColdChain      int `json:"coldChain"`
}

type ComplianceDashboardSnapshot struct {
	Summary      ComplianceDashboardSummary `json:"summary"`
	Items        []ComplianceRiskItem       `json:"items"`
	MatchedCount int                        `json:"matchedCount"`
	GeneratedAt  time.Time                  `json:"generatedAt"`
}

type complianceDashboardService struct {
	qualifications ComplianceQualificationReader
	complaints     ComplianceComplaintReader
	recalls        ComplianceRecallReader
	coldChain      ComplianceColdChainReader
	audit          auditsvc.Service
	nowFn          func() time.Time
}

func NewComplianceDashboardService(qualifications ComplianceQualificationReader, complaints ComplianceComplaintReader, recalls ComplianceRecallReader, coldChain ComplianceColdChainReader, audit auditsvc.Service) ComplianceDashboardService {
	return &complianceDashboardService{qualifications: qualifications, complaints: complaints, recalls: recalls, coldChain: coldChain, audit: audit, nowFn: func() time.Time { return time.Now().UTC() }}
}

func (s *complianceDashboardService) Get(ctx context.Context, in ComplianceDashboardInput) (*ComplianceDashboardSnapshot, error) {
	if err := validateComplianceDashboardInput(&in); err != nil {
		return nil, err
	}
	snapshot, err := s.collect(ctx, in)
	if err != nil {
		return nil, err
	}
	if err = s.appendAudit(ctx, in, "pharma_oa.compliance_dashboard.view", snapshot); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (s *complianceDashboardService) Export(ctx context.Context, in ComplianceDashboardInput) ([]byte, error) {
	if err := validateComplianceDashboardInput(&in); err != nil {
		return nil, err
	}
	snapshot, err := s.collect(ctx, in)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	out.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(&out)
	if err = writer.Write([]string{"Risk", "Source", "Reference", "Title", "Subject", "Batch", "Status", "Observed at", "Pending actions", "Target path"}); err != nil {
		return nil, err
	}
	for _, item := range snapshot.Items {
		row := []string{string(item.Risk), string(item.Source), item.Reference, item.Title, item.Subject, item.BatchNo, item.Status, item.ObservedAt.Format(time.RFC3339), strconv.Itoa(item.PendingActions), item.TargetPath}
		for index := range row {
			row[index] = complianceCSVCell(row[index])
		}
		if err = writer.Write(row); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err = writer.Error(); err != nil {
		return nil, err
	}
	if err = s.appendAudit(ctx, in, "pharma_oa.compliance_dashboard.export", snapshot); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (s *complianceDashboardService) collect(ctx context.Context, in ComplianceDashboardInput) (*ComplianceDashboardSnapshot, error) {
	if s == nil || s.qualifications == nil || s.complaints == nil || s.recalls == nil || s.coldChain == nil || s.audit == nil {
		return nil, fmt.Errorf("compliance dashboard dependencies are required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	items := make([]ComplianceRiskItem, 0)
	qualifications, err := s.qualifications.List(ctx, QualificationListInput{Days: 30})
	if err != nil {
		return nil, fmt.Errorf("list compliance qualifications: %w", err)
	}
	for _, item := range qualifications {
		if item.Status != QualificationStatusExpired && item.Status != QualificationStatusExpiring {
			continue
		}
		risk := ComplianceRiskMedium
		title := "Qualification expires soon"
		if item.Status == QualificationStatusExpired {
			risk, title = ComplianceRiskHigh, "Qualification expired"
		}
		items = append(items, ComplianceRiskItem{
			ID: "qualification:" + item.ID, Source: ComplianceSourceQualification, SourceID: item.ID, Risk: risk, Status: string(item.Status), Reference: complianceReference(item.QualificationName, item.Number), Title: title,
			Subject: complianceReference(item.SubjectCode, item.SubjectName), TargetPath: item.TargetPath, ObservedAt: item.ExpiresAt, PendingActions: 1, TotalActions: 1,
			Trace: []ComplianceTraceEntry{{Label: "Subject type", Value: string(item.SubjectType)}, {Label: "Qualification", Value: item.QualificationName}, {Label: "Number", Value: item.Number}, {Label: "Expires", Value: item.ExpiresAt.Format("2006-01-02")}},
		})
	}
	complaints, err := s.complaints.List(ctx, QualityComplaintListInput{Status: domainpharma.QualityComplaintPending})
	if err != nil {
		return nil, fmt.Errorf("list compliance complaints: %w", err)
	}
	for _, item := range complaints {
		if item == nil || item.Status != domainpharma.QualityComplaintPending {
			continue
		}
		items = append(items, ComplianceRiskItem{
			ID: "quality_complaint:" + item.ID.String(), Source: ComplianceSourceQualityComplaint, SourceID: item.ID.String(), Risk: ComplianceRiskHigh, Status: string(item.Status), Reference: item.Number, Title: item.Title,
			Subject: complianceReference(item.CustomerName, item.ProductName), BatchID: item.BatchID, BatchNo: item.BatchNo, TargetPath: "/skoll/pharma-oa/quality-complaints?complaintId=" + url.QueryEscape(item.ID.String()), ObservedAt: item.Meta.CreatedAt, PendingActions: 1, TotalActions: 1,
			Trace: []ComplianceTraceEntry{{Label: "Customer", Value: item.CustomerName}, {Label: "Product", Value: item.ProductName}, {Label: "Batch", Value: item.BatchNo}, {Label: "Workflow", Value: item.WorkflowInstanceID}},
		})
	}
	recalls, err := s.recalls.List(ctx, DrugRecallListInput{Status: domainpharma.DrugRecallActive})
	if err != nil {
		return nil, fmt.Errorf("list compliance recalls: %w", err)
	}
	for _, item := range recalls {
		if item == nil || item.Status != domainpharma.DrugRecallActive {
			continue
		}
		pending := 0
		for _, task := range item.Tasks {
			if task.Status == domainpharma.DrugRecallTaskPending {
				pending++
			}
		}
		items = append(items, ComplianceRiskItem{
			ID: "drug_recall:" + item.ID.String(), Source: ComplianceSourceDrugRecall, SourceID: item.ID.String(), Risk: ComplianceRiskHigh, Status: string(item.Status), Reference: item.Number, Title: item.Title,
			Subject: item.ProductName, BatchID: item.BatchID, BatchNo: item.BatchNo, TargetPath: "/skoll/pharma-oa/drug-recalls?recallId=" + url.QueryEscape(item.ID.String()), ObservedAt: item.InitiatedAt, PendingActions: pending, TotalActions: len(item.Tasks),
			Trace: []ComplianceTraceEntry{{Label: "Product", Value: item.ProductName}, {Label: "Batch", Value: item.BatchNo}, {Label: "Pending customer tasks", Value: fmt.Sprintf("%d of %d", pending, len(item.Tasks))}, {Label: "Source complaint", Value: item.SourceComplaintID}},
		})
	}
	anomalies, err := s.coldChain.ListAnomalies(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("list compliance cold-chain anomalies: %w", err)
	}
	for _, item := range anomalies {
		if item == nil || item.Status != domainpharma.ColdChainAnomalyActive {
			continue
		}
		risk := ComplianceRisk(item.Risk)
		if risk != ComplianceRiskHigh && risk != ComplianceRiskMedium {
			risk = ComplianceRiskMedium
		}
		items = append(items, ComplianceRiskItem{
			ID: "cold_chain:" + item.ID.String(), Source: ComplianceSourceColdChain, SourceID: item.ID.String(), Risk: risk, Status: string(item.Status), Reference: item.BatchNo, Title: "Cold-chain limit excursion",
			Subject: complianceReference(item.ProductID, complianceReference(item.WarehouseID, item.LocationID)), BatchID: item.BatchID, BatchNo: item.BatchNo, TargetPath: item.TargetPath, ObservedAt: item.LastSeenAt, PendingActions: 1, TotalActions: 1,
			Trace: []ComplianceTraceEntry{{Label: "Temperature", Value: fmt.Sprintf("%.1f C (%.1f to %.1f C)", item.TemperatureCelsius, item.MinCelsius, item.MaxCelsius)}, {Label: "Humidity", Value: fmt.Sprintf("%.1f%% (%.1f to %.1f%%)", item.HumidityPercent, item.MinHumidityPercent, item.MaxHumidityPercent)}, {Label: "Location", Value: complianceReference(item.WarehouseID, complianceReference(item.AreaID, item.LocationID))}, {Label: "Reasons", Value: strings.Join(item.Reasons, ", ")}},
		})
	}

	summary := complianceDashboardSummary(items)
	filtered := filterComplianceItems(items, in)
	matched := len(filtered)
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].Risk != filtered[j].Risk {
			return filtered[i].Risk == ComplianceRiskHigh
		}
		if !filtered[i].ObservedAt.Equal(filtered[j].ObservedAt) {
			return filtered[i].ObservedAt.Before(filtered[j].ObservedAt)
		}
		return filtered[i].ID < filtered[j].ID
	})
	if len(filtered) > in.Limit {
		filtered = filtered[:in.Limit]
	}
	return &ComplianceDashboardSnapshot{Summary: summary, Items: filtered, MatchedCount: matched, GeneratedAt: s.nowFn().UTC()}, nil
}

func validateComplianceDashboardInput(in *ComplianceDashboardInput) error {
	in.ActorID = strings.TrimSpace(in.ActorID)
	in.Keyword = strings.TrimSpace(in.Keyword)
	if in.ActorID == "" {
		return fmt.Errorf("compliance dashboard actor is required")
	}
	if in.Risk != "" && in.Risk != ComplianceRiskHigh && in.Risk != ComplianceRiskMedium {
		return fmt.Errorf("invalid compliance risk: %s", in.Risk)
	}
	if in.Source != "" && in.Source != ComplianceSourceQualification && in.Source != ComplianceSourceQualityComplaint && in.Source != ComplianceSourceDrugRecall && in.Source != ComplianceSourceColdChain {
		return fmt.Errorf("invalid compliance source: %s", in.Source)
	}
	if in.Limit == 0 {
		in.Limit = 200
	}
	if in.Limit < 1 || in.Limit > 500 {
		return fmt.Errorf("limit must be between 1 and 500")
	}
	return nil
}

func filterComplianceItems(items []ComplianceRiskItem, in ComplianceDashboardInput) []ComplianceRiskItem {
	keyword := strings.ToLower(in.Keyword)
	out := make([]ComplianceRiskItem, 0, len(items))
	for _, item := range items {
		if in.Risk != "" && item.Risk != in.Risk || in.Source != "" && item.Source != in.Source {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(strings.Join([]string{item.Reference, item.Title, item.Subject, item.BatchNo, item.Status, string(item.Source), string(item.Risk)}, " ")), keyword) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func complianceDashboardSummary(items []ComplianceRiskItem) ComplianceDashboardSummary {
	summary := ComplianceDashboardSummary{Total: len(items)}
	for _, item := range items {
		if item.Risk == ComplianceRiskHigh {
			summary.High++
		} else {
			summary.Medium++
		}
		switch item.Source {
		case ComplianceSourceQualification:
			summary.Qualifications++
		case ComplianceSourceQualityComplaint:
			summary.Complaints++
		case ComplianceSourceDrugRecall:
			summary.Recalls++
		case ComplianceSourceColdChain:
			summary.ColdChain++
		}
	}
	return summary
}

func (s *complianceDashboardService) appendAudit(ctx context.Context, in ComplianceDashboardInput, action string, snapshot *ComplianceDashboardSnapshot) error {
	_, err := s.audit.Append(ctx, in.ActorID, action, "pharma_oa_compliance_dashboard", "active-risks", map[string]any{
		"keyword": in.Keyword, "risk": in.Risk, "source": in.Source, "limit": in.Limit,
		"active": snapshot.Summary.Total, "matched": snapshot.MatchedCount, "returned": len(snapshot.Items), "generatedAt": snapshot.GeneratedAt,
	})
	if err != nil {
		return fmt.Errorf("append compliance dashboard audit: %w", err)
	}
	return nil
}

func complianceReference(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			out = append(out, value)
		}
	}
	return strings.Join(out, " / ")
}

func complianceCSVCell(value string) string {
	value = strings.TrimSpace(value)
	if value != "" && strings.ContainsRune("=+-@", rune(value[0])) {
		return "'" + value
	}
	return value
}
