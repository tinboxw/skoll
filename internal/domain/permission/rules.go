package permission

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	resourceKeyPattern    = regexp.MustCompile(`^[a-z][a-z0-9_:.\-]{1,127}$`)
	resourceModulePattern = regexp.MustCompile(`^[a-z][a-z0-9_.\-]{0,63}$`)
)

func NormalizeIdentity(identity ResourceIdentity) ResourceIdentity {
	return ResourceIdentity{
		Key:    strings.TrimSpace(strings.ToLower(identity.Key)),
		Type:   identity.Type,
		Module: strings.TrimSpace(strings.ToLower(identity.Module)),
		Source: strings.TrimSpace(strings.ToLower(identity.Source)),
	}
}

func ValidateResource(identity ResourceIdentity, name string) error {
	if err := ValidateKey(identity.Key); err != nil {
		return err
	}
	if err := ValidateType(identity.Type); err != nil {
		return err
	}
	if err := ValidateModule(identity.Module); err != nil {
		return err
	}
	if err := ValidateSource(identity.Source); err != nil {
		return err
	}
	return ValidateName(name)
}

func ValidateKey(key string) error {
	v := strings.TrimSpace(key)
	if !resourceKeyPattern.MatchString(v) {
		return fmt.Errorf("permission key must match %s", resourceKeyPattern.String())
	}
	return nil
}

func ValidateType(resourceType ResourceType) error {
	switch resourceType {
	case ResourceTypeAPI, ResourceTypeMenu, ResourceTypeButton, ResourceTypeDataScope, ResourceTypePlugin:
		return nil
	default:
		return fmt.Errorf("permission type is invalid")
	}
}

func ValidateModule(module string) error {
	v := strings.TrimSpace(module)
	if !resourceModulePattern.MatchString(v) {
		return fmt.Errorf("permission module must match %s", resourceModulePattern.String())
	}
	return nil
}

func ValidateSource(source string) error {
	v := strings.TrimSpace(source)
	if !resourceModulePattern.MatchString(v) {
		return fmt.Errorf("permission source must match %s", resourceModulePattern.String())
	}
	return nil
}

func ValidateName(name string) error {
	v := strings.TrimSpace(name)
	if v == "" {
		return fmt.Errorf("permission name is required")
	}
	if len([]rune(v)) > 128 {
		return fmt.Errorf("permission name is too long")
	}
	return nil
}
