package pharmaoa

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
)

func TestSupplierHandlerCreateReminderAndPurchaseBlock(t *testing.T) {
	service := pharmaoasvc.NewSupplierService(nil)
	mux := http.NewServeMux()
	RegisterSupplierRoutes(mux, service)

	createResp := performRequest(mux, http.MethodPost, "/v1/pharma-oa/suppliers", []byte(`{
		"code":"SUP001",
		"name":"Skoll Medical Supply",
		"rating":5,
		"contacts":[{"id":"primary","name":"Jane","phone":"13800000000","email":"jane@skoll.local","primary":true}],
		"qualifications":[{"id":"gsp","name":"GSP License","number":"GSP-SUP-001","expiresAt":"2026-07-20","attachments":[{"fileId":"file-001","fileName":"gsp-license.pdf","mimeType":"application/pdf","size":1024}]}],
		"actorId":"admin"
	}`))
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", createResp.Code, createResp.Body.String())
	}
	var createPayload struct {
		Data struct {
			Item struct {
				ID string `json:"id"`
			} `json:"item"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createPayload.Data.Item.ID == "" {
		t.Fatalf("expected supplier id: %+v", createPayload)
	}

	reminderResp := performRequest(mux, http.MethodGet, "/v1/pharma-oa/suppliers/qualification-reminders?days=60", nil)
	if reminderResp.Code != http.StatusOK || !bytes.Contains(reminderResp.Body.Bytes(), []byte("GSP License")) {
		t.Fatalf("reminder response invalid: status=%d body=%s", reminderResp.Code, reminderResp.Body.String())
	}

	eligibleResp := performRequest(mux, http.MethodGet, "/v1/pharma-oa/suppliers/"+createPayload.Data.Item.ID+"/purchase-eligibility", nil)
	if eligibleResp.Code != http.StatusOK || !bytes.Contains(eligibleResp.Body.Bytes(), []byte(`"allowed":true`)) {
		t.Fatalf("eligible response invalid: status=%d body=%s", eligibleResp.Code, eligibleResp.Body.String())
	}

	disableResp := performRequest(mux, http.MethodPost, "/v1/pharma-oa/suppliers/"+createPayload.Data.Item.ID+"/disable", []byte(`{"reason":"blacklisted","actorId":"admin"}`))
	if disableResp.Code != http.StatusOK || !bytes.Contains(disableResp.Body.Bytes(), []byte("disabled")) {
		t.Fatalf("disable response invalid: status=%d body=%s", disableResp.Code, disableResp.Body.String())
	}

	blockResp := performRequest(mux, http.MethodGet, "/v1/pharma-oa/suppliers/"+createPayload.Data.Item.ID+"/purchase-eligibility", nil)
	if blockResp.Code != http.StatusOK || !bytes.Contains(blockResp.Body.Bytes(), []byte(`"allowed":false`)) {
		t.Fatalf("blocked response invalid: status=%d body=%s", blockResp.Code, blockResp.Body.String())
	}
}

func TestSupplierHandlerRejectsUnsafeAttachment(t *testing.T) {
	service := pharmaoasvc.NewSupplierService(nil)
	mux := http.NewServeMux()
	RegisterSupplierRoutes(mux, service)

	resp := performRequest(mux, http.MethodPost, "/v1/pharma-oa/suppliers", []byte(`{
		"code":"SUP001",
		"name":"Skoll Medical Supply",
		"qualifications":[{"name":"GSP License","attachments":[{"fileId":"file-001","fileName":"../run.exe","size":1}]}]
	}`))
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected unsafe attachment rejected, status=%d body=%s", resp.Code, resp.Body.String())
	}
}
