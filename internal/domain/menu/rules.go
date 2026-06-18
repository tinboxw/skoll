package menu

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	menuKeyPattern    = regexp.MustCompile(`^[a-z][a-z0-9_.\-]{1,127}$`)
	menuSourcePattern = regexp.MustCompile(`^[a-z][a-z0-9_.\-]{0,63}$`)
)

func NormalizeIdentity(identity NodeIdentity) NodeIdentity {
	return NodeIdentity{
		Key:       strings.TrimSpace(strings.ToLower(identity.Key)),
		ParentKey: strings.TrimSpace(strings.ToLower(identity.ParentKey)),
		Source:    strings.TrimSpace(strings.ToLower(identity.Source)),
	}
}

func NormalizeView(view NodeView) NodeView {
	return NodeView{
		Name:      strings.TrimSpace(view.Name),
		Path:      strings.TrimSpace(view.Path),
		Component: strings.TrimSpace(view.Component),
		Icon:      strings.TrimSpace(view.Icon),
	}
}

func ValidateNode(identity NodeIdentity, view NodeView, sort int) error {
	if err := ValidateKey(identity.Key); err != nil {
		return err
	}
	if err := ValidateParentKey(identity.ParentKey); err != nil {
		return err
	}
	if err := ValidateSource(identity.Source); err != nil {
		return err
	}
	if err := ValidateName(view.Name); err != nil {
		return err
	}
	if err := ValidatePath(view.Path); err != nil {
		return err
	}
	return ValidateSort(sort)
}

func ValidateKey(key string) error {
	v := strings.TrimSpace(key)
	if !menuKeyPattern.MatchString(v) {
		return fmt.Errorf("menu key must match %s", menuKeyPattern.String())
	}
	return nil
}

func ValidateParentKey(parentKey string) error {
	v := strings.TrimSpace(parentKey)
	if v == "" {
		return nil
	}
	if !menuKeyPattern.MatchString(v) {
		return fmt.Errorf("menu parent key must match %s", menuKeyPattern.String())
	}
	return nil
}

func ValidateSource(source string) error {
	v := strings.TrimSpace(source)
	if !menuSourcePattern.MatchString(v) {
		return fmt.Errorf("menu source must match %s", menuSourcePattern.String())
	}
	return nil
}

func ValidateName(name string) error {
	v := strings.TrimSpace(name)
	if v == "" {
		return fmt.Errorf("menu name is required")
	}
	if len([]rune(v)) > 128 {
		return fmt.Errorf("menu name is too long")
	}
	return nil
}

func ValidatePath(path string) error {
	v := strings.TrimSpace(path)
	if v == "" {
		return fmt.Errorf("menu path is required")
	}
	if !strings.HasPrefix(v, "/") {
		return fmt.Errorf("menu path must start with /")
	}
	if strings.ContainsAny(v, " \t\r\n") {
		return fmt.Errorf("menu path must not contain whitespace")
	}
	if len([]rune(v)) > 256 {
		return fmt.Errorf("menu path is too long")
	}
	return nil
}

func ValidateSort(sort int) error {
	if sort < 0 {
		return fmt.Errorf("menu sort must be non-negative")
	}
	return nil
}
