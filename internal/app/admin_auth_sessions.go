package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/module/user"
)

func createUserHandler(svc UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		respondJSON(w, http.StatusCreated, svc.Create(req.Name, req.Email))
	}
}

func createUsersBulkHandler(svc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
			out = append(out, svc.Create(item.Name, item.Email))
		}
		auditSvc.Append("admin", "users_bulk_create", fmt.Sprintf("count:%d", len(out)))
		respondJSON(w, http.StatusCreated, createUsersBulkResponse{Atomic: true, Count: len(out), Items: out})
	}
}

func listUsersHandler(svc UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, svc.List())
	}
}

func getUserHandler(svc UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		out, err := svc.Get(id)
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
}

func rotateUserPasswordHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		state, err := userSvc.RotatePassword(userID, time.Duration(req.MinIntervalMinutes)*time.Minute, time.Now().UTC())
		if err != nil {
			if err == user.ErrUserNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		auditSvc.Append("security", "password_rotate", fmt.Sprintf("user:%d", userID))
		respondJSON(w, http.StatusOK, state)
	}
}

func registerUserLoginFailureHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		state, err := userSvc.RegisterLoginFailure(userID, req.LockThreshold, time.Duration(req.LockDurationMinute)*time.Minute, time.Now().UTC())
		if err != nil {
			if err == user.ErrUserNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		auditSvc.Append("security", "login_failure", fmt.Sprintf("user:%d", userID))
		respondJSON(w, http.StatusOK, state)
	}
}

func resetUserLockHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		state, err := userSvc.ResetUserLock(userID, time.Now().UTC())
		if err != nil {
			if err == user.ErrUserNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		auditSvc.Append("security", "lock_reset", fmt.Sprintf("user:%d", userID))
		respondJSON(w, http.StatusOK, state)
	}
}

func setUserMFAHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		state, err := userSvc.SetMFA(userID, req.Enabled, provider, time.Now().UTC())
		if err != nil {
			if err == user.ErrUserNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		auditSvc.Append("security", "mfa_update", fmt.Sprintf("user:%d", userID))
		respondJSON(w, http.StatusOK, state)
	}
}

func loginAdminAuthHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req adminAuthLoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		if req.UserID <= 0 || req.RoleID <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id and role_id must be > 0"})
			return
		}

		pair, err := userSvc.CreateAuthSession(req.UserID, req.RoleID, req.ClaimsVersion, time.Now().UTC())
		if err != nil {
			if err == user.ErrUserNotFound {
				respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		auditSvc.Append("auth", "login", fmt.Sprintf("user:%d session:%s", req.UserID, pair.SessionID))
		respondJSON(w, http.StatusOK, pair)
	}
}

func refreshAdminAuthHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req adminAuthRefreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		pair, err := userSvc.RefreshAuthSession(strings.TrimSpace(req.RefreshToken), time.Now().UTC())
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

		auditSvc.Append("auth", "refresh", pair.SessionID)
		respondJSON(w, http.StatusOK, pair)
	}
}

func logoutAdminAuthHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		session, err := userSvc.RevokeAuthSession(req.SessionID, req.Reason, time.Now().UTC())
		if err != nil {
			if err == user.ErrAuthSessionNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		_ = userSvc.RevokeSession(req.SessionID, req.Reason, time.Now().UTC())
		auditSvc.Append("auth", "logout", req.SessionID)
		respondJSON(w, http.StatusOK, session)
	}
}

func getAdminAuthSessionHandler(userSvc UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, err := parsePathString(r, "session_id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		session, err := userSvc.GetAuthSession(sessionID)
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
}

func revokeSessionHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		status := userSvc.RevokeSession(req.SessionID, req.Reason, time.Now().UTC())
		auditSvc.Append("security", "session_revoke", req.SessionID)
		respondJSON(w, http.StatusOK, status)
	}
}

func getSessionStatusHandler(userSvc UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, err := parsePathString(r, "session_id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusOK, userSvc.SessionStatus(sessionID))
	}
}

func reportSessionAnomalyHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		out := userSvc.ReportSessionAnomaly(req.SessionID, req.Category, req.Detail, time.Now().UTC())
		auditSvc.Append("security", "session_anomaly", req.SessionID+":"+req.Category)
		respondJSON(w, http.StatusCreated, out)
	}
}

func heartbeatSessionConsistencyHandler(userSvc UserService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		out := userSvc.HeartbeatSessionConsistency(req.SessionID, req.InstanceID, req.Version, time.Now().UTC())
		action := "session_consistency_heartbeat"
		if !out.Consistent {
			action = "session_consistency_conflict"
		}
		auditSvc.Append("consistency", action, req.SessionID)
		respondJSON(w, http.StatusOK, out)
	}
}

func getSessionConsistencyHandler(userSvc UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, err := parsePathString(r, "session_id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusOK, userSvc.SessionConsistencyStatus(sessionID))
	}
}
