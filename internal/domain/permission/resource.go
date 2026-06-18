package permission

import "strings"

type ResourceType string

const (
	ResourceTypeAPI       ResourceType = "api"
	ResourceTypeMenu      ResourceType = "menu"
	ResourceTypeButton    ResourceType = "button"
	ResourceTypeDataScope ResourceType = "data_scope"
	ResourceTypePlugin    ResourceType = "plugin"
)

type ResourceIdentity struct {
	Key    string
	Type   ResourceType
	Module string
	Source string
}

type PermissionResource struct {
	Identity ResourceIdentity
	Name     string
	Risk     RiskLevel
	Metadata map[string]string
	Enabled  bool
}

func NewResource(identity ResourceIdentity, name string) (PermissionResource, error) {
	return NewResourceWithMetadata(identity, name, RiskLevelLow, nil)
}

func NewResourceWithMetadata(identity ResourceIdentity, name string, risk RiskLevel, metadata map[string]string) (PermissionResource, error) {
	identity = NormalizeIdentity(identity)
	name = strings.TrimSpace(name)
	if err := ValidateResource(identity, name); err != nil {
		return PermissionResource{}, err
	}
	if err := ValidateRisk(risk); err != nil {
		return PermissionResource{}, err
	}
	metadata = NormalizeMetadata(metadata)
	if err := ValidateMetadata(metadata); err != nil {
		return PermissionResource{}, err
	}
	return PermissionResource{
		Identity: identity,
		Name:     name,
		Risk:     risk,
		Metadata: metadata,
		Enabled:  true,
	}, nil
}

func (r PermissionResource) Key() string {
	return r.Identity.Key
}

func (r PermissionResource) Type() ResourceType {
	return r.Identity.Type
}

func (r PermissionResource) Module() string {
	return r.Identity.Module
}

func (r PermissionResource) Source() string {
	return r.Identity.Source
}
