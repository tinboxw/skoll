package middleware

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/pkg/security"
)

type AuditEventSink interface {
	AppendEvent(ctx context.Context, event *domainaudit.Event) error
}

type AuditIDFunc func() shared.ID

type AuditClock func() time.Time

type RequestAuditOption func(*requestAuditOptions)

type requestAuditOptions struct {
	nowFn     AuditClock
	idFn      AuditIDFunc
	skipPaths []string
}

func RequestAudit(sink AuditEventSink, opts ...RequestAuditOption) func(http.Handler) http.Handler {
	cfg := requestAuditOptions{
		nowFn: func() time.Time { return time.Now().UTC() },
		idFn: func() shared.ID {
			return shared.ID("audit-event-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10))
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder := &auditResponseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(recorder, r)
			if sink == nil || shouldSkipAuditPath(r.URL.Path, cfg.skipPaths) {
				return
			}
			event, err := newRequestAuditEvent(r, recorder.statusCode, recorder.bytesWritten, cfg)
			if err != nil {
				return
			}
			_ = sink.AppendEvent(r.Context(), event)
		})
	}
}

func WithRequestAuditClock(nowFn AuditClock) RequestAuditOption {
	return func(cfg *requestAuditOptions) {
		if nowFn != nil {
			cfg.nowFn = nowFn
		}
	}
}

func WithRequestAuditID(idFn AuditIDFunc) RequestAuditOption {
	return func(cfg *requestAuditOptions) {
		if idFn != nil {
			cfg.idFn = idFn
		}
	}
}

func WithRequestAuditSkipPaths(paths ...string) RequestAuditOption {
	return func(cfg *requestAuditOptions) {
		cfg.skipPaths = append(cfg.skipPaths, paths...)
	}
}

type auditResponseRecorder struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	wroteHeader  bool
}

func (w *auditResponseRecorder) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.statusCode = statusCode
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *auditResponseRecorder) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(data)
	w.bytesWritten += n
	return n, err
}

func newRequestAuditEvent(r *http.Request, statusCode, bytesWritten int, cfg requestAuditOptions) (*domainaudit.Event, error) {
	if r == nil {
		return nil, nil
	}
	now := cfg.nowFn()
	action, err := domainaudit.ParseAuditAction("http.request." + strings.ToLower(strings.TrimSpace(r.Method)))
	if err != nil {
		return nil, err
	}
	actor := requestActor(r.Context())
	result, risk := requestResultAndRisk(statusCode)
	return domainaudit.NewEvent(domainaudit.EventInput{
		ID:     cfg.idFn(),
		Type:   domainaudit.EventTypeOperation,
		Action: action,
		Actor:  actor,
		Resource: domainaudit.ResourceRef{
			Type: "http_request",
			ID:   strings.TrimSpace(r.Method) + " " + strings.TrimSpace(r.URL.Path),
		},
		Result: result,
		Trace: domainaudit.TraceContext{
			TraceID:   strings.TrimSpace(r.Header.Get("X-Trace-Id")),
			RequestID: strings.TrimSpace(r.Header.Get("X-Request-Id")),
			Method:    r.Method,
			Path:      r.URL.Path,
			IP:        requestIP(r),
			UserAgent: r.UserAgent(),
		},
		Risk: risk,
		Metadata: map[string]any{
			"status":       statusCode,
			"bytesWritten": bytesWritten,
		},
		SourceData: map[string]any{
			"method":      r.Method,
			"path":        r.URL.Path,
			"query":       r.URL.RawQuery,
			"status":      statusCode,
			"remoteAddr":  r.RemoteAddr,
			"userAgent":   r.UserAgent(),
			"requestId":   strings.TrimSpace(r.Header.Get("X-Request-Id")),
			"contentType": strings.TrimSpace(r.Header.Get("Content-Type")),
		},
		OccurredAt: now,
	})
}

func requestActor(ctx context.Context) domainaudit.ActorRef {
	if claims, ok := security.JWTClaimsFromContext(ctx); ok && strings.TrimSpace(claims.Subject) != "" {
		return domainaudit.ActorRef{Type: "user", ID: shared.ID(strings.TrimSpace(claims.Subject)), Name: strings.TrimSpace(claims.Role)}
	}
	return domainaudit.ActorRef{Type: "anonymous", ID: shared.ID("anonymous")}
}

func requestResultAndRisk(statusCode int) (domainaudit.EventResult, domainaudit.EventRisk) {
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return domainaudit.EventResultDenied, domainaudit.EventRiskMedium
	}
	if statusCode >= http.StatusBadRequest {
		return domainaudit.EventResultFailure, domainaudit.EventRiskMedium
	}
	return domainaudit.EventResultSuccess, domainaudit.EventRiskLow
}

func requestIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func shouldSkipAuditPath(path string, skipPaths []string) bool {
	cleanPath := strings.TrimSpace(path)
	for _, raw := range skipPaths {
		skip := strings.TrimSpace(raw)
		if skip == "" {
			continue
		}
		if cleanPath == skip || strings.HasPrefix(cleanPath, skip+"/") {
			return true
		}
	}
	return false
}
