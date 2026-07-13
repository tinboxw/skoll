package pharmaoa

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	"github.com/tinboxw/skoll/pkg/security"
)

func TestAnnouncementHTTPPublishAudienceAndReadConfirmation(t *testing.T) {
	service := pharmaoasvc.NewAnnouncementService(nil)
	mux := http.NewServeMux()
	RegisterAnnouncementRoutes(mux, service)

	create := announcementRequest(t, mux, http.MethodPost, "/v1/pharma-oa/announcements", `{"kind":"policy","title":"Quality policy","content":"Read before shift","audience":{"organizationIds":[],"roleIds":["quality"]},"documents":[{"fileId":"file-1","fileName":"policy.pdf"}],"actorId":"admin"}`, security.JWTClaims{Subject: "admin", Role: "admin"})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	items, _ := service.List(context.Background(), pharmaoasvc.AnnouncementListInput{ActorID: "admin", IncludeDraft: true})
	if len(items) != 1 {
		t.Fatalf("draft not available to creator: %+v", items)
	}
	id := items[0].ID.String()

	publish := announcementRequest(t, mux, http.MethodPost, "/v1/pharma-oa/announcements/"+id+"/publish", `{"actorId":"admin"}`, security.JWTClaims{Subject: "admin", Role: "admin"})
	if publish.Code != http.StatusOK {
		t.Fatalf("publish status=%d body=%s", publish.Code, publish.Body.String())
	}
	visible := announcementRequest(t, mux, http.MethodGet, "/v1/pharma-oa/announcements", "", security.JWTClaims{Subject: "quality-user", Role: "quality"})
	if visible.Code != http.StatusOK || !bytes.Contains(visible.Body.Bytes(), []byte(id)) {
		t.Fatalf("targeted list status=%d body=%s", visible.Code, visible.Body.String())
	}
	hidden := announcementRequest(t, mux, http.MethodGet, "/v1/pharma-oa/announcements", "", security.JWTClaims{Subject: "sales-user", Role: "sales"})
	if hidden.Code != http.StatusOK || bytes.Contains(hidden.Body.Bytes(), []byte(id)) {
		t.Fatalf("out-of-audience list status=%d body=%s", hidden.Code, hidden.Body.String())
	}
	read := announcementRequest(t, mux, http.MethodPost, "/v1/pharma-oa/announcements/"+id+"/read", "", security.JWTClaims{Subject: "quality-user", Role: "quality"})
	if read.Code != http.StatusOK {
		t.Fatalf("confirm read status=%d body=%s", read.Code, read.Body.String())
	}
	receipts := announcementRequest(t, mux, http.MethodGet, "/v1/pharma-oa/announcements/"+id+"/read-confirmations", "", security.JWTClaims{Subject: "admin", Role: "admin"})
	if receipts.Code != http.StatusOK || !bytes.Contains(receipts.Body.Bytes(), []byte("quality-user")) {
		t.Fatalf("receipt list status=%d body=%s", receipts.Code, receipts.Body.String())
	}
}

func announcementRequest(t *testing.T, handler http.Handler, method, target, body string, claims security.JWTClaims) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(security.WithJWTClaimsContext(req.Context(), &claims))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}
