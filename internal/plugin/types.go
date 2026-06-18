package plugin

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	ErrPluginNotFound        = errors.New("plugin not found")
	ErrPluginAlreadyExists   = errors.New("plugin already exists")
	ErrPluginInvalidState    = errors.New("plugin state transition is invalid")
	ErrPluginManifestBroken  = errors.New("plugin manifest is invalid")
	ErrPluginDependency      = errors.New("plugin dependency error")
	ErrPluginSystemProtected = errors.New("system builtin plugin cannot be disabled or uninstalled")
)

type State string

const (
	StateInstalled   State = "installed"
	StateEnabled     State = "enabled"
	StateDisabled    State = "disabled"
	StateUninstalled State = "uninstalled"
)

type Dependency struct {
	ID      string
	Version string
}

type UIMode string

const (
	UIModeBackendOnly  UIMode = "backend_only"
	UIModeFrontendOnly UIMode = "frontend_only"
	UIModeMonolith     UIMode = "monolith"
	UIModeSeparated    UIMode = "separated"
)

type Level string

const (
	LevelSystem Level = "system"
	LevelApp    Level = "app"
)

type MountPolicy string

const (
	MountPolicyAdmin MountPolicy = "admin"
	MountPolicyUser  MountPolicy = "user"
	MountPolicyMixed MountPolicy = "mixed"
)

type UINavPosition string

const (
	UINavPositionNone    UINavPosition = "none"
	UINavPositionSidebar UINavPosition = "sidebar"
	UINavPositionTopTab  UINavPosition = "top_tab"
)

type UIOpenMode string

const (
	UIOpenModeIntegrated UIOpenMode = "integrated"
	UIOpenModeStandalone UIOpenMode = "standalone"
)

type UITabMode string

const (
	UITabModeOptional UITabMode = "optional"
	UITabModeFixed    UITabMode = "fixed"
	UITabModeDisabled UITabMode = "disabled"
)

var appIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,62}$`)
var apiVersionPattern = regexp.MustCompile(`^v[0-9]+$`)
var migrationVersionPattern = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+$`)
var localePattern = regexp.MustCompile(`^[a-z]{2}(?:-[A-Z]{2})?$`)
var permissionKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_:.\-]{1,127}$`)
var permissionModulePattern = regexp.MustCompile(`^[a-z][a-z0-9_.\-]{0,63}$`)

type SignatureAlgorithm string

const (
	SigAlgoRSASHA256 SignatureAlgorithm = "RSA-SHA256"
)

type Signature struct {
	Algorithm SignatureAlgorithm
	Timestamp time.Time
	Value     string
	VendorID  string
	PublicKey string
}

type UIMenu struct {
	Key                 string
	ParentKey           string
	Label               string
	LabelZhCN           string
	LabelEnUS           string
	Path                string
	Component           string
	Icon                string
	Order               int
	Visible             *bool
	RequiredRoles       []string
	RequiredPermissions []string
}

type ConfigSchema struct {
	Title       string
	TitleZhCN   string
	TitleEnUS   string
	Description string
	Fields      []ConfigField
}

type ConfigField struct {
	Key         string
	Label       string
	LabelZhCN   string
	LabelEnUS   string
	Type        string
	Required    bool
	Default     string
	Placeholder string
	Help        string
	Min         *float64
	Max         *float64
	MinLength   *int
	MaxLength   *int
	Pattern     string
	Options     []ConfigOption
}

type ConfigOption struct {
	Label     string
	LabelZhCN string
	LabelEnUS string
	Value     string
}

type PermissionDeclaration struct {
	Key      string
	Type     string
	Module   string
	Name     string
	Risk     string
	Metadata map[string]string
}

type Info struct {
	ID                  string
	Name                string
	NameZhCN            string
	NameEnUS            string
	Version             string
	APIVersion          string
	CompatibilitySkoll  string
	ServiceBaseURL      string
	ServiceHealthURL    string
	MigrationVersion    string
	Description         string
	ConfigJSON          string
	Dependencies        []Dependency
	Permissions         []string
	PermissionResources []PermissionDeclaration
	State               State
	InstalledAt         time.Time
	EnabledAt           *time.Time
	Source              string
	UIMode              UIMode
	Level               Level
	AppID               string
	MountPolicy         MountPolicy
	FrontendEntry       string
	UINavPosition       UINavPosition
	UIOpenMode          UIOpenMode
	UITabMode           UITabMode
	I18nLocales         []string
	UIMenu              *UIMenu
	ConfigSchema        *ConfigSchema
	SystemBuiltin       bool
	Vendor              string
	VendorURL           string
	Signature           *Signature
}

func (i Info) ValidateManifest() error {
	if i.ID == "" || i.Name == "" || i.Version == "" {
		return ErrPluginManifestBroken
	}
	if v := strings.TrimSpace(i.APIVersion); v != "" && !apiVersionPattern.MatchString(v) {
		return ErrPluginManifestBroken
	}
	if mv := strings.TrimSpace(i.MigrationVersion); mv != "" && !migrationVersionPattern.MatchString(mv) {
		return ErrPluginManifestBroken
	}
	if cu := strings.TrimSpace(i.CompatibilitySkoll); cu == "" && strings.TrimSpace(i.APIVersion) != "" {
		return ErrPluginManifestBroken
	}
	if base := strings.TrimSpace(i.ServiceBaseURL); base != "" && !isHTTPURL(base) {
		return ErrPluginManifestBroken
	}
	if health := strings.TrimSpace(i.ServiceHealthURL); health != "" && !isHTTPURL(health) {
		return ErrPluginManifestBroken
	}

	for _, dep := range i.Dependencies {
		if dep.ID == "" {
			return ErrPluginManifestBroken
		}
	}
	if err := validatePermissionDeclarations(i.Permissions, i.PermissionResources); err != nil {
		return err
	}

	mode := i.UIMode
	if mode == "" {
		mode = UIModeBackendOnly
	}
	if mode != UIModeBackendOnly && mode != UIModeFrontendOnly && mode != UIModeMonolith && mode != UIModeSeparated {
		return ErrPluginManifestBroken
	}

	level := i.Level
	if level == "" {
		level = LevelSystem
	}
	if level != LevelSystem && level != LevelApp {
		return ErrPluginManifestBroken
	}
	appID := strings.TrimSpace(i.AppID)
	if level == LevelApp {
		if !appIDPattern.MatchString(appID) {
			return ErrPluginManifestBroken
		}
		if appID == "skoll" {
			return ErrPluginManifestBroken
		}
	} else {
		if appID != "" {
			return ErrPluginManifestBroken
		}
	}

	mp := i.MountPolicy
	if mp == "" {
		mp = MountPolicyAdmin
	}
	if mp != MountPolicyAdmin && mp != MountPolicyUser && mp != MountPolicyMixed {
		return ErrPluginManifestBroken
	}

	nav := i.UINavPosition
	if nav == "" {
		nav = UINavPositionNone
	}
	if nav != UINavPositionNone && nav != UINavPositionSidebar && nav != UINavPositionTopTab {
		return ErrPluginManifestBroken
	}

	openMode := i.UIOpenMode
	if openMode == "" {
		openMode = UIOpenModeIntegrated
	}
	if openMode != UIOpenModeIntegrated && openMode != UIOpenModeStandalone {
		return ErrPluginManifestBroken
	}

	tabMode := i.UITabMode
	if tabMode == "" {
		tabMode = UITabModeOptional
	}
	if tabMode != UITabModeOptional && tabMode != UITabModeFixed && tabMode != UITabModeDisabled {
		return ErrPluginManifestBroken
	}
	if openMode == UIOpenModeStandalone {
		if nav != UINavPositionNone {
			return ErrPluginManifestBroken
		}
		if tabMode != UITabModeDisabled {
			return ErrPluginManifestBroken
		}
	}
	if i.UIMenu != nil {
		if path := strings.TrimSpace(i.UIMenu.Path); path != "" && !strings.HasPrefix(path, "/") {
			return ErrPluginManifestBroken
		}
		for _, role := range i.UIMenu.RequiredRoles {
			if strings.TrimSpace(role) == "" {
				return ErrPluginManifestBroken
			}
		}
		for _, permission := range i.UIMenu.RequiredPermissions {
			if strings.TrimSpace(permission) == "" {
				return ErrPluginManifestBroken
			}
		}
		if _, err := i.MenuNodes(); err != nil {
			return ErrPluginManifestBroken
		}
	}
	if i.ConfigSchema != nil {
		seenFields := map[string]struct{}{}
		for _, field := range i.ConfigSchema.Fields {
			key := strings.TrimSpace(field.Key)
			if key == "" {
				return ErrPluginManifestBroken
			}
			if _, ok := seenFields[key]; ok {
				return ErrPluginManifestBroken
			}
			seenFields[key] = struct{}{}
			switch strings.TrimSpace(field.Type) {
			case "", "string", "textarea", "number", "boolean", "select":
			default:
				return ErrPluginManifestBroken
			}
			if field.Min != nil && field.Max != nil && *field.Min > *field.Max {
				return ErrPluginManifestBroken
			}
			if field.MinLength != nil && *field.MinLength < 0 {
				return ErrPluginManifestBroken
			}
			if field.MaxLength != nil && *field.MaxLength < 0 {
				return ErrPluginManifestBroken
			}
			if field.MinLength != nil && field.MaxLength != nil && *field.MinLength > *field.MaxLength {
				return ErrPluginManifestBroken
			}
			if strings.TrimSpace(field.Pattern) != "" {
				if _, err := regexp.Compile(strings.TrimSpace(field.Pattern)); err != nil {
					return ErrPluginManifestBroken
				}
			}
			if strings.TrimSpace(field.Type) == "select" && len(field.Options) == 0 {
				return ErrPluginManifestBroken
			}
			for _, option := range field.Options {
				if strings.TrimSpace(option.Value) == "" {
					return ErrPluginManifestBroken
				}
			}
		}
	}

	if mode != UIModeBackendOnly {
		if len(i.I18nLocales) == 0 {
			return ErrPluginManifestBroken
		}
		seenLocales := make(map[string]struct{}, len(i.I18nLocales))
		for _, locale := range i.I18nLocales {
			trimmed := strings.TrimSpace(locale)
			if !localePattern.MatchString(trimmed) {
				return ErrPluginManifestBroken
			}
			if _, exists := seenLocales[trimmed]; exists {
				return ErrPluginManifestBroken
			}
			seenLocales[trimmed] = struct{}{}
		}
	}

	// 签名字段验证（如果有签名）
	if i.Signature != nil {
		if i.Signature.Algorithm != SigAlgoRSASHA256 {
			return ErrPluginManifestBroken
		}
		if strings.TrimSpace(i.Signature.Value) == "" {
			return ErrPluginManifestBroken
		}
		if i.Signature.Timestamp.IsZero() {
			return ErrPluginManifestBroken
		}
	}

	return nil
}

func validatePermissionDeclarations(keys []string, declarations []PermissionDeclaration) error {
	seen := map[string]struct{}{}
	if len(declarations) > 0 {
		for _, declaration := range declarations {
			key := strings.TrimSpace(strings.ToLower(declaration.Key))
			if !permissionKeyPattern.MatchString(key) {
				return ErrPluginManifestBroken
			}
			if _, ok := seen[key]; ok {
				return ErrPluginManifestBroken
			}
			seen[key] = struct{}{}
			resourceType := strings.TrimSpace(strings.ToLower(declaration.Type))
			switch resourceType {
			case "", "api", "menu", "button", "data_scope", "plugin":
			default:
				return ErrPluginManifestBroken
			}
			module := strings.TrimSpace(strings.ToLower(declaration.Module))
			if module != "" && !permissionModulePattern.MatchString(module) {
				return ErrPluginManifestBroken
			}
			risk := strings.TrimSpace(strings.ToLower(declaration.Risk))
			switch risk {
			case "", "low", "medium", "high", "critical":
			default:
				return ErrPluginManifestBroken
			}
			for metadataKey := range declaration.Metadata {
				if strings.TrimSpace(metadataKey) == "" {
					return ErrPluginManifestBroken
				}
			}
		}
		return nil
	}
	for _, key := range keys {
		normalized := strings.TrimSpace(strings.ToLower(key))
		if !permissionKeyPattern.MatchString(normalized) {
			return ErrPluginManifestBroken
		}
		if _, ok := seen[normalized]; ok {
			return ErrPluginManifestBroken
		}
		seen[normalized] = struct{}{}
	}
	return nil
}

func isHTTPURL(raw string) bool {
	u, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func NormalizeEntryPath(path string) string {
	v := strings.TrimSpace(path)
	if v == "" {
		return ""
	}
	if !strings.HasPrefix(v, "/") {
		v = "/" + v
	}
	v = strings.ReplaceAll(v, "//", "/")
	v = strings.TrimRight(v, "/")
	if v == "" {
		return "/"
	}
	return v
}

func ResolveFrontendEntry(info Info) string {
	explicit := NormalizeEntryPath(info.FrontendEntry)
	if explicit != "" {
		return explicit
	}
	if strings.TrimSpace(info.ID) == "" {
		return ""
	}

	level := info.Level
	if level == "" {
		level = LevelSystem
	}
	if level == LevelApp {
		appID := strings.TrimSpace(info.AppID)
		if appID == "" {
			return "/unknown"
		}
		return fmt.Sprintf("/%s", appID)
	}
	return fmt.Sprintf("/skoll/plugins/%s", info.ID)
}
