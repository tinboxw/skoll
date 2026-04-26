package adminauth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const HeaderToken = "X-Admin-Token"

const (
	HeaderSignature = "X-Admin-Signature"
	HeaderTimestamp = "X-Admin-Timestamp"
	HeaderNonce     = "X-Admin-Nonce"
	HeaderBodySHA   = "X-Admin-Body-SHA256"
	hmacTimeSkew    = 5 * time.Minute
)

// Verifier defines a pluggable admin auth verification contract.
type Verifier interface {
	Mode() string
	Verify(*http.Request) bool
}

// ReplayNonceStore defines one-time nonce consumption for replay defense.
type ReplayNonceStore interface {
	UseOnce(nonce string, now time.Time, ttl time.Duration) bool
}

type staticTokenVerifier struct {
	token string
}

type hmacSHA256Verifier struct {
	secret     []byte
	now        func() time.Time
	nonceStore ReplayNonceStore
}

type memoryNonceStore struct {
	mu   sync.Mutex
	seen map[string]int64
}

func (v *staticTokenVerifier) Mode() string {
	return "static-token"
}

func (v *staticTokenVerifier) Verify(r *http.Request) bool {
	result := false
	reason := reasonInvalidToken
	defer func() {
		observeVerification(v.Mode(), result, reason)
	}()

	got := r.Header.Get(HeaderToken)
	if got == "" {
		reason = reasonMissingToken
		return false
	}
	if subtle.ConstantTimeCompare([]byte(got), []byte(v.token)) != 1 {
		reason = reasonInvalidToken
		return false
	}

	result = true
	reason = reasonOK
	return true
}

func (v *hmacSHA256Verifier) Mode() string {
	return "hmac-sha256"
}

func (v *hmacSHA256Verifier) Verify(r *http.Request) bool {
	result := false
	reason := reasonInvalidSignature
	defer func() {
		observeVerification(v.Mode(), result, reason)
	}()

	timestampRaw := r.Header.Get(HeaderTimestamp)
	signatureRaw := r.Header.Get(HeaderSignature)
	nonceRaw := r.Header.Get(HeaderNonce)
	bodyHashRaw := strings.ToLower(r.Header.Get(HeaderBodySHA))
	if timestampRaw == "" || signatureRaw == "" || nonceRaw == "" {
		reason = reasonMissingHeaders
		return false
	}

	timestampUnix, err := strconv.ParseInt(timestampRaw, 10, 64)
	if err != nil {
		reason = reasonInvalidTimestamp
		return false
	}
	timestamp := time.Unix(timestampUnix, 0)
	now := v.now()
	if timestamp.Before(now.Add(-hmacTimeSkew)) || timestamp.After(now.Add(hmacTimeSkew)) {
		reason = reasonTimestampSkew
		return false
	}
	if !v.nonceStore.UseOnce(nonceRaw, now, hmacTimeSkew) {
		reason = reasonReplayNonce
		return false
	}

	if bodyHashRaw != "" {
		if !isBodyHashValid(r, bodyHashRaw) {
			reason = reasonInvalidBodyHash
			return false
		}
	}

	mac := hmac.New(sha256.New, v.secret)
	_, _ = mac.Write([]byte(signaturePayload(r.Method, r.URL.Path, timestampRaw, nonceRaw, bodyHashRaw)))
	want := mac.Sum(nil)

	got, err := hex.DecodeString(signatureRaw)
	if err != nil {
		reason = reasonInvalidSignature
		return false
	}
	if len(got) != len(want) {
		reason = reasonInvalidSignature
		return false
	}
	if subtle.ConstantTimeCompare(got, want) != 1 {
		reason = reasonInvalidSignature
		return false
	}

	result = true
	reason = reasonOK
	return true
}

func ResolveVerifier(mode, token, hmacSecret string) (Verifier, bool, error) {
	return ResolveVerifierWithNonceStore(mode, token, hmacSecret, nil)
}

func ResolveVerifierWithNonceStore(mode, token, hmacSecret string, nonceStore ReplayNonceStore) (Verifier, bool, error) {
	switch mode {
	case "", "auto":
		if token == "" {
			if hmacSecret == "" {
				return nil, false, nil
			}
			return &hmacSHA256Verifier{secret: []byte(hmacSecret), now: time.Now, nonceStore: resolveNonceStore(nonceStore)}, true, nil
		}
		return &staticTokenVerifier{token: token}, true, nil
	case "none":
		return nil, false, nil
	case "static-token":
		if token == "" {
			return nil, false, fmt.Errorf("admin auth mode static-token requires non-empty token")
		}
		return &staticTokenVerifier{token: token}, true, nil
	case "hmac-sha256":
		if hmacSecret == "" {
			return nil, false, fmt.Errorf("admin auth mode hmac-sha256 requires non-empty hmac secret")
		}
		return &hmacSHA256Verifier{secret: []byte(hmacSecret), now: time.Now, nonceStore: resolveNonceStore(nonceStore)}, true, nil
	default:
		return nil, false, fmt.Errorf("unsupported admin auth mode %q, valid: auto|none|static-token|hmac-sha256", mode)
	}
}

func resolveNonceStore(store ReplayNonceStore) ReplayNonceStore {
	if store != nil {
		return store
	}
	return newMemoryNonceStore()
}

func signaturePayload(method, path, timestamp, nonce, bodyHash string) string {
	return method + "\n" + path + "\n" + timestamp + "\n" + nonce + "\n" + bodyHash
}

func isBodyHashValid(r *http.Request, expectedHex string) bool {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	sum := sha256.Sum256(body)
	gotHex := hex.EncodeToString(sum[:])
	if len(gotHex) != len(expectedHex) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(gotHex), []byte(expectedHex)) == 1
}

func newMemoryNonceStore() *memoryNonceStore {
	return &memoryNonceStore{seen: make(map[string]int64)}
}

func (s *memoryNonceStore) UseOnce(nonce string, now time.Time, ttl time.Duration) bool {
	if nonce == "" {
		return false
	}
	nowUnix := now.Unix()
	expiresAt := now.Add(ttl).Unix()

	s.mu.Lock()
	defer s.mu.Unlock()

	for key, exp := range s.seen {
		if exp <= nowUnix {
			delete(s.seen, key)
		}
	}
	if exp, ok := s.seen[nonce]; ok && exp > nowUnix {
		return false
	}
	s.seen[nonce] = expiresAt
	return true
}

func WithVerifier(next http.Handler, verifier Verifier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !verifier.Verify(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "admin authentication required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
