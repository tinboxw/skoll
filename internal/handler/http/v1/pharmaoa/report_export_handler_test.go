package pharmaoa

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	"github.com/tinboxw/skoll/pkg/security"
)

type reportExportHandlerFixture struct {
	input    pharmaoasvc.ReportExportCreateInput
	actorID  string
	download *pharmaoasvc.ReportExportFile
	err      error
}

func (f *reportExportHandlerFixture) Queue(_ context.Context, in pharmaoasvc.ReportExportCreateInput) (*pharmaoasvc.ReportExportJob, error) {
	f.input = in
	return &pharmaoasvc.ReportExportJob{ID: "job-1", Status: pharmaoasvc.ReportExportPending, OwnerID: in.ActorID}, f.err
}
func (f *reportExportHandlerFixture) List(_ context.Context, actorID string) ([]*pharmaoasvc.ReportExportJob, error) {
	f.actorID = actorID
	return []*pharmaoasvc.ReportExportJob{{ID: "job-1"}}, f.err
}
func (f *reportExportHandlerFixture) Get(_ context.Context, _, actorID string) (*pharmaoasvc.ReportExportJob, error) {
	f.actorID = actorID
	return &pharmaoasvc.ReportExportJob{ID: "job-1"}, f.err
}
func (f *reportExportHandlerFixture) Retry(_ context.Context, _, actorID string) (*pharmaoasvc.ReportExportJob, error) {
	f.actorID = actorID
	return &pharmaoasvc.ReportExportJob{ID: "job-1", Status: pharmaoasvc.ReportExportPending}, f.err
}
func (f *reportExportHandlerFixture) Download(_ context.Context, _, actorID string) (*pharmaoasvc.ReportExportFile, error) {
	f.actorID = actorID
	return f.download, f.err
}

func withReportClaims(req *http.Request) *http.Request {
	return req.WithContext(security.WithJWTClaimsContext(req.Context(), &security.JWTClaims{Subject: "manager-1", Role: "manager"}))
}

func TestReportExportHTTPQueuesWithJWTActorAndBoundedQuery(t *testing.T) {
	fixture := &reportExportHandlerFixture{}
	mux := http.NewServeMux()
	RegisterReportExportRoutes(mux, fixture)
	body := []byte(`{"reportType":"sales_trend","from":"2026-07-01","to":"2026-07-07","bucket":"week","qualificationDays":45,"actorId":"spoofed"}`)
	req := withReportClaims(httptest.NewRequest(http.MethodPost, "/v1/plugins/pharma_oa/api/report-export-jobs", bytes.NewReader(body)))
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if fixture.input.ActorID != "manager-1" || fixture.input.ReportType != pharmaoasvc.ReportExportSalesTrend || fixture.input.Bucket != pharmaoasvc.BusinessMetricsBucketWeek || fixture.input.QualificationDays != 45 {
		t.Fatalf("unexpected queue input: %+v", fixture.input)
	}
	if got := fixture.input.To.Format("2006-01-02T15:04:05.999999999Z07:00"); got != "2026-07-07T23:59:59.999999999Z" {
		t.Fatalf("unexpected inclusive to: %s", got)
	}
}

func TestReportExportHTTPDownloadsCSVAsAttachment(t *testing.T) {
	fixture := &reportExportHandlerFixture{download: &pharmaoasvc.ReportExportFile{Filename: "sales_trend.csv", ContentType: "text/csv", Body: []byte("amount_cents\n3234\n")}}
	mux := http.NewServeMux()
	RegisterReportExportRoutes(mux, fixture)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, withReportClaims(httptest.NewRequest(http.MethodGet, "/v1/plugins/pharma_oa/api/report-export-jobs/job-1/download", nil)))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "amount_cents\n3234\n" {
		t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Content-Type") != "text/csv" || recorder.Header().Get("X-Content-Type-Options") != "nosniff" || recorder.Header().Get("Content-Disposition") != `attachment; filename=sales_trend.csv` {
		t.Fatalf("unsafe download headers: %+v", recorder.Header())
	}
	if fixture.actorID != "manager-1" {
		t.Fatalf("unexpected download actor: %s", fixture.actorID)
	}
}

func TestReportExportHTTPMapsOwnershipAndReadinessErrors(t *testing.T) {
	for _, item := range []struct {
		err  error
		want int
	}{{pharmaoasvc.ErrReportExportNotFound, http.StatusNotFound}, {pharmaoasvc.ErrReportExportAccessDenied, http.StatusForbidden}, {pharmaoasvc.ErrReportExportNotReady, http.StatusConflict}, {errors.New("invalid report"), http.StatusBadRequest}} {
		fixture := &reportExportHandlerFixture{err: item.err}
		mux := http.NewServeMux()
		RegisterReportExportRoutes(mux, fixture)
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, withReportClaims(httptest.NewRequest(http.MethodGet, "/v1/plugins/pharma_oa/api/report-export-jobs/job-1/download", nil)))
		if recorder.Code != item.want {
			t.Fatalf("err=%v status=%d body=%s", item.err, recorder.Code, recorder.Body.String())
		}
	}
}
