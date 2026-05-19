package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainuser "github.com/tinboxw/skoll/internal/domain/user"
	httpHandler "github.com/tinboxw/skoll/internal/handler/http"
	rbacrepo "github.com/tinboxw/skoll/internal/repository/rbac"
	rolerepo "github.com/tinboxw/skoll/internal/repository/role"
	userrepo "github.com/tinboxw/skoll/internal/repository/user"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	"github.com/tinboxw/skoll/pkg/logging"
	"github.com/tinboxw/skoll/pkg/security"
)

type builtinAuthHandler struct {
	jwtSecret string
	usersRepo userrepo.UserRepository
	rolesRepo rolerepo.RoleRepository
	rbacRepo  rbacrepo.RBACRepository
	auditSvc  auditsvc.Service
	logger    logging.Logger
}

func newBuiltinAuthHandler(jwtSecret string, usersRepo userrepo.UserRepository, rolesRepo rolerepo.RoleRepository, rbacRepo rbacrepo.RBACRepository, auditSvc auditsvc.Service, logger logging.Logger) *builtinAuthHandler {
	if usersRepo == nil || rolesRepo == nil || rbacRepo == nil {
		return nil
	}
	return &builtinAuthHandler{
		jwtSecret: jwtSecret,
		usersRepo: usersRepo,
		rolesRepo: rolesRepo,
		rbacRepo:  rbacRepo,
		auditSvc:  auditSvc,
		logger:    logger,
	}
}

func (h *builtinAuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.appendAuthAudit(r.Context(), "anonymous", "login_failed", "auth", map[string]any{"reason": "invalid_payload"})
		httpHandler.WriteError(w, http.StatusBadRequest, err)
		return
	}
	account := strings.TrimSpace(req.Account)
	password := strings.TrimSpace(req.Password)
	if account == "" || password == "" {
		h.appendAuthAudit(r.Context(), account, "login_failed", "auth", map[string]any{"reason": "missing_credentials", "account": account})
		httpHandler.WriteMessage(w, http.StatusBadRequest, "invalid_credentials", "account and password are required")
		return
	}

	entity, err := h.usersRepo.GetByAccount(r.Context(), account)
	if err != nil {
		httpHandler.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	if entity == nil || entity.Status != domainuser.StatusActive || !domainuser.VerifyPassword(password, entity.Password) {
		h.appendAuthAudit(r.Context(), account, "login_failed", "auth", map[string]any{"account": account})
		httpHandler.WriteMessage(w, http.StatusUnauthorized, "invalid_credentials", "invalid account or password")
		return
	}

	roleKey, permissions, err := h.resolveRoleAndPermissions(r.Context(), entity.ID.String())
	if err != nil {
		httpHandler.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	token, err := security.SignJWT(h.jwtSecret, entity.ID.String(), roleKey, time.Hour, time.Now().UTC())
	if err != nil {
		httpHandler.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	httpHandler.WriteJSON(w, http.StatusOK, map[string]any{
		"token":       token,
		"tokenType":   "Bearer",
		"expiresIn":   3600,
		"permissions": permissions,
		"user": map[string]any{
			"id":      entity.ID.String(),
			"account": entity.Account,
			"name":    entity.Name,
			"email":   entity.Email.String(),
			"role":    roleKey,
		},
	})
	h.appendAuthAudit(r.Context(), entity.ID.String(), "login", "auth", map[string]any{"account": entity.Account, "role": roleKey})
}

func (h *builtinAuthHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	actorID := "anonymous"
	account := ""
	if claims, ok := security.JWTClaimsFromContext(r.Context()); ok {
		if subject := strings.TrimSpace(claims.Subject); subject != "" {
			actorID = subject
			account = h.resolveAccountBySubject(r.Context(), subject)
		}
	}
	detail := map[string]any{"source": "builtin-auth"}
	if account != "" {
		detail["account"] = account
	}
	h.appendAuthAudit(r.Context(), actorID, "logout", "auth", detail)
	httpHandler.WriteMessage(w, http.StatusOK, "ok", "logout success")
}

func (h *builtinAuthHandler) resolveAccountBySubject(ctx context.Context, subject string) string {
	if h == nil || h.usersRepo == nil {
		return ""
	}
	lookup := strings.TrimSpace(subject)
	if lookup == "" {
		return ""
	}
	if entity, err := h.usersRepo.GetByID(ctx, sharedID(lookup)); err == nil && entity != nil {
		return strings.TrimSpace(entity.Account)
	}
	if entity, err := h.usersRepo.GetByAccount(ctx, lookup); err == nil && entity != nil {
		return strings.TrimSpace(entity.Account)
	}
	return ""
}

func (h *builtinAuthHandler) appendAuthAudit(ctx context.Context, actorID, action, resource string, detail map[string]any) {
	if h == nil || h.auditSvc == nil {
		return
	}
	targetActor := strings.TrimSpace(actorID)
	if targetActor == "" {
		targetActor = "anonymous"
	}
	if _, err := h.auditSvc.Append(ctx, targetActor, action, resource, "", detail); err != nil {
		if h.logger != nil {
			h.logger.Warn("append auth audit failed", "actor", targetActor, "action", action, "resource", resource, "error", err)
		}
	}
}

func (h *builtinAuthHandler) handleMe(w http.ResponseWriter, r *http.Request) {
	entity, claims, ok := h.userFromRequest(r)
	if !ok {
		httpHandler.WriteMessage(w, http.StatusUnauthorized, "unauthorized", "invalid session")
		return
	}

	roleKey := claims.Role
	permissions, err := h.permissionsForRole(r.Context(), entity.ID.String(), roleKey)
	if err != nil {
		httpHandler.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	httpHandler.WriteJSON(w, http.StatusOK, map[string]any{
		"id":          entity.ID.String(),
		"account":     entity.Account,
		"name":        entity.Name,
		"email":       entity.Email.String(),
		"role":        roleKey,
		"permissions": permissions,
	})
}

func (h *builtinAuthHandler) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	entity, _, ok := h.userFromRequest(r)
	if !ok {
		httpHandler.WriteMessage(w, http.StatusUnauthorized, "unauthorized", "invalid session")
		return
	}

	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpHandler.WriteError(w, http.StatusBadRequest, err)
		return
	}

	now := time.Now().UTC()
	if v := strings.TrimSpace(req.Name); v != "" {
		if err := entity.Rename(v, now); err != nil {
			httpHandler.WriteError(w, http.StatusBadRequest, err)
			return
		}
	}
	if v := strings.TrimSpace(req.Email); v != "" {
		if err := entity.ChangeEmail(v, now); err != nil {
			httpHandler.WriteError(w, http.StatusBadRequest, err)
			return
		}
	}

	if err := h.usersRepo.Save(r.Context(), entity); err != nil {
		httpHandler.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	httpHandler.WriteJSON(w, http.StatusOK, map[string]any{
		"id":      entity.ID.String(),
		"account": entity.Account,
		"name":    entity.Name,
		"email":   entity.Email.String(),
	})
}

func (h *builtinAuthHandler) handleUpdatePassword(w http.ResponseWriter, r *http.Request) {
	entity, _, ok := h.userFromRequest(r)
	if !ok {
		httpHandler.WriteMessage(w, http.StatusUnauthorized, "unauthorized", "invalid session")
		return
	}

	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
		ConfirmPassword string `json:"confirmPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpHandler.WriteError(w, http.StatusBadRequest, err)
		return
	}

	if !domainuser.VerifyPassword(req.CurrentPassword, entity.Password) {
		httpHandler.WriteMessage(w, http.StatusBadRequest, "invalid_password", "current password is incorrect")
		return
	}
	if strings.TrimSpace(req.NewPassword) == "" || len(req.NewPassword) < 8 {
		httpHandler.WriteMessage(w, http.StatusBadRequest, "invalid_password", "new password must be at least 8 characters")
		return
	}
	if req.NewPassword != req.ConfirmPassword {
		httpHandler.WriteMessage(w, http.StatusBadRequest, "invalid_password", "password confirmation mismatch")
		return
	}

	hash, err := domainuser.HashPassword(req.NewPassword)
	if err != nil {
		httpHandler.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if err := entity.SetPasswordHash(hash.String()); err != nil {
		httpHandler.WriteError(w, http.StatusBadRequest, err)
		return
	}
	entity.Meta.Touch(time.Now().UTC())

	if err := h.usersRepo.Save(r.Context(), entity); err != nil {
		httpHandler.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	httpHandler.WriteMessage(w, http.StatusOK, "ok", "password updated")
}

func (h *builtinAuthHandler) userFromRequest(r *http.Request) (*domainuser.User, security.JWTClaims, bool) {
	claims, ok := security.JWTClaimsFromContext(r.Context())
	if !ok {
		return nil, security.JWTClaims{}, false
	}

	entity, err := h.usersRepo.GetByID(r.Context(), sharedID(claims.Subject))
	if err != nil {
		return nil, security.JWTClaims{}, false
	}
	if entity == nil {
		entity, err = h.usersRepo.GetByAccount(r.Context(), claims.Subject)
		if err != nil || entity == nil {
			return nil, security.JWTClaims{}, false
		}
	}
	return entity, claims, true
}

func (h *builtinAuthHandler) resolveRoleAndPermissions(ctx context.Context, userID string) (string, []string, error) {
	bindings, err := h.rbacRepo.ListBindingsBySubject(ctx, domainrbac.SubjectUser, sharedID(userID))
	if err != nil {
		return "", nil, err
	}
	if len(bindings) == 0 {
		return "user", []string{"user.read"}, nil
	}

	role, err := h.rolesRepo.GetByID(ctx, bindings[0].RoleID)
	if err != nil {
		return "", nil, err
	}
	if role == nil {
		return "user", []string{"user.read"}, nil
	}

	permissions, err := h.permissionsForRole(ctx, userID, role.Key)
	if err != nil {
		return "", nil, err
	}
	return role.Key, permissions, nil
}

func (h *builtinAuthHandler) permissionsForRole(ctx context.Context, userID, roleKey string) ([]string, error) {
	result := map[string]struct{}{}

	bindings, err := h.rbacRepo.ListBindingsBySubject(ctx, domainrbac.SubjectUser, sharedID(userID))
	if err != nil {
		return nil, err
	}
	for _, binding := range bindings {
		role, err := h.rolesRepo.GetByID(ctx, binding.RoleID)
		if err != nil || role == nil {
			continue
		}
		for _, permission := range role.Permissions {
			result[permission] = struct{}{}
		}

		rules, err := h.rbacRepo.ListPolicyRulesByRoleID(ctx, role.ID)
		if err != nil {
			return nil, err
		}
		for _, rule := range rules {
			if rule.Effect == domainrbac.EffectAllow {
				result[fmt.Sprintf("%s.%s", strings.TrimSpace(rule.Resource), strings.TrimSpace(rule.Action))] = struct{}{}
			}
		}
	}

	if strings.EqualFold(strings.TrimSpace(roleKey), "super_admin") {
		result["*"] = struct{}{}
	}

	permissions := make([]string, 0, len(result))
	for key := range result {
		permissions = append(permissions, key)
	}
	sort.Strings(permissions)
	return permissions, nil
}

func sharedID(raw string) shared.ID {
	return shared.ID(strings.TrimSpace(raw))
}
