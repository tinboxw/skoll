package permission

import domainpermission "github.com/tinboxw/skoll/internal/domain/permission"

type RegisterResourceInput struct {
	Key      string
	Type     domainpermission.ResourceType
	Module   string
	Source   string
	Name     string
	Risk     domainpermission.RiskLevel
	Metadata map[string]string
}

type ListResourcesInput struct {
	Type    domainpermission.ResourceType
	Module  string
	Source  string
	Enabled *bool
	Offset  int
	Limit   int
}

type DiffResourcesInput struct {
	Source  string
	Desired []RegisterResourceInput
}

type DiffResult struct {
	Added   []domainpermission.PermissionResource
	Updated []domainpermission.PermissionResource
	Removed []domainpermission.PermissionResource
}
