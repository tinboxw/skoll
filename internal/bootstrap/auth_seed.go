package bootstrap

import (
	"context"
	"strings"
	"time"

	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	domainrole "github.com/tinboxw/skoll/internal/domain/role"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainuser "github.com/tinboxw/skoll/internal/domain/user"
	rbacrepo "github.com/tinboxw/skoll/internal/repository/rbac"
	rolerepo "github.com/tinboxw/skoll/internal/repository/role"
	userrepo "github.com/tinboxw/skoll/internal/repository/user"
	"github.com/tinboxw/skoll/pkg/logging"
)

type seedRole struct {
	id          string
	name        string
	key         string
	description string
	permissions []string
	builtIn     bool
	rules       []domainrbac.PolicyRule
}

type seedUser struct {
	id       string
	account  string
	name     string
	email    string
	password string
	roleKey  string
}

func ensureBuiltinAuthData(ctx context.Context, logger logging.Logger, usersRepo userrepo.UserRepository, rolesRepo rolerepo.RoleRepository, rbacRepo rbacrepo.RBACRepository) {
	if usersRepo == nil || rolesRepo == nil || rbacRepo == nil {
		return
	}

	roles := []seedRole{
		{
			id:          "seed-role-super-admin",
			name:        "Super Admin",
			key:         "super_admin",
			description: "system super administrator",
			permissions: []string{"*", "user.read", "user.create", "user.update", "user.delete", "role.read", "role.create", "role.update", "role.delete", "permission.manage"},
			builtIn:     true,
			rules:       []domainrbac.PolicyRule{},
		},
		{
			id:          "seed-role-dept-admin",
			name:        "Department Admin",
			key:         "dept_admin",
			description: "department level administrator",
			permissions: []string{"user.read", "user.update", "role.read"},
			builtIn:     true,
			rules: []domainrbac.PolicyRule{
				{Resource: "user", Action: "read", Effect: domainrbac.EffectAllow, Scope: domainrbac.DataScopeDeptTree},
				{Resource: "user", Action: "update", Effect: domainrbac.EffectAllow, Scope: domainrbac.DataScopeDeptTree},
				{Resource: "role", Action: "read", Effect: domainrbac.EffectAllow, Scope: domainrbac.DataScopeAll},
			},
		},
		{
			id:          "seed-role-operator",
			name:        "System Operator",
			key:         "operator",
			description: "system operation account",
			permissions: []string{"user.read", "role.read"},
			builtIn:     true,
			rules: []domainrbac.PolicyRule{
				{Resource: "user", Action: "read", Effect: domainrbac.EffectAllow, Scope: domainrbac.DataScopeAll},
				{Resource: "role", Action: "read", Effect: domainrbac.EffectAllow, Scope: domainrbac.DataScopeAll},
			},
		},
		{
			id:          "seed-role-user",
			name:        "Standard User",
			key:         "user",
			description: "standard business user",
			permissions: []string{"user.read"},
			builtIn:     true,
			rules: []domainrbac.PolicyRule{
				{Resource: "user", Action: "read", Effect: domainrbac.EffectAllow, Scope: domainrbac.DataScopeSelf},
			},
		},
	}

	users := []seedUser{
		{id: "seed-user-admin", account: "admin", name: "System Admin", email: "admin@skoll.local", password: "Admin@123456", roleKey: "super_admin"},
		{id: "seed-user-dept-admin", account: "dept_admin", name: "Dept Admin", email: "dept_admin@skoll.local", password: "Dept@123456", roleKey: "dept_admin"},
		{id: "seed-user-operator", account: "sys_operator", name: "System Operator", email: "operator@skoll.local", password: "Ops@123456", roleKey: "operator"},
		{id: "seed-user-normal", account: "normal_user", name: "Normal User", email: "user@skoll.local", password: "User@123456", roleKey: "user"},
	}

	roleByKey := map[string]*domainrole.Role{}
	for _, item := range roles {
		current, err := rolesRepo.GetByKey(ctx, item.key)
		if err != nil {
			logger.Warn("seed role load failed", "role", item.key, "error", err)
			continue
		}
		if current == nil {
			entity, err := domainrole.New(shared.ID(item.id), item.name, item.key, item.description, item.permissions, item.builtIn, time.Now().UTC())
			if err != nil {
				logger.Warn("seed role build failed", "role", item.key, "error", err)
				continue
			}
			if err := rolesRepo.Save(ctx, entity); err != nil {
				logger.Warn("seed role save failed", "role", item.key, "error", err)
				continue
			}
			current = entity
		}
		roleByKey[item.key] = current

		existingRules, err := rbacRepo.ListPolicyRulesByRoleID(ctx, current.ID)
		if err == nil && len(existingRules) == 0 && len(item.rules) > 0 {
			if err := rbacRepo.ReplacePolicyRules(ctx, current.ID, item.rules); err != nil {
				logger.Warn("seed role policy failed", "role", item.key, "error", err)
			}
		}
	}

	for _, item := range users {
		role := roleByKey[item.roleKey]
		if role == nil {
			logger.Warn("seed user skipped because role missing", "account", item.account, "role", item.roleKey)
			continue
		}

		hash, err := domainuser.HashPassword(item.password)
		if err != nil {
			logger.Warn("seed user password failed", "account", item.account, "error", err)
			continue
		}

		current, err := usersRepo.GetByAccount(ctx, item.account)
		if err != nil {
			logger.Warn("seed user load failed", "account", item.account, "error", err)
			continue
		}
		if current == nil {
			entity, err := domainuser.New(shared.ID(item.id), item.account, item.name, item.email, time.Now().UTC())
			if err != nil {
				logger.Warn("seed user build failed", "account", item.account, "error", err)
				continue
			}
			if err := entity.SetPasswordHash(hash.String()); err != nil {
				logger.Warn("seed user password set failed", "account", item.account, "error", err)
				continue
			}
			if err := usersRepo.Save(ctx, entity); err != nil {
				logger.Warn("seed user save failed", "account", item.account, "error", err)
				continue
			}
			current = entity
		}

		shouldSave := false
		if !strings.EqualFold(strings.TrimSpace(current.Password.String()), strings.TrimSpace(hash.String())) {
			if err := current.SetPasswordHash(hash.String()); err != nil {
				logger.Warn("seed user password reset failed", "account", item.account, "error", err)
				continue
			}
			shouldSave = true
		}
		if current.Status != domainuser.StatusActive {
			current.Activate(time.Now().UTC())
			shouldSave = true
		}
		if shouldSave {
			if err := usersRepo.Save(ctx, current); err != nil {
				logger.Warn("seed user update failed", "account", item.account, "error", err)
				continue
			}
		}

		bindings, err := rbacRepo.ListBindingsBySubject(ctx, domainrbac.SubjectUser, current.ID)
		if err != nil {
			logger.Warn("seed user bindings load failed", "account", item.account, "error", err)
			continue
		}
		hasBinding := false
		for _, binding := range bindings {
			if strings.EqualFold(binding.RoleID.String(), role.ID.String()) {
				hasBinding = true
				break
			}
		}
		if hasBinding {
			continue
		}

		binding, err := domainrbac.NewBinding(
			shared.ID("seed-binding-"+item.account+"-"+item.roleKey),
			domainrbac.SubjectUser,
			current.ID,
			role.ID,
			domainrbac.DataScopeAll,
			time.Now().UTC(),
		)
		if err != nil {
			logger.Warn("seed user binding build failed", "account", item.account, "error", err)
			continue
		}
		if err := rbacRepo.CreateBinding(ctx, binding); err != nil {
			logger.Warn("seed user binding save failed", "account", item.account, "error", err)
		}
	}
}
