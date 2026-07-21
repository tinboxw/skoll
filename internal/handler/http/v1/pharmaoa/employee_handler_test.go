package pharmaoa

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	"github.com/tinboxw/skoll/pkg/security"
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
	var listPayload struct {
		Data struct {
			Items      []map[string]any `json:"items"`
			Total      int64            `json:"total"`
			Limit      int              `json:"limit"`
			HasMore    bool             `json:"hasMore"`
			NextCursor string           `json:"nextCursor"`
			Sort       string           `json:"sort"`
		} `json:"data"`
	}
	pageResp := performRequest(mux, http.MethodGet, "/v1/pharma-oa/employees?limit=1", nil)
	if err := json.Unmarshal(pageResp.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("decode page response: %v", err)
	}
	if listPayload.Data.Total != 1 || listPayload.Data.Limit != 1 || listPayload.Data.HasMore || listPayload.Data.NextCursor != "" || listPayload.Data.Sort != pharmaListSort {
		t.Fatalf("unexpected page metadata: %+v", listPayload.Data)
	}
	invalidCursor := performRequest(mux, http.MethodGet, "/v1/pharma-oa/employees?cursor=not-valid-*", nil)
	if invalidCursor.Code != http.StatusBadRequest {
		t.Fatalf("invalid cursor status=%d body=%s", invalidCursor.Code, invalidCursor.Body.String())
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

func TestActorIDFromRequestPrefersJWTSubject(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/pharma-oa/employees", nil)
	req = req.WithContext(security.WithJWTClaimsContext(req.Context(), &security.JWTClaims{Subject: "jwt-user", Role: "employee"}))
	if actorID := actorIDFromRequest(req, "spoofed-user"); actorID != "jwt-user" {
		t.Fatalf("actor id=%s, want jwt-user", actorID)
	}
}

func TestNormalizePagination(t *testing.T) {
	tests := []struct {
		offset     int
		limit      int
		wantOffset int
		wantLimit  int
	}{
		{-1, 0, 0, 50},
		{20, 100, 20, 100},
		{30, 500, 30, 200},
	}
	for _, test := range tests {
		offset, limit := normalizePagination(test.offset, test.limit)
		if offset != test.wantOffset || limit != test.wantLimit {
			t.Errorf("normalizePagination(%d, %d)=(%d, %d), want (%d, %d)", test.offset, test.limit, offset, limit, test.wantOffset, test.wantLimit)
		}
	}
}

func TestPharmaListCursorRoundTrip(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/pharma-oa/employees?cursor=NTA&limit=25", nil)
	pagination, err := parsePharmaListPagination(req)
	if err != nil || pagination.Offset != 50 || pagination.Limit != 25 {
		t.Fatalf("pagination=%+v err=%v", pagination, err)
	}
}

func TestEmployeeListSerializesEmptyCertificatesAsArray(t *testing.T) {
	service := pharmaoasvc.NewEmployeeService(nil)
	mux := http.NewServeMux()
	RegisterEmployeeRoutes(mux, service)
	created := performRequest(mux, http.MethodPost, "/v1/pharma-oa/employees", []byte(`{"code":"EMP-EMPTY","name":"Empty certificates","departmentId":"quality","positionId":"qa","certificates":[]}`))
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	listed := performRequest(mux, http.MethodGet, "/v1/pharma-oa/employees?keyword=EMP-EMPTY", nil)
	if listed.Code != http.StatusOK || !bytes.Contains(listed.Body.Bytes(), []byte(`"certificates":[]`)) {
		t.Fatalf("empty certificates must serialize as []: status=%d body=%s", listed.Code, listed.Body.String())
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
