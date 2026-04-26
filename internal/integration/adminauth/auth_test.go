package adminauth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

type stubNonceStore struct {
	mu   sync.Mutex
	seen map[string]struct{}
	hits int
}

func (s *stubNonceStore) UseOnce(nonce string, _ time.Time, _ time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hits++
	if nonce == "" {
		return false
	}
	if _, ok := s.seen[nonce]; ok {
		return false
	}
	s.seen[nonce] = struct{}{}
	return true
}

func (s *stubNonceStore) Hits() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits
}

func TestResolveVerifier(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		token       string
		hmacSecret  string
		wantEnabled bool
		wantErr     bool
	}{
		{name: "auto disabled without credentials", mode: "auto", token: "", hmacSecret: "", wantEnabled: false},
		{name: "auto enabled with token", mode: "auto", token: "secret", wantEnabled: true},
		{name: "auto enabled with hmac", mode: "auto", token: "", hmacSecret: "hmac-secret", wantEnabled: true},
		{name: "none disabled", mode: "none", token: "secret", wantEnabled: false},
		{name: "static token enabled", mode: "static-token", token: "secret", wantEnabled: true},
		{name: "static token missing token", mode: "static-token", token: "", wantErr: true},
		{name: "hmac enabled", mode: "hmac-sha256", hmacSecret: "hmac-secret", wantEnabled: true},
		{name: "hmac missing secret", mode: "hmac-sha256", wantErr: true},
		{name: "invalid mode", mode: "jwt", token: "secret", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			verifier, enabled, err := ResolveVerifier(tc.mode, tc.token, tc.hmacSecret)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if enabled != tc.wantEnabled {
				t.Fatalf("expected enabled=%t, got %t", tc.wantEnabled, enabled)
			}
			if enabled && verifier == nil {
				t.Fatalf("expected verifier when enabled")
			}
		})
	}
}

func TestWithVerifier(t *testing.T) {
	verifier, enabled, err := ResolveVerifier("static-token", "secret", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !enabled {
		t.Fatalf("expected enabled verifier")
	}

	protected := WithVerifier(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), verifier)

	t.Run("reject missing token", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
		protected.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("allow valid token", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
		req.Header.Set(HeaderToken, "secret")
		protected.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})
}

func TestHMACSHA256Verifier(t *testing.T) {
	fixedNow := time.Unix(1710000000, 0)
	verifier := &hmacSHA256Verifier{
		secret: []byte("hmac-secret"),
		now: func() time.Time {
			return fixedNow
		},
		nonceStore: newMemoryNonceStore(),
	}

	buildSignature := func(method, path, timestamp, nonce, bodyHash string) string {
		mac := hmac.New(sha256.New, []byte("hmac-secret"))
		_, _ = mac.Write([]byte(signaturePayload(method, path, timestamp, nonce, bodyHash)))
		return hex.EncodeToString(mac.Sum(nil))
	}

	t.Run("allow valid signature", func(t *testing.T) {
		ts := strconv.FormatInt(fixedNow.Unix(), 10)
		nonce := "nonce-1"
		req := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
		req.Header.Set(HeaderTimestamp, ts)
		req.Header.Set(HeaderNonce, nonce)
		req.Header.Set(HeaderSignature, buildSignature(http.MethodGet, "/admin/ping", ts, nonce, ""))
		if !verifier.Verify(req) {
			t.Fatalf("expected verification success")
		}
	})

	t.Run("reject replay nonce", func(t *testing.T) {
		ts := strconv.FormatInt(fixedNow.Unix(), 10)
		nonce := "nonce-replay"
		sig := buildSignature(http.MethodGet, "/admin/ping", ts, nonce, "")

		req1 := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
		req1.Header.Set(HeaderTimestamp, ts)
		req1.Header.Set(HeaderNonce, nonce)
		req1.Header.Set(HeaderSignature, sig)
		if !verifier.Verify(req1) {
			t.Fatalf("expected first verification success")
		}

		req2 := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
		req2.Header.Set(HeaderTimestamp, ts)
		req2.Header.Set(HeaderNonce, nonce)
		req2.Header.Set(HeaderSignature, sig)
		if verifier.Verify(req2) {
			t.Fatalf("expected replay verification failure")
		}
	})

	t.Run("reject stale timestamp", func(t *testing.T) {
		stale := strconv.FormatInt(fixedNow.Add(-6*time.Minute).Unix(), 10)
		nonce := "nonce-2"
		req := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
		req.Header.Set(HeaderTimestamp, stale)
		req.Header.Set(HeaderNonce, nonce)
		req.Header.Set(HeaderSignature, buildSignature(http.MethodGet, "/admin/ping", stale, nonce, ""))
		if verifier.Verify(req) {
			t.Fatalf("expected verification failure for stale timestamp")
		}
	})

	t.Run("reject invalid signature", func(t *testing.T) {
		ts := strconv.FormatInt(fixedNow.Unix(), 10)
		nonce := "nonce-3"
		req := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
		req.Header.Set(HeaderTimestamp, ts)
		req.Header.Set(HeaderNonce, nonce)
		req.Header.Set(HeaderSignature, "deadbeef")
		if verifier.Verify(req) {
			t.Fatalf("expected verification failure for invalid signature")
		}
	})

	t.Run("allow valid body hash", func(t *testing.T) {
		ts := strconv.FormatInt(fixedNow.Unix(), 10)
		nonce := "nonce-body-1"
		body := []byte(`{"k":"v"}`)
		sum := sha256.Sum256(body)
		bodyHash := hex.EncodeToString(sum[:])

		req := httptest.NewRequest(http.MethodPost, "/admin/ping", bytes.NewReader(body))
		req.Header.Set(HeaderTimestamp, ts)
		req.Header.Set(HeaderNonce, nonce)
		req.Header.Set(HeaderBodySHA, bodyHash)
		req.Header.Set(HeaderSignature, buildSignature(http.MethodPost, "/admin/ping", ts, nonce, bodyHash))

		if !verifier.Verify(req) {
			t.Fatalf("expected verification success with body hash")
		}
	})

	t.Run("reject invalid body hash", func(t *testing.T) {
		ts := strconv.FormatInt(fixedNow.Unix(), 10)
		nonce := "nonce-body-2"
		body := []byte(`{"k":"v"}`)
		wrong := "0000000000000000000000000000000000000000000000000000000000000000"

		req := httptest.NewRequest(http.MethodPost, "/admin/ping", bytes.NewReader(body))
		req.Header.Set(HeaderTimestamp, ts)
		req.Header.Set(HeaderNonce, nonce)
		req.Header.Set(HeaderBodySHA, wrong)
		req.Header.Set(HeaderSignature, buildSignature(http.MethodPost, "/admin/ping", ts, nonce, wrong))

		if verifier.Verify(req) {
			t.Fatalf("expected verification failure for invalid body hash")
		}
	})
}

func TestResolveVerifierWithNonceStore(t *testing.T) {
	store := &stubNonceStore{seen: make(map[string]struct{})}
	verifier, enabled, err := ResolveVerifierWithNonceStore("hmac-sha256", "", "hmac-secret", store)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !enabled {
		t.Fatalf("expected enabled verifier")
	}

	hv, ok := verifier.(*hmacSHA256Verifier)
	if !ok {
		t.Fatalf("expected hmacSHA256Verifier type")
	}
	fixedNow := time.Unix(1710000000, 0)
	hv.now = func() time.Time { return fixedNow }

	ts := strconv.FormatInt(fixedNow.Unix(), 10)
	nonce := "pluggable-nonce"
	mac := hmac.New(sha256.New, []byte("hmac-secret"))
	_, _ = mac.Write([]byte(signaturePayload(http.MethodGet, "/admin/ping", ts, nonce, "")))
	sig := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
	req.Header.Set(HeaderTimestamp, ts)
	req.Header.Set(HeaderNonce, nonce)
	req.Header.Set(HeaderSignature, sig)

	if !hv.Verify(req) {
		t.Fatalf("expected verification success")
	}
	if store.Hits() != 1 {
		t.Fatalf("expected nonce store hits=1, got %d", store.Hits())
	}
}
