package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainuser "github.com/tinboxw/skoll/internal/domain/user"
	httpHandler "github.com/tinboxw/skoll/internal/handler/http"
	organizationrepo "github.com/tinboxw/skoll/internal/repository/organization"
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
	orgRepo   organizationrepo.OrganizationRepository
	auditSvc  auditsvc.Service
	eventSvc  auditsvc.EventService
	logger    logging.Logger
	nowFn     func() time.Time
	idFn      func(prefix string) shared.ID
}

func newBuiltinAuthHandler(jwtSecret string, usersRepo userrepo.UserRepository, rolesRepo rolerepo.RoleRepository, rbacRepo rbacrepo.RBACRepository, orgRepo organizationrepo.OrganizationRepository, auditSvc auditsvc.Service, eventSvc auditsvc.EventService, logger logging.Logger) *builtinAuthHandler {
	if usersRepo == nil || rolesRepo == nil || rbacRepo == nil || orgRepo == nil {
		return nil
	}
	return &builtinAuthHandler{
		jwtSecret: jwtSecret,
		usersRepo: usersRepo,
		rolesRepo: rolesRepo,
		rbacRepo:  rbacRepo,
		orgRepo:   orgRepo,
		auditSvc:  auditSvc,
		eventSvc:  eventSvc,
		logger:    logger,
		nowFn:     func() time.Time { return time.Now().UTC() },
		idFn: func(prefix string) shared.ID {
			return shared.ID(prefix + "-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10))
		},
	}
}

func (h *builtinAuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.appendLoginAudit(r, "anonymous", "anonymous", domainaudit.LoginResultFailure, "invalid_payload", "", nil)
		httpHandler.WriteMessage(w, http.StatusBadRequest, "invalid_auth_request", "authentication request is invalid")
		return
	}
	account := strings.TrimSpace(req.Account)
	password := strings.TrimSpace(req.Password)
	if account == "" || password == "" {
		h.appendLoginAudit(r, account, account, domainaudit.LoginResultFailure, "missing_credentials", "", map[string]any{"account": account})
		httpHandler.WriteMessage(w, http.StatusBadRequest, "invalid_credentials", "account and password are required")
		return
	}

	entity, err := h.usersRepo.GetByAccount(r.Context(), account)
	if err != nil {
		httpHandler.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	if entity == nil || entity.Status != domainuser.StatusActive || !domainuser.VerifyPassword(password, entity.Password) {
		h.appendLoginAudit(r, account, account, domainaudit.LoginResultFailure, "invalid_credentials", "", map[string]any{"account": account})
		httpHandler.WriteMessage(w, http.StatusUnauthorized, "invalid_credentials", "invalid account or password")
		return
	}

	roleKey, roles, permissions, err := h.resolveRolesAndPermissions(r.Context(), entity.ID.String())
	if err != nil {
		httpHandler.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	organizationID, organizationPath, err := h.resolveOrganizationClaims(r.Context(), entity)
	if err != nil {
		httpHandler.WriteMessage(w, http.StatusUnauthorized, "invalid_organization", "user organization is invalid")
		return
	}

	token, err := security.SignJWT(h.jwtSecret, security.JWTIdentity{
		Subject:          entity.ID.String(),
		OrganizationID:   organizationID,
		OrganizationPath: organizationPath,
		Role:             roleKey,
		Roles:            roles,
	}, time.Hour, h.nowFn())
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
			"id":               entity.ID.String(),
			"account":          entity.Account,
			"name":             entity.Name,
			"email":            entity.Email.String(),
			"role":             roleKey,
			"roles":            roles,
			"organizationId":   organizationID,
			"organizationPath": organizationPath,
		},
	})
	h.appendLoginAudit(r, entity.ID.String(), entity.Account, domainaudit.LoginResultSuccess, "", "", map[string]any{"account": entity.Account, "organizationId": organizationID, "roles": roles})
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

func (h *builtinAuthHandler) appendLoginAudit(r *http.Request, actorID, account string, result domainaudit.LoginResult, failureReason, sessionID string, metadata map[string]any) {
	if h == nil || h.eventSvc == nil || r == nil {
		return
	}
	now := h.nowFn()
	eventID := h.idFn("audit-event")
	logID := h.idFn("login-log")
	if strings.TrimSpace(sessionID) == "" && result == domainaudit.LoginResultSuccess {
		sessionID = eventID.String()
	}
	normalizedAccount := strings.TrimSpace(account)
	if normalizedAccount == "" {
		normalizedAccount = "anonymous"
	}
	trace := domainaudit.TraceContext{
		TraceID:   strings.TrimSpace(r.Header.Get("X-Trace-Id")),
		RequestID: strings.TrimSpace(r.Header.Get("X-Request-Id")),
		Method:    r.Method,
		Path:      r.URL.Path,
		IP:        authRequestIP(r),
		UserAgent: r.UserAgent(),
	}
	loginLog, err := domainaudit.NewLoginLog(domainaudit.LoginLogInput{
		ID:            logID,
		Account:       normalizedAccount,
		ActorID:       shared.ID(strings.TrimSpace(actorID)),
		Result:        result,
		IP:            trace.IP,
		UserAgent:     trace.UserAgent,
		FailureReason: failureReason,
		SessionID:     sessionID,
		Trace:         trace,
		Metadata:      metadata,
		OccurredAt:    now,
	})
	if err != nil {
		h.logAuditEventError("build login audit failed", actorID, string(result), err)
		return
	}

	action := domainaudit.AuditAction("auth.session.login")
	eventResult := domainaudit.EventResultSuccess
	risk := domainaudit.EventRiskLow
	if result == domainaudit.LoginResultFailure {
		action = domainaudit.AuditAction("auth.session.login_failed")
		eventResult = domainaudit.EventResultFailure
		risk = domainaudit.EventRiskMedium
	}
	actor := domainaudit.ActorRef{Type: "user", ID: shared.ID(strings.TrimSpace(actorID)), Name: normalizedAccount}
	if actor.ID.IsZero() || result == domainaudit.LoginResultFailure {
		actor.Type = "account"
		actor.ID = shared.ID(normalizedAccount)
	}
	event, err := domainaudit.NewEvent(domainaudit.EventInput{
		ID:       eventID,
		Type:     domainaudit.EventTypeLogin,
		Action:   action,
		Actor:    actor,
		Resource: domainaudit.ResourceRef{Type: "auth_session", ID: sessionID, Name: normalizedAccount},
		Result:   eventResult,
		Trace:    trace,
		Risk:     risk,
		Metadata: metadata,
		SourceData: map[string]any{
			"kind":          "login_log",
			"id":            loginLog.ID.String(),
			"account":       loginLog.Account,
			"actorId":       loginLog.ActorID.String(),
			"result":        string(loginLog.Result),
			"ip":            loginLog.IP,
			"userAgent":     loginLog.UserAgent,
			"failureReason": loginLog.FailureReason,
			"sessionId":     loginLog.SessionID,
			"metadata":      loginLog.Metadata,
			"occurredAt":    loginLog.OccurredAt.Format(time.RFC3339Nano),
		},
		OccurredAt: now,
	})
	if err != nil {
		h.logAuditEventError("build login event failed", actorID, action.String(), err)
		return
	}
	if err := h.eventSvc.AppendEvent(r.Context(), event); err != nil {
		h.logAuditEventError("append login audit failed", actorID, action.String(), err)
	}
}

func (h *builtinAuthHandler) logAuditEventError(message, actorID, action string, err error) {
	if h != nil && h.logger != nil && err != nil {
		h.logger.Warn(message, "actor", actorID, "action", action, "error", err)
	}
}

func authRequestIP(r *http.Request) string {
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

func (h *builtinAuthHandler) handleMe(w http.ResponseWriter, r *http.Request) {
	entity, claims, ok := h.userFromRequest(r)
	if !ok {
		httpHandler.WriteMessage(w, http.StatusUnauthorized, "unauthorized", "invalid session")
		return
	}

	permissions, err := h.permissionsForRoles(r.Context(), entity.ID.String(), claims.Roles)
	if err != nil {
		httpHandler.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	httpHandler.WriteJSON(w, http.StatusOK, map[string]any{
		"id":               entity.ID.String(),
		"account":          entity.Account,
		"name":             entity.Name,
		"email":            entity.Email.String(),
		"role":             claims.Role,
		"roles":            claims.Roles,
		"organizationId":   claims.OrganizationID,
		"organizationPath": claims.OrganizationPath,
		"permissions":      permissions,
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

func (h *builtinAuthHandler) resolveRolesAndPermissions(ctx context.Context, userID string) (string, []string, []string, error) {
	bindings, err := h.rbacRepo.ListBindingsBySubject(ctx, domainrbac.SubjectUser, sharedID(userID))
	if err != nil {
		return "", nil, nil, err
	}
	roleSet := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		role, roleErr := h.rolesRepo.GetByID(ctx, binding.RoleID)
		if roleErr != nil {
			return "", nil, nil, roleErr
		}
		if role == nil || strings.TrimSpace(role.Key) == "" {
			continue
		}
		roleSet[strings.TrimSpace(role.Key)] = struct{}{}
	}
	roles := make([]string, 0, len(roleSet))
	for roleKey := range roleSet {
		roles = append(roles, roleKey)
	}
	if len(roles) == 0 {
		return "user", []string{"user"}, []string{"user.read"}, nil
	}
	sort.Strings(roles)
	primaryRole := roles[0]
	for _, roleKey := range roles {
		if strings.EqualFold(roleKey, "super_admin") {
			primaryRole = roleKey
			break
		}
	}
	permissions, err := h.permissionsForRoles(ctx, userID, roles)
	if err != nil {
		return "", nil, nil, err
	}
	return primaryRole, roles, permissions, nil
}

func (h *builtinAuthHandler) permissionsForRoles(ctx context.Context, userID string, roles []string) ([]string, error) {
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

	for _, roleKey := range roles {
		if strings.EqualFold(strings.TrimSpace(roleKey), "super_admin") {
			result["*"] = struct{}{}
			break
		}
	}

	permissions := make([]string, 0, len(result))
	for key := range result {
		permissions = append(permissions, key)
	}
	sort.Strings(permissions)
	return permissions, nil
}

func (h *builtinAuthHandler) resolveOrganizationClaims(ctx context.Context, entity *domainuser.User) (string, []string, error) {
	if entity == nil {
		return "", nil, fmt.Errorf("user is required")
	}
	organizationID := strings.TrimSpace(entity.DepartmentID)
	if organizationID == "" {
		return "", []string{}, nil
	}

	path := make([]string, 0, 4)
	seen := map[shared.ID]struct{}{}
	currentID := sharedID(organizationID)
	for !currentID.IsZero() {
		if _, ok := seen[currentID]; ok {
			return "", nil, fmt.Errorf("organization path contains cycle: %s", currentID)
		}
		seen[currentID] = struct{}{}
		department, err := h.orgRepo.GetDepartmentByID(ctx, currentID)
		if err != nil {
			return "", nil, err
		}
		if department == nil {
			return "", nil, fmt.Errorf("organization does not exist: %s", currentID)
		}
		path = append(path, department.ID.String())
		currentID = department.ParentID
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return organizationID, path, nil
}

func sharedID(raw string) shared.ID {
	return shared.ID(strings.TrimSpace(raw))
}
