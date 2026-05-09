package rbac

import domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"

type BindRoleInput struct {
	SubjectType domainrbac.SubjectType
	SubjectID   string
	RoleID      string
	Scope       domainrbac.DataScope
}

type SetRolePoliciesInput struct {
	RoleID string
	Rules  []domainrbac.PolicyRule
}

type CheckPermissionInput struct {
	SubjectType domainrbac.SubjectType
	SubjectID   string
	Resource    string
	Action      string
}
