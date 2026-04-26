package adminauth

import (
	"fmt"
	"strings"
	"sync/atomic"
)

const (
	reasonOK               = "ok"
	reasonMissingToken     = "missing_token"
	reasonInvalidToken     = "invalid_token"
	reasonMissingHeaders   = "missing_headers"
	reasonInvalidTimestamp = "invalid_timestamp"
	reasonTimestampSkew    = "timestamp_skew"
	reasonReplayNonce      = "replay_nonce"
	reasonInvalidBodyHash  = "invalid_body_hash"
	reasonInvalidSignature = "invalid_signature"
)

type modeMetrics struct {
	success atomic.Uint64
	failure atomic.Uint64

	missingToken     atomic.Uint64
	invalidToken     atomic.Uint64
	missingHeaders   atomic.Uint64
	invalidTimestamp atomic.Uint64
	timestampSkew    atomic.Uint64
	replayNonce      atomic.Uint64
	invalidBodyHash  atomic.Uint64
	invalidSignature atomic.Uint64
}

type MetricsModeSnapshot struct {
	Success uint64            `json:"success"`
	Failure uint64            `json:"failure"`
	Reasons map[string]uint64 `json:"reasons"`
}

type MetricsSnapshot struct {
	StaticToken MetricsModeSnapshot `json:"static_token"`
	HMACSHA256  MetricsModeSnapshot `json:"hmac_sha256"`
}

var adminAuthMetrics = struct {
	static modeMetrics
	hmac   modeMetrics
}{}

func observeVerification(mode string, success bool, reason string) {
	m := modeCounter(mode)
	if m == nil {
		return
	}

	if success {
		m.success.Add(1)
		return
	}

	m.failure.Add(1)
	switch reason {
	case reasonMissingToken:
		m.missingToken.Add(1)
	case reasonInvalidToken:
		m.invalidToken.Add(1)
	case reasonMissingHeaders:
		m.missingHeaders.Add(1)
	case reasonInvalidTimestamp:
		m.invalidTimestamp.Add(1)
	case reasonTimestampSkew:
		m.timestampSkew.Add(1)
	case reasonReplayNonce:
		m.replayNonce.Add(1)
	case reasonInvalidBodyHash:
		m.invalidBodyHash.Add(1)
	case reasonInvalidSignature:
		m.invalidSignature.Add(1)
	}
}

func modeCounter(mode string) *modeMetrics {
	switch mode {
	case "static-token":
		return &adminAuthMetrics.static
	case "hmac-sha256":
		return &adminAuthMetrics.hmac
	default:
		return nil
	}
}

func MetricsPrometheus() string {
	var b strings.Builder
	b.WriteString("# HELP skoll_admin_auth_verifications_total Total number of admin auth verifications by mode and result.\n")
	b.WriteString("# TYPE skoll_admin_auth_verifications_total counter\n")
	writeVerificationResult(&b, "static-token", "success", adminAuthMetrics.static.success.Load())
	writeVerificationResult(&b, "static-token", "failure", adminAuthMetrics.static.failure.Load())
	writeVerificationResult(&b, "hmac-sha256", "success", adminAuthMetrics.hmac.success.Load())
	writeVerificationResult(&b, "hmac-sha256", "failure", adminAuthMetrics.hmac.failure.Load())

	b.WriteString("# HELP skoll_admin_auth_failures_total Total number of failed admin auth verifications by mode and reason.\n")
	b.WriteString("# TYPE skoll_admin_auth_failures_total counter\n")
	writeFailureReasons(&b, "static-token", &adminAuthMetrics.static)
	writeFailureReasons(&b, "hmac-sha256", &adminAuthMetrics.hmac)
	return b.String()
}

func Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		StaticToken: modeSnapshot(&adminAuthMetrics.static),
		HMACSHA256:  modeSnapshot(&adminAuthMetrics.hmac),
	}
}

func modeSnapshot(m *modeMetrics) MetricsModeSnapshot {
	return MetricsModeSnapshot{
		Success: m.success.Load(),
		Failure: m.failure.Load(),
		Reasons: map[string]uint64{
			reasonMissingToken:     m.missingToken.Load(),
			reasonInvalidToken:     m.invalidToken.Load(),
			reasonMissingHeaders:   m.missingHeaders.Load(),
			reasonInvalidTimestamp: m.invalidTimestamp.Load(),
			reasonTimestampSkew:    m.timestampSkew.Load(),
			reasonReplayNonce:      m.replayNonce.Load(),
			reasonInvalidBodyHash:  m.invalidBodyHash.Load(),
			reasonInvalidSignature: m.invalidSignature.Load(),
		},
	}
}

func writeVerificationResult(b *strings.Builder, mode, result string, total uint64) {
	if total == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("skoll_admin_auth_verifications_total{mode=%q,result=%q} %d\n", mode, result, total))
}

func writeFailureReasons(b *strings.Builder, mode string, m *modeMetrics) {
	writeFailureReason(b, mode, reasonMissingToken, m.missingToken.Load())
	writeFailureReason(b, mode, reasonInvalidToken, m.invalidToken.Load())
	writeFailureReason(b, mode, reasonMissingHeaders, m.missingHeaders.Load())
	writeFailureReason(b, mode, reasonInvalidTimestamp, m.invalidTimestamp.Load())
	writeFailureReason(b, mode, reasonTimestampSkew, m.timestampSkew.Load())
	writeFailureReason(b, mode, reasonReplayNonce, m.replayNonce.Load())
	writeFailureReason(b, mode, reasonInvalidBodyHash, m.invalidBodyHash.Load())
	writeFailureReason(b, mode, reasonInvalidSignature, m.invalidSignature.Load())
}

func writeFailureReason(b *strings.Builder, mode, reason string, total uint64) {
	if total == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("skoll_admin_auth_failures_total{mode=%q,reason=%q} %d\n", mode, reason, total))
}

func resetMetricsForTest() {
	adminAuthMetrics = struct {
		static modeMetrics
		hmac   modeMetrics
	}{}
}
