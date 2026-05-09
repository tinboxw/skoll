package rbac

import "github.com/tinboxw/skoll/internal/domain/shared"

type Permission struct {
	Resource string
	Action   string
}

type Binding struct {
	RoleID      shared.ID
	Permissions []Permission
	DataScope   DataScope
}
