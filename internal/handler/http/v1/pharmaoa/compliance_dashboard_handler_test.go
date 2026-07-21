package pharmaoa

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	"github.com/tinboxw/skoll/pkg/security"
)

type complianceDashboardHTTPFixture struct {
	inputs  []pharmaoasvc.ComplianceDashboardInput
	exports int
}

func (f *complianceDashboardHTTPFixture) Get(_ context.Context, in pharmaoasvc.ComplianceDashboardInput) (*pharmaoasvc.ComplianceDashboardSnapshot, error) {
	f.inputs = append(f.inputs, in)
	return &pharmaoasvc.ComplianceDashboardSnapshot{
		Summary: pharmaoasvc.ComplianceDashboardSummary{Total: 1, High: 1, Complaints: 1}, MatchedCount: 1, GeneratedAt: time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC),
		Items: []pharmaoasvc.ComplianceRiskItem{{ID: "quality_complaint:complaint-1", Source: pharmaoasvc.ComplianceSourceQualityComplaint, SourceID: "complaint-1", Risk: pharmaoasvc.ComplianceRiskHigh, Reference: "QC-1", Title: "Quality issue", TargetPath: "/skoll/pharma-oa/quality-complaints?complaintId=complaint-1"}},
	}, nil
}

func (f *complianceDashboardHTTPFixture) Export(_ context.Context, in pharmaoasvc.ComplianceDashboardInput) ([]byte, error) {
	f.inputs = append(f.inputs, in)
	f.exports++
	return []byte("Risk,Source\nhigh,quality_complaint\n"), nil
}

func TestComplianceDashboardHTTPFiltersJWTActorAndExport(t *testing.T) {
	service := &complianceDashboardHTTPFixture{}
	mux := http.NewServeMux()
	RegisterComplianceDashboardRoutes(mux, service)

	list := contractHTTPRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/compliance-dashboard?risk=high&source=quality_complaint&keyword=QC-1&limit=25&actorId=spoofed", "", security.JWTClaims{Subject: "compliance-auditor", Role: "quality"})
	if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte(`"matchedCount":1`)) || !bytes.Contains(list.Body.Bytes(), []byte(`"sourceId":"complaint-1"`)) {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}
	if len(service.inputs) != 1 || service.inputs[0].ActorID != "compliance-auditor" || service.inputs[0].Risk != pharmaoasvc.ComplianceRiskHigh || service.inputs[0].Source != pharmaoasvc.ComplianceSourceQualityComplaint || service.inputs[0].Keyword != "QC-1" || service.inputs[0].Limit != 25 {
		t.Fatalf("unexpected list input: %+v", service.inputs)
	}

	export := contractHTTPRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/compliance-dashboard/export?source=quality_complaint", "", security.JWTClaims{Subject: "compliance-auditor", Role: "quality"})
	if export.Code != http.StatusOK || export.Header().Get("Content-Type") != "text/csv; charset=utf-8" || !bytes.Contains(export.Body.Bytes(), []byte("high,quality_complaint")) || service.exports != 1 || service.inputs[1].ActorID != "compliance-auditor" {
		t.Fatalf("export status=%d headers=%v body=%s inputs=%+v", export.Code, export.Header(), export.Body.String(), service.inputs)
	}

	invalid := contractHTTPRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/compliance-dashboard?limit=501", "", security.JWTClaims{Subject: "compliance-auditor", Role: "quality"})
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid status=%d body=%s", invalid.Code, invalid.Body.String())
	}
}
