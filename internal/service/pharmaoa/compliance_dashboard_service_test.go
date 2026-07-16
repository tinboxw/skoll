package pharmaoa

import (
	"context"
	"strings"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
)

type complianceQualificationFixture struct{ items []QualificationRecord }

func (f complianceQualificationFixture) List(context.Context, QualificationListInput) ([]QualificationRecord, error) {
	return append([]QualificationRecord(nil), f.items...), nil
}

type complianceComplaintFixture struct {
	items []*domainpharma.QualityComplaint
}

func (f complianceComplaintFixture) List(context.Context, QualityComplaintListInput) ([]*domainpharma.QualityComplaint, error) {
	return append([]*domainpharma.QualityComplaint(nil), f.items...), nil
}

type complianceRecallFixture struct{ items []*domainpharma.DrugRecall }

func (f complianceRecallFixture) List(context.Context, DrugRecallListInput) ([]*domainpharma.DrugRecall, error) {
	return append([]*domainpharma.DrugRecall(nil), f.items...), nil
}

type complianceColdChainFixture struct {
	items []*domainpharma.ColdChainAnomaly
}

func (f complianceColdChainFixture) ListAnomalies(context.Context, bool) ([]*domainpharma.ColdChainAnomaly, error) {
	return append([]*domainpharma.ColdChainAnomaly(nil), f.items...), nil
}

func TestComplianceDashboardAggregatesFiltersExportsAndAudits(t *testing.T) {
	now := time.Date(2026, 7, 13, 9, 0, 0, 0, time.UTC)
	audit := auditsvc.NewService(clickhouse.NewAuditStore())
	complaintMeta := shared.AuditMeta{CreatedAt: now.Add(-72 * time.Hour), UpdatedAt: now.Add(-48 * time.Hour)}
	service := NewComplianceDashboardService(
		complianceQualificationFixture{items: []QualificationRecord{
			{ID: "qualification-expired", SubjectType: QualificationSubjectSupplier, SubjectCode: "SUP-1", SubjectName: "Supplier One", QualificationName: "GSP", Number: "=FORMULA", ExpiresAt: now.Add(-96 * time.Hour), Status: QualificationStatusExpired, TargetPath: "/skoll/pharma-oa/suppliers?supplierId=supplier-1"},
			{ID: "qualification-expiring", SubjectType: QualificationSubjectCustomer, SubjectCode: "CUS-1", SubjectName: "Hospital One", QualificationName: "License", Number: "LIC-1", ExpiresAt: now.Add(48 * time.Hour), Status: QualificationStatusExpiring, TargetPath: "/skoll/pharma-oa/customers?customerId=customer-1"},
			{ID: "qualification-valid", Status: QualificationStatusValid},
		}},
		complianceComplaintFixture{items: []*domainpharma.QualityComplaint{{ID: "complaint-1", Number: "QC-1", Title: "Visible particles", CustomerName: "Hospital One", ProductName: "Drug One", BatchID: "batch-1", BatchNo: "B-001", WorkflowInstanceID: "workflow-1", Status: domainpharma.QualityComplaintPending, Meta: complaintMeta}, {ID: "complaint-done", Status: domainpharma.QualityComplaintResolved}}},
		complianceRecallFixture{items: []*domainpharma.DrugRecall{{ID: "recall-1", Number: "RC-1", Title: "Batch recall", ProductName: "Drug One", BatchID: "batch-1", BatchNo: "B-001", Status: domainpharma.DrugRecallActive, InitiatedAt: now.Add(-24 * time.Hour), Tasks: []domainpharma.DrugRecallTask{{ID: "task-1", Status: domainpharma.DrugRecallTaskPending}, {ID: "task-2", Status: domainpharma.DrugRecallTaskCompleted}}}, {ID: "recall-done", Status: domainpharma.DrugRecallCompleted}}},
		complianceColdChainFixture{items: []*domainpharma.ColdChainAnomaly{{ID: "cold-1", Status: domainpharma.ColdChainAnomalyActive, Risk: domainpharma.ColdChainRiskMedium, ProductID: "product-1", BatchID: "batch-2", BatchNo: "B-002", WarehouseID: "WH-1", AreaID: "A-1", LocationID: "L-1", TemperatureCelsius: 9, MinCelsius: 2, MaxCelsius: 8, HumidityPercent: 55, MinHumidityPercent: 30, MaxHumidityPercent: 70, Reasons: []string{"temperature_above_max"}, TargetPath: "/skoll/pharma-oa/cold-chain?anomalyId=cold-1", LastSeenAt: now.Add(-12 * time.Hour)}, {ID: "cold-done", Status: domainpharma.ColdChainAnomalyResolved}}},
		audit,
	)
	service.(*complianceDashboardService).nowFn = func() time.Time { return now }

	snapshot, err := service.Get(context.Background(), ComplianceDashboardInput{ActorID: "auditor-1"})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Summary.Total != 5 || snapshot.Summary.High != 3 || snapshot.Summary.Medium != 2 || snapshot.Summary.Qualifications != 2 || snapshot.Summary.Complaints != 1 || snapshot.Summary.Recalls != 1 || snapshot.Summary.ColdChain != 1 {
		t.Fatalf("unexpected compliance summary: %+v", snapshot.Summary)
	}
	if len(snapshot.Items) != 5 || snapshot.Items[0].Risk != ComplianceRiskHigh || snapshot.Items[0].Source != ComplianceSourceQualification || snapshot.Items[0].SourceID != "qualification-expired" {
		t.Fatalf("unexpected compliance ordering: %+v", snapshot.Items)
	}
	filtered, err := service.Get(context.Background(), ComplianceDashboardInput{ActorID: "auditor-1", Risk: ComplianceRiskMedium, Source: ComplianceSourceColdChain, Keyword: "B-002", Limit: 1})
	if err != nil || filtered.MatchedCount != 1 || len(filtered.Items) != 1 || filtered.Items[0].SourceID != "cold-1" {
		t.Fatalf("unexpected filtered compliance snapshot: snapshot=%+v err=%v", filtered, err)
	}
	csvBytes, err := service.Export(context.Background(), ComplianceDashboardInput{ActorID: "auditor-1", Source: ComplianceSourceQualification})
	if err != nil {
		t.Fatal(err)
	}
	csvText := string(csvBytes)
	if !strings.Contains(csvText, "Risk,Source,Reference") || complianceCSVCell("=FORMULA") != "'=FORMULA" {
		t.Fatalf("unexpected or unsafe compliance export: %q", csvText)
	}
	audits, err := audit.ListByActor(context.Background(), "auditor-1", 10)
	viewCount, exportCount := 0, 0
	defaultLimitAudits := 0
	for _, record := range audits {
		if record.Action == "pharma_oa.compliance_dashboard.view" {
			viewCount++
		}
		if record.Action == "pharma_oa.compliance_dashboard.export" {
			exportCount++
		}
		if record.Detail["limit"] == 200 {
			defaultLimitAudits++
		}
	}
	if err != nil || len(audits) != 3 || viewCount != 2 || exportCount != 1 || defaultLimitAudits != 2 {
		t.Fatalf("unexpected compliance audits: audits=%+v err=%v", audits, err)
	}
}

func TestComplianceDashboardValidatesFiltersAndDependencies(t *testing.T) {
	service := NewComplianceDashboardService(nil, nil, nil, nil, nil)
	if _, err := service.Get(context.Background(), ComplianceDashboardInput{ActorID: "auditor"}); err == nil {
		t.Fatal("missing compliance dependencies were accepted")
	}
	audit := auditsvc.NewService(clickhouse.NewAuditStore())
	service = NewComplianceDashboardService(complianceQualificationFixture{}, complianceComplaintFixture{}, complianceRecallFixture{}, complianceColdChainFixture{}, audit)
	for _, input := range []ComplianceDashboardInput{{ActorID: ""}, {ActorID: "auditor", Risk: "critical"}, {ActorID: "auditor", Source: "contract"}, {ActorID: "auditor", Limit: 501}} {
		if _, err := service.Get(context.Background(), input); err == nil {
			t.Fatalf("invalid compliance input was accepted: %+v", input)
		}
	}
}
