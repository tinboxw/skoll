package pharmaoa

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
)

func TestEmployeeHandlerCreateListLeaveAndReminders(t *testing.T) {
	service := pharmaoasvc.NewEmployeeService(nil)
	mux := http.NewServeMux()
	RegisterEmployeeRoutes(mux, service)

	createBody := []byte(`{
		"code":"EMP001",
		"name":"Alice",
		"departmentId":"quality",
		"positionId":"qa",
		"certificates":[{"id":"cert-gsp","name":"GSP","number":"GSP-001","expiresAt":"2026-07-20"}],
		"actorId":"admin"
	}`)
	createResp := performRequest(mux, http.MethodPost, "/v1/pharma-oa/employees", createBody)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", createResp.Code, createResp.Body.String())
	}
	var createPayload struct {
		Data struct {
			Item struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"item"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createPayload.Data.Item.ID == "" || createPayload.Data.Item.Status != "active" {
		t.Fatalf("unexpected create response: %+v", createPayload.Data.Item)
	}

	listResp := performRequest(mux, http.MethodGet, "/v1/pharma-oa/employees?keyword=alice", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listResp.Code, listResp.Body.String())
	}
	if !bytes.Contains(listResp.Body.Bytes(), []byte("EMP001")) {
		t.Fatalf("list response missing employee: %s", listResp.Body.String())
	}

	reminderResp := performRequest(mux, http.MethodGet, "/v1/pharma-oa/employees/qualification-reminders?days=60", nil)
	if reminderResp.Code != http.StatusOK || !bytes.Contains(reminderResp.Body.Bytes(), []byte("GSP")) {
		t.Fatalf("reminder response invalid: status=%d body=%s", reminderResp.Code, reminderResp.Body.String())
	}

	leaveResp := performRequest(mux, http.MethodPost, "/v1/pharma-oa/employees/"+createPayload.Data.Item.ID+"/leave", []byte(`{"reason":"resigned","actorId":"admin"}`))
	if leaveResp.Code != http.StatusOK || !bytes.Contains(leaveResp.Body.Bytes(), []byte("left")) {
		t.Fatalf("leave response invalid: status=%d body=%s", leaveResp.Code, leaveResp.Body.String())
	}
}

func performRequest(handler http.Handler, method string, path string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}
