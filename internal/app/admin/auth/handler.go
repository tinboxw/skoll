package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/user"
)

// APIRegistry keeps route registration and permission discovery consistent.
type APIRegistry interface {
	RegisterMany(entries []string)
}

type UserService interface {
	Create(name, email string) user.User
	Get(id int64) (user.User, error)
	List() []user.User
	RotatePassword(userID int64, minInterval time.Duration, now time.Time) (user.SecurityState, error)
	RegisterLoginFailure(userID int64, lockThreshold int, lockDuration time.Duration, now time.Time) (user.SecurityState, error)
	ResetUserLock(userID int64, now time.Time) (user.SecurityState, error)
	SetMFA(userID int64, enabled bool, provider string, now time.Time) (user.SecurityState, error)
	RevokeSession(sessionID, reason string, now time.Time) user.SessionStatus
	SessionStatus(sessionID string) user.SessionStatus
	ReportSessionAnomaly(sessionID, category, detail string, now time.Time) user.SessionAnomaly
	HeartbeatSessionConsistency(sessionID, instanceID string, version int64, now time.Time) user.SessionConsistency
	SessionConsistencyStatus(sessionID string) user.SessionConsistency
	CreateAuthSession(userID, roleID int64, claimsVersion string, now time.Time) (user.AuthTokenPair, error)
	RefreshAuthSession(refreshToken string, now time.Time) (user.AuthTokenPair, error)
	RevokeAuthSession(sessionID, reason string, now time.Time) (user.AuthSession, error)
	GetAuthSession(sessionID string) (user.AuthSession, error)
}

type AuditService interface {
	Append(actor, action, target string) audit.Record
}

type Handler struct {
	users UserService
	audit AuditService
}

func NewHandler(users UserService, auditSvc AuditService) *Handler {
	return &Handler{users: users, audit: auditSvc}
}

func (h *Handler) Register(mux *http.ServeMux, wrapper func(http.Handler) http.Handler, apis APIRegistry) {
	if mux == nil || h == nil || h.users == nil {
		return
	}

	handle := func(pattern string, next http.HandlerFunc) {
		if apis != nil {
			apis.RegisterMany([]string{pattern})
		}
		hd := http.Handler(next)
		if wrapper != nil {
			hd = wrapper(hd)
		}
		mux.Handle(pattern, hd)
	}

	handlePublic := func(pattern string, next http.HandlerFunc) {
		if apis != nil {
			apis.RegisterMany([]string{pattern})
		}
		mux.Handle(pattern, next)
	}

	handlePublic("POST /admin/v1/auth/login", h.loginAdminAuth)
	handlePublic("POST /admin/v1/auth/refresh", h.refreshAdminAuth)
	handlePublic("POST /admin/v1/auth/logout", h.logoutAdminAuth)
	handlePublic("GET /admin/v1/auth/sessions/{session_id}", h.getAdminAuthSession)

	handle("POST /admin/v1/users", h.createUser)
	handle("POST /admin/v1/users/bulk", h.createUsersBulk)
	handle("GET /admin/v1/users", h.listUsers)
	handle("GET /admin/v1/users/{id}", h.getUser)
	handle("POST /admin/v1/users/{id}/password/rotate", h.rotateUserPassword)
	handle("POST /admin/v1/users/{id}/login-failures", h.registerUserLoginFailure)
	handle("POST /admin/v1/users/{id}/lock/reset", h.resetUserLock)
	handle("POST /admin/v1/users/{id}/mfa", h.setUserMFA)
	handle("POST /admin/v1/sessions/revoke", h.revokeSession)
	handle("GET /admin/v1/sessions/{session_id}/status", h.getSessionStatus)
	handle("POST /admin/v1/sessions/anomalies", h.reportSessionAnomaly)
	handle("POST /admin/v1/sessions/consistency/heartbeat", h.heartbeatSessionConsistency)
	handle("GET /admin/v1/sessions/{session_id}/consistency", h.getSessionConsistency)
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	if req.Name == "" || req.Email == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "name and email are required"})
		return
	}

	respondJSON(w, http.StatusCreated, h.users.Create(req.Name, req.Email))
}

func (h *Handler) createUsersBulk(w http.ResponseWriter, r *http.Request) {
	var req createUsersBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if len(req.Items) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "items are required"})
		return
	}
	for i := range req.Items {
		req.Items[i].Name = strings.TrimSpace(req.Items[i].Name)
		req.Items[i].Email = strings.TrimSpace(req.Items[i].Email)
		if req.Items[i].Name == "" || req.Items[i].Email == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid item at index %d", i)})
			return
		}
	}

	out := make([]user.User, 0, len(req.Items))
	for _, item := range req.Items {
		out = append(out, h.users.Create(item.Name, item.Email))
	}
	h.appendAudit("admin", "users_bulk_create", fmt.Sprintf("count:%d", len(out)))
	respondJSON(w, http.StatusCreated, createUsersBulkResponse{Atomic: true, Count: len(out), Items: out})
}

func (h *Handler) listUsers(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, h.users.List())
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt64(r, "id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	out, err := h.users.Get(id)
	if err != nil {
		if err == user.ErrUserNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	respondJSON(w, http.StatusOK, out)
}

func (h *Handler) rotateUserPassword(w http.ResponseWriter, r *http.Request) {
	userID, err := parsePathInt64(r, "id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var req rotateUserPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if req.MinIntervalMinutes < 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "min_interval_minutes must be >= 0"})
		return
	}
	state, err := h.users.RotatePassword(userID, time.Duration(req.MinIntervalMinutes)*time.Minute, time.Now().UTC())
	if err != nil {
		if err == user.ErrUserNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.appendAudit("security", "password_rotate", fmt.Sprintf("user:%d", userID))
	respondJSON(w, http.StatusOK, state)
}

func (h *Handler) registerUserLoginFailure(w http.ResponseWriter, r *http.Request) {
	userID, err := parsePathInt64(r, "id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var req registerUserLoginFailureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if req.LockThreshold <= 0 {
		req.LockThreshold = 5
	}
	if req.LockDurationMinute <= 0 {
		req.LockDurationMinute = 30
	}
	state, err := h.users.RegisterLoginFailure(userID, req.LockThreshold, time.Duration(req.LockDurationMinute)*time.Minute, time.Now().UTC())
	if err != nil {
		if err == user.ErrUserNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.appendAudit("security", "login_failure", fmt.Sprintf("user:%d", userID))
	respondJSON(w, http.StatusOK, state)
}

func (h *Handler) resetUserLock(w http.ResponseWriter, r *http.Request) {
	userID, err := parsePathInt64(r, "id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	state, err := h.users.ResetUserLock(userID, time.Now().UTC())
	if err != nil {
		if err == user.ErrUserNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.appendAudit("security", "lock_reset", fmt.Sprintf("user:%d", userID))
	respondJSON(w, http.StatusOK, state)
}

func (h *Handler) setUserMFA(w http.ResponseWriter, r *http.Request) {
	userID, err := parsePathInt64(r, "id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var req setUserMFARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	provider := strings.TrimSpace(req.Provider)
	if req.Enabled && provider == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "provider is required when MFA is enabled"})
		return
	}
	state, err := h.users.SetMFA(userID, req.Enabled, provider, time.Now().UTC())
	if err != nil {
		if err == user.ErrUserNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.appendAudit("security", "mfa_update", fmt.Sprintf("user:%d", userID))
	respondJSON(w, http.StatusOK, state)
}

func (h *Handler) loginAdminAuth(w http.ResponseWriter, r *http.Request) {
	var req adminAuthLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if req.UserID <= 0 || req.RoleID <= 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id and role_id must be > 0"})
		return
	}

	pair, err := h.users.CreateAuthSession(req.UserID, req.RoleID, req.ClaimsVersion, time.Now().UTC())
	if err != nil {
		if err == user.ErrUserNotFound {
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	h.appendAudit("auth", "login", fmt.Sprintf("user:%d session:%s", req.UserID, pair.SessionID))
	respondJSON(w, http.StatusOK, pair)
}

func (h *Handler) refreshAdminAuth(w http.ResponseWriter, r *http.Request) {
	var req adminAuthRefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	pair, err := h.users.RefreshAuthSession(strings.TrimSpace(req.RefreshToken), time.Now().UTC())
	if err != nil {
		switch err {
		case user.ErrAuthInvalidRefreshToken, user.ErrAuthRefreshTokenExpired, user.ErrAuthSessionRevoked:
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_refresh_token"})
			return
		case user.ErrAuthSessionNotFound:
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		default:
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
	}

	h.appendAudit("auth", "refresh", pair.SessionID)
	respondJSON(w, http.StatusOK, pair)
}

func (h *Handler) logoutAdminAuth(w http.ResponseWriter, r *http.Request) {
	var req adminAuthLogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.Reason = strings.TrimSpace(req.Reason)
	if req.SessionID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "session_id is required"})
		return
	}
	if req.Reason == "" {
		req.Reason = "logout"
	}

	session, err := h.users.RevokeAuthSession(req.SessionID, req.Reason, time.Now().UTC())
	if err != nil {
		if err == user.ErrAuthSessionNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	_ = h.users.RevokeSession(req.SessionID, req.Reason, time.Now().UTC())
	h.appendAudit("auth", "logout", req.SessionID)
	respondJSON(w, http.StatusOK, session)
}

func (h *Handler) getAdminAuthSession(w http.ResponseWriter, r *http.Request) {
	sessionID, err := parsePathString(r, "session_id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	session, err := h.users.GetAuthSession(sessionID)
	if err != nil {
		if err == user.ErrAuthSessionNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	respondJSON(w, http.StatusOK, session)
}

func (h *Handler) revokeSession(w http.ResponseWriter, r *http.Request) {
	var req revokeSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.Reason = strings.TrimSpace(req.Reason)
	if req.SessionID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "session_id is required"})
		return
	}
	if req.Reason == "" {
		req.Reason = "manual revoke"
	}
	status := h.users.RevokeSession(req.SessionID, req.Reason, time.Now().UTC())
	h.appendAudit("security", "session_revoke", req.SessionID)
	respondJSON(w, http.StatusOK, status)
}

func (h *Handler) getSessionStatus(w http.ResponseWriter, r *http.Request) {
	sessionID, err := parsePathString(r, "session_id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, h.users.SessionStatus(sessionID))
}

func (h *Handler) reportSessionAnomaly(w http.ResponseWriter, r *http.Request) {
	var req reportSessionAnomalyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.Category = strings.TrimSpace(req.Category)
	req.Detail = strings.TrimSpace(req.Detail)
	if req.SessionID == "" || req.Category == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "session_id and category are required"})
		return
	}
	out := h.users.ReportSessionAnomaly(req.SessionID, req.Category, req.Detail, time.Now().UTC())
	h.appendAudit("security", "session_anomaly", req.SessionID+":"+req.Category)
	respondJSON(w, http.StatusCreated, out)
}

func (h *Handler) heartbeatSessionConsistency(w http.ResponseWriter, r *http.Request) {
	var req heartbeatSessionConsistencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.InstanceID = strings.TrimSpace(req.InstanceID)
	if req.SessionID == "" || req.InstanceID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "session_id and instance_id are required"})
		return
	}
	if req.Version <= 0 {
		req.Version = 1
	}

	out := h.users.HeartbeatSessionConsistency(req.SessionID, req.InstanceID, req.Version, time.Now().UTC())
	action := "session_consistency_heartbeat"
	if !out.Consistent {
		action = "session_consistency_conflict"
	}
	h.appendAudit("consistency", action, req.SessionID)
	respondJSON(w, http.StatusOK, out)
}

func (h *Handler) getSessionConsistency(w http.ResponseWriter, r *http.Request) {
	sessionID, err := parsePathString(r, "session_id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, h.users.SessionConsistencyStatus(sessionID))
}

func (h *Handler) appendAudit(actor, action, target string) {
	if h.audit == nil {
		return
	}
	h.audit.Append(actor, action, target)
}

func parsePathInt64(r *http.Request, key string) (int64, error) {
	raw := strings.TrimSpace(r.PathValue(key))
	if raw == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return id, nil
}

func parsePathString(r *http.Request, key string) (string, error) {
	raw := strings.TrimSpace(r.PathValue(key))
	if raw == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return raw, nil
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type createUsersBulkRequest struct {
	Items []createUserRequest `json:"items"`
}

type createUsersBulkResponse struct {
	Atomic bool        `json:"atomic"`
	Count  int         `json:"count"`
	Items  []user.User `json:"items"`
}

type rotateUserPasswordRequest struct {
	MinIntervalMinutes int `json:"min_interval_minutes"`
}

type registerUserLoginFailureRequest struct {
	LockThreshold      int `json:"lock_threshold"`
	LockDurationMinute int `json:"lock_duration_minutes"`
}

type setUserMFARequest struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"`
}

type revokeSessionRequest struct {
	SessionID string `json:"session_id"`
	Reason    string `json:"reason"`
}

type reportSessionAnomalyRequest struct {
	SessionID string `json:"session_id"`
	Category  string `json:"category"`
	Detail    string `json:"detail"`
}

type heartbeatSessionConsistencyRequest struct {
	SessionID  string `json:"session_id"`
	InstanceID string `json:"instance_id"`
	Version    int64  `json:"version"`
}

type adminAuthLoginRequest struct {
	UserID        int64  `json:"user_id"`
	RoleID        int64  `json:"role_id"`
	ClaimsVersion string `json:"claims_version"`
}

type adminAuthRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type adminAuthLogoutRequest struct {
	SessionID string `json:"session_id"`
	Reason    string `json:"reason"`
}
