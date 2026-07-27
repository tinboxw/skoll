package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	ReverificationPurposeWorkflowSignature = "workflow-signature"
	maxReverificationTTL                   = 5 * time.Minute
)

type ReverificationClaims struct {
	ID         string    `json:"id"`
	Subject    string    `json:"subject"`
	Audience   string    `json:"audience"`
	Purpose    string    `json:"purpose"`
	Method     string    `json:"method"`
	VerifiedAt time.Time `json:"verifiedAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

type ReverificationProofVerifier interface {
	VerifyReverificationProof(token string, now time.Time) (ReverificationClaims, error)
}

type ReverificationProofService struct {
	key []byte
}

func NewReverificationProofService(secret string) (*ReverificationProofService, error) {
	secret = strings.TrimSpace(secret)
	if len(secret) < 16 {
		return nil, errors.New("reverification secret must contain at least 16 characters")
	}
	return &ReverificationProofService{key: []byte(secret)}, nil
}

func (s *ReverificationProofService) IssueReverificationProof(subject, audience, purpose, method string, now time.Time, ttl time.Duration) (string, ReverificationClaims, error) {
	if s == nil || len(s.key) == 0 {
		return "", ReverificationClaims{}, errors.New("reverification proof service is not configured")
	}
	subject = strings.TrimSpace(subject)
	audience = strings.TrimSpace(audience)
	purpose = strings.TrimSpace(purpose)
	method = strings.TrimSpace(method)
	if subject == "" || len(subject) > 128 {
		return "", ReverificationClaims{}, errors.New("reverification subject is invalid")
	}
	if audience == "" || len(audience) > 128 {
		return "", ReverificationClaims{}, errors.New("reverification audience is invalid")
	}
	if purpose != ReverificationPurposeWorkflowSignature {
		return "", ReverificationClaims{}, errors.New("reverification purpose is invalid")
	}
	if method != "password" {
		return "", ReverificationClaims{}, errors.New("reverification method is invalid")
	}
	if ttl <= 0 || ttl > maxReverificationTTL {
		return "", ReverificationClaims{}, errors.New("reverification lifetime is invalid")
	}
	if now.IsZero() {
		return "", ReverificationClaims{}, errors.New("reverification time is required")
	}
	verificationID, err := randomVerificationID()
	if err != nil {
		return "", ReverificationClaims{}, err
	}
	claims := ReverificationClaims{
		ID: verificationID, Subject: subject, Audience: audience, Purpose: purpose, Method: method,
		VerifiedAt: now.UTC(), ExpiresAt: now.UTC().Add(ttl),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", ReverificationClaims{}, fmt.Errorf("encode reverification claims: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	signature := s.sign(encoded)
	return "v1." + encoded + "." + signature, claims, nil
}

func (s *ReverificationProofService) VerifyReverificationProof(token string, now time.Time) (ReverificationClaims, error) {
	if s == nil || len(s.key) == 0 {
		return ReverificationClaims{}, errors.New("reverification proof service is not configured")
	}
	token = strings.TrimSpace(token)
	if token == "" || len(token) > 4096 {
		return ReverificationClaims{}, errors.New("reverification proof is invalid")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != "v1" {
		return ReverificationClaims{}, errors.New("reverification proof is invalid")
	}
	expected, err := hex.DecodeString(s.sign(parts[1]))
	if err != nil {
		return ReverificationClaims{}, errors.New("reverification proof is invalid")
	}
	actual, err := hex.DecodeString(parts[2])
	if err != nil || !hmac.Equal(actual, expected) {
		return ReverificationClaims{}, errors.New("reverification proof signature is invalid")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ReverificationClaims{}, errors.New("reverification proof payload is invalid")
	}
	var claims ReverificationClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ReverificationClaims{}, errors.New("reverification proof payload is invalid")
	}
	if claims.ID == "" || claims.Subject == "" || claims.Audience == "" ||
		claims.Purpose != ReverificationPurposeWorkflowSignature || claims.Method != "password" ||
		claims.VerifiedAt.IsZero() || claims.ExpiresAt.IsZero() || !claims.ExpiresAt.After(claims.VerifiedAt) ||
		claims.ExpiresAt.Sub(claims.VerifiedAt) > maxReverificationTTL {
		return ReverificationClaims{}, errors.New("reverification proof claims are invalid")
	}
	now = now.UTC()
	if now.Before(claims.VerifiedAt.Add(-30*time.Second)) || !now.Before(claims.ExpiresAt) {
		return ReverificationClaims{}, errors.New("reverification proof is expired")
	}
	return claims, nil
}

func (s *ReverificationProofService) sign(payload string) string {
	mac := hmac.New(sha256.New, s.key)
	_, _ = mac.Write([]byte("skoll-reverification\x00" + payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func randomVerificationID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate reverification identity: %w", err)
	}
	return "verify-" + hex.EncodeToString(value[:]), nil
}
