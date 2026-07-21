package pharmaoa

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
	servicepharma "github.com/tinboxw/skoll/internal/service/pharmaoa"
	"github.com/tinboxw/skoll/pkg/security"
)

func TestQualificationHTTPListAndExpiryScan(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	employees := servicepharma.NewEmployeeService(nil)
	_, err := employees.Create(ctx, servicepharma.EmployeeWriteInput{Code: "EMP-HTTP-Q", Name: "Alice", DepartmentID: "dept-1", PositionID: "position-1", Certificates: []domainpharma.EmployeeCertificate{{ID: "health", Name: "Health certificate", Number: "HC-1", ExpiresAt: now.AddDate(0, 0, 10)}}})
	if err != nil {
		t.Fatal(err)
	}
	suppliers := servicepharma.NewSupplierService(nil)
	customers := servicepharma.NewCustomerService(nil)
	service := servicepharma.NewQualificationService(employees, suppliers, customers, notificationsvc.NewService(nil, nil), nil)
	mux := http.NewServeMux()
	RegisterQualificationRoutes(mux, service)

	list := contractHTTPRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/qualifications?subjectType=employee&status=expiring&days=30", "", security.JWTClaims{Subject: "compliance-1", Role: "compliance"})
	if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte("Health certificate")) {
		t.Fatalf("qualification list status=%d body=%s", list.Code, list.Body.String())
	}
	scan := contractHTTPRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/qualifications/expiry-scan", `{"days":30,"actorId":"spoofed"}`, security.JWTClaims{Subject: "compliance-1", Role: "compliance"})
	if scan.Code != http.StatusOK || !bytes.Contains(scan.Body.Bytes(), []byte(`"createdCount":1`)) {
		t.Fatalf("qualification scan status=%d body=%s", scan.Code, scan.Body.String())
	}
	invalid := contractHTTPRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/qualifications?days=999", "", security.JWTClaims{Subject: "compliance-1", Role: "compliance"})
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid qualification days status=%d body=%s", invalid.Code, invalid.Body.String())
	}
	invalidScan := contractHTTPRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/qualifications/expiry-scan", `{"days":999}`, security.JWTClaims{Subject: "compliance-1", Role: "compliance"})
	if invalidScan.Code != http.StatusBadRequest {
		t.Fatalf("invalid qualification scan days status=%d body=%s", invalidScan.Code, invalidScan.Body.String())
	}
}
