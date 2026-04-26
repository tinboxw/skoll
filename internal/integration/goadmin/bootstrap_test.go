package goadmin

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	adminsdk "github.com/go-admin-team/go-admin-core/sdk/pkg"
	"github.com/tinboxw/skoll/internal/integration/adminauth"
)

func TestNewBootstrapValidMode(t *testing.T) {
	b, err := New(true, "dev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !b.Enabled() {
		t.Fatalf("expected enabled bootstrap")
	}
	if b.Mode() != "dev" {
		t.Fatalf("expected mode dev, got %q", b.Mode())
	}
}

func TestNewBootstrapInvalidMode(t *testing.T) {
	_, err := New(true, "staging")
	if err == nil {
		t.Fatalf("expected error for invalid mode")
	}
}

func TestInitDisabledNoError(t *testing.T) {
	b, err := New(false, "prod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := b.Init(context.Background()); err != nil {
		t.Fatalf("expected no error for disabled bootstrap, got %v", err)
	}
}

func TestAdminPingHandler(t *testing.T) {
	b, err := New(true, "dev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, AdminPingPath, nil)
	b.AdminPingHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	requestID := rr.Header().Get(adminsdk.TrafficKey)
	if len(requestID) != 20 {
		t.Fatalf("expected request id length 20, got %d (%q)", len(requestID), requestID)
	}

	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}

	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", body["status"])
	}
	if body["integration"] != "go-admin" {
		t.Fatalf("expected integration go-admin, got %q", body["integration"])
	}
	if body["mode"] != "dev" {
		t.Fatalf("expected mode dev, got %q", body["mode"])
	}
	if body["request_id"] != requestID {
		t.Fatalf("expected body request id to match header, got %q vs %q", body["request_id"], requestID)
	}
}

func TestAdminPingHandlerWithHMACVerifier(t *testing.T) {
	b, err := New(true, "dev")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	verifier, enabled, err := adminauth.ResolveVerifier("hmac-sha256", "", "hmac-secret")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !enabled {
		t.Fatalf("expected verifier enabled")
	}

	handler := adminauth.WithVerifier(http.HandlerFunc(b.AdminPingHandler()), verifier)

	buildSignature := func(method, path, timestamp, nonce string) string {
		mac := hmac.New(sha256.New, []byte("hmac-secret"))
		_, _ = mac.Write([]byte(method + "\n" + path + "\n" + timestamp + "\n" + nonce + "\n"))
		return hex.EncodeToString(mac.Sum(nil))
	}

	t.Run("allow signed request", func(t *testing.T) {
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		nonce := "nonce-it-1"
		req := httptest.NewRequest(http.MethodGet, AdminPingPath, nil)
		req.Header.Set(adminauth.HeaderTimestamp, ts)
		req.Header.Set(adminauth.HeaderNonce, nonce)
		req.Header.Set(adminauth.HeaderSignature, buildSignature(http.MethodGet, AdminPingPath, ts, nonce))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		var body map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if body["integration"] != "go-admin" {
			t.Fatalf("expected integration go-admin, got %q", body["integration"])
		}
	})

	t.Run("reject unsigned request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, AdminPingPath, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rr.Code)
		}
	})
}
