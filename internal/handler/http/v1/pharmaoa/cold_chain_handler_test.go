package pharmaoa

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	"github.com/tinboxw/skoll/pkg/security"
)

func TestColdChainHTTPRecordsJobsAndAnomalies(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 13, 11, 0, 0, 0, time.UTC)
	warehouses := pharmaoasvc.NewWarehouseService(nil)
	warehouse, err := warehouses.Create(ctx, pharmaoasvc.WarehouseWriteInput{Code: "HTTP-COLD", Name: "HTTP Cold", Region: "East", Temperature: domainpharma.WarehouseTemperature{Controlled: true, MinCelsius: 2, MaxCelsius: 8}, Areas: []domainpharma.WarehouseArea{{ID: "area-http", Code: "AH", Name: "Area", Locations: []domainpharma.WarehouseLocation{{ID: "location-http", Code: "LH", Name: "Location"}}}}})
	if err != nil {
		t.Fatalf("create warehouse: %v", err)
	}
	inventory := pharmaoasvc.NewInventoryService(nil)
	stock, err := inventory.Inbound(ctx, pharmaoasvc.StockMovementInput{ReferenceID: "HTTP-COLD-IN", ProductID: "product-http", WarehouseID: warehouse.ID.String(), AreaID: "area-http", LocationID: "location-http", BatchNo: "HTTP-B-001", ProductionDate: now.AddDate(-1, 0, 0), ExpiresAt: now.AddDate(1, 0, 0), Quantity: 10})
	if err != nil {
		t.Fatalf("seed inventory: %v", err)
	}
	service := pharmaoasvc.NewColdChainService(inventory, warehouses, notificationsvc.NewService(nil, nil), nil)
	mux := http.NewServeMux()
	RegisterColdChainRoutes(mux, service)

	contexts := contractHTTPRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/cold-chain-contexts", "", security.JWTClaims{Subject: "quality-user", Role: "quality"})
	if contexts.Code != http.StatusOK || !bytes.Contains(contexts.Body.Bytes(), []byte(`"batchNo":"HTTP-B-001"`)) || !bytes.Contains(contexts.Body.Bytes(), []byte(`"minCelsius":2`)) {
		t.Fatalf("contexts status=%d body=%s", contexts.Code, contexts.Body.String())
	}
	createBody := `{"balanceId":"` + stock.Balance.ID.String() + `","temperatureCelsius":10,"humidityPercent":80,"source":"sensor-http","recordedAt":"2026-07-13T11:00:00Z","actorId":"spoofed"}`
	created := contractHTTPRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/cold-chain-records", createBody, security.JWTClaims{Subject: "quality-user", Role: "quality"})
	if created.Code != http.StatusCreated || !bytes.Contains(created.Body.Bytes(), []byte(`"recordedBy":"quality-user"`)) || !bytes.Contains(created.Body.Bytes(), []byte(`"batchId":"`+stock.Balance.BatchID+`"`)) {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	run := contractHTTPRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/cold-chain-jobs", `{"minHumidityPercent":30,"maxHumidityPercent":70,"recipientId":"quality-manager","actorId":"spoofed"}`, security.JWTClaims{Subject: "quality-user", Role: "quality"})
	if run.Code != http.StatusCreated || !bytes.Contains(run.Body.Bytes(), []byte(`"initiatedBy":"quality-user"`)) || !bytes.Contains(run.Body.Bytes(), []byte(`"createdCount":1`)) {
		t.Fatalf("run status=%d body=%s", run.Code, run.Body.String())
	}
	anomalies := contractHTTPRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/cold-chain-anomalies?activeOnly=true", "", security.JWTClaims{Subject: "quality-user", Role: "quality"})
	if anomalies.Code != http.StatusOK || !bytes.Contains(anomalies.Body.Bytes(), []byte(`"risk":"high"`)) || !bytes.Contains(anomalies.Body.Bytes(), []byte(`"batchId":"`+stock.Balance.BatchID+`"`)) {
		t.Fatalf("anomalies status=%d body=%s", anomalies.Code, anomalies.Body.String())
	}
	invalid := contractHTTPRequest(mux, http.MethodGet, "/v1/plugins/pharma_oa/api/cold-chain-records?limit=501", "", security.JWTClaims{Subject: "quality-user", Role: "quality"})
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid limit status=%d body=%s", invalid.Code, invalid.Body.String())
	}
	retry := contractHTTPRequest(mux, http.MethodPost, "/v1/plugins/pharma_oa/api/cold-chain-jobs/cold-chain-job-1/retry", `{"actorId":"spoofed"}`, security.JWTClaims{Subject: "quality-user", Role: "quality"})
	if retry.Code != http.StatusBadRequest {
		t.Fatalf("succeeded retry status=%d body=%s", retry.Code, retry.Body.String())
	}
}
