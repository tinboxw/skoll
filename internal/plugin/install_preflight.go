package plugin

import (
	"fmt"
	"sort"
	"strings"
)

const (
	InstallPreflightStatusPass    = "pass"
	InstallPreflightStatusBlocked = "blocked"
)

type InstallPreflightService struct {
	loader MetadataLoader
}

type InstallPreflightInput struct {
	Path        string
	CoreVersion string
	Installed   []Info
}

type InstallPreflightResult struct {
	Status      string                       `json:"status"`
	Plugin      InstallPreflightPlugin       `json:"plugin"`
	Blockers    []string                     `json:"blockers,omitempty"`
	Warnings    []string                     `json:"warnings,omitempty"`
	Permissions InstallPreflightResourceDiff `json:"permissions"`
	Menus       InstallPreflightMenuDiff     `json:"menus"`
	Config      InstallPreflightConfig       `json:"config"`
	Resources   InstallPreflightResources    `json:"resources"`
	Data        InstallPreflightData         `json:"data"`
	Migration   InstallPreflightMigration    `json:"migration"`
	Signature   InstallPreflightSignature    `json:"signature"`
	Risk        InstallPreflightRisk         `json:"risk"`
}

type InstallPreflightPlugin struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source"`
}

type InstallPreflightResourceDiff struct {
	Add      []InstallPreflightPermission `json:"add,omitempty"`
	Update   []InstallPreflightPermission `json:"update,omitempty"`
	Conflict []InstallPreflightPermission `json:"conflict,omitempty"`
}

type InstallPreflightPermission struct {
	Key      string `json:"key"`
	Type     string `json:"type,omitempty"`
	Module   string `json:"module,omitempty"`
	Name     string `json:"name,omitempty"`
	Risk     string `json:"risk,omitempty"`
	Source   string `json:"source,omitempty"`
	Existing string `json:"existing,omitempty"`
}

type InstallPreflightMenuDiff struct {
	Add      []InstallPreflightMenu `json:"add,omitempty"`
	Update   []InstallPreflightMenu `json:"update,omitempty"`
	Conflict []InstallPreflightMenu `json:"conflict,omitempty"`
}

type InstallPreflightMenu struct {
	Key                 string   `json:"key"`
	ParentKey           string   `json:"parentKey,omitempty"`
	Path                string   `json:"path,omitempty"`
	Name                string   `json:"name,omitempty"`
	Source              string   `json:"source,omitempty"`
	Existing            string   `json:"existing,omitempty"`
	RequiredRoles       []string `json:"requiredRoles,omitempty"`
	RequiredPermissions []string `json:"requiredPermissions,omitempty"`
}

type InstallPreflightConfig struct {
	HasSchema      bool     `json:"hasSchema"`
	FieldCount     int      `json:"fieldCount"`
	RequiredFields []string `json:"requiredFields,omitempty"`
	DefaultFields  []string `json:"defaultFields,omitempty"`
}

type InstallPreflightResources struct {
	UIMode           string   `json:"uiMode,omitempty"`
	FrontendEntry    string   `json:"frontendEntry,omitempty"`
	ServiceBaseURL   string   `json:"serviceBaseUrl,omitempty"`
	ServiceHealthURL string   `json:"serviceHealthUrl,omitempty"`
	Dependencies     []string `json:"dependencies,omitempty"`
}

type InstallPreflightData struct {
	Namespace          string                      `json:"namespace,omitempty"`
	MigrationVersion   string                      `json:"migrationVersion,omitempty"`
	MigrationDirectory string                      `json:"migrationDirectory,omitempty"`
	UninstallPolicy    string                      `json:"uninstallPolicy,omitempty"`
	RollbackPolicy     string                      `json:"rollbackPolicy,omitempty"`
	Tables             []InstallPreflightDataTable `json:"tables,omitempty"`
}

type InstallPreflightDataTable struct {
	Name        string                      `json:"name"`
	Description string                      `json:"description,omitempty"`
	PrimaryKey  string                      `json:"primaryKey,omitempty"`
	Columns     []string                    `json:"columns,omitempty"`
	Indexes     []InstallPreflightDataIndex `json:"indexes,omitempty"`
}

type InstallPreflightDataIndex struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns,omitempty"`
	Unique  bool     `json:"unique,omitempty"`
}

type InstallPreflightMigration struct {
	Version string                          `json:"version,omitempty"`
	Pending []InstallPreflightMigrationStep `json:"pending,omitempty"`
	Applied []InstallPreflightMigrationStep `json:"applied,omitempty"`
	Error   string                          `json:"error,omitempty"`
}

type InstallPreflightMigrationStep struct {
	Version  int    `json:"version"`
	Name     string `json:"name"`
	UpPath   string `json:"upPath,omitempty"`
	DownPath string `json:"downPath,omitempty"`
}

type InstallPreflightSignature struct {
	Status    string `json:"status"`
	Algorithm string `json:"algorithm,omitempty"`
	VendorID  string `json:"vendorId,omitempty"`
	SignedAt  string `json:"signedAt,omitempty"`
}

type InstallPreflightRisk struct {
	Level   string   `json:"level"`
	Summary []string `json:"summary,omitempty"`
}

func NewInstallPreflightService(loader MetadataLoader) *InstallPreflightService {
	if loader == nil {
		loader = NewFileLoader()
	}
	return &InstallPreflightService{loader: loader}
}

func (s *InstallPreflightService) Check(in InstallPreflightInput) (InstallPreflightResult, error) {
	source := strings.TrimSpace(in.Path)
	if source == "" {
		return InstallPreflightResult{}, fmt.Errorf("%w: plugin path is required", ErrPluginManifestBroken)
	}
	info, err := s.loader.Load(source)
	if err != nil {
		return InstallPreflightResult{}, err
	}

	result := InstallPreflightResult{
		Status: InstallPreflightStatusPass,
		Plugin: InstallPreflightPlugin{
			ID:          strings.TrimSpace(info.ID),
			Name:        strings.TrimSpace(info.Name),
			Version:     strings.TrimSpace(info.Version),
			Description: strings.TrimSpace(info.Description),
			Source:      source,
		},
		Permissions: buildInstallPreflightPermissionDiff(info, in.Installed),
		Menus:       buildInstallPreflightMenuDiff(info, in.Installed),
		Config:      buildInstallPreflightConfig(info),
		Resources:   buildInstallPreflightResources(info),
		Data:        buildInstallPreflightData(info),
		Migration:   buildInstallPreflightMigration(info, source),
		Signature:   buildInstallPreflightSignature(info),
	}

	for _, installed := range in.Installed {
		if samePluginReleaseTarget(installed, info) && installed.State != StateUninstalled {
			result.Blockers = append(result.Blockers, "plugin already installed: "+info.ID)
			break
		}
	}
	if err := info.ValidateManifest(); err != nil {
		result.Blockers = append(result.Blockers, "manifest invalid: "+err.Error())
	}
	if err := info.ValidateCompatibility(in.CoreVersion); err != nil {
		result.Blockers = append(result.Blockers, "compatibility failed: "+err.Error())
	}
	if len(result.Permissions.Conflict) > 0 {
		result.Blockers = append(result.Blockers, "permission conflicts detected")
	}
	if len(result.Menus.Conflict) > 0 {
		result.Blockers = append(result.Blockers, "menu conflicts detected")
	}
	if result.Migration.Error != "" {
		result.Blockers = append(result.Blockers, "migration plan invalid: "+result.Migration.Error)
	}
	if result.Signature.Status == "incomplete" || result.Signature.Status == "unsupported" {
		result.Blockers = append(result.Blockers, "signature invalid: "+result.Signature.Status)
	}
	if result.Signature.Status == "unsigned" {
		result.Warnings = append(result.Warnings, "plugin manifest is unsigned")
	}

	result.Risk = buildInstallPreflightRisk(result)
	if len(result.Blockers) > 0 {
		result.Status = InstallPreflightStatusBlocked
	}
	return result, nil
}

func buildInstallPreflightPermissionDiff(info Info, installed []Info) InstallPreflightResourceDiff {
	existing := map[string]string{}
	for _, item := range installed {
		for _, permission := range mustCatalogPermissions(item) {
			existing[permission.Key()] = permission.Source()
		}
	}
	diff := InstallPreflightResourceDiff{}
	for _, permission := range mustCatalogPermissions(info) {
		record := InstallPreflightPermission{
			Key:    permission.Key(),
			Type:   string(permission.Type()),
			Module: permission.Module(),
			Name:   permission.name,
			Risk:   permission.risk,
			Source: permission.Source(),
		}
		if source, ok := existing[permission.Key()]; ok {
			record.Existing = source
			if source == permission.Source() {
				diff.Update = append(diff.Update, record)
			} else {
				diff.Conflict = append(diff.Conflict, record)
			}
			continue
		}
		diff.Add = append(diff.Add, record)
	}
	sortInstallPreflightPermissions(diff.Add)
	sortInstallPreflightPermissions(diff.Update)
	sortInstallPreflightPermissions(diff.Conflict)
	return diff
}

func buildInstallPreflightMenuDiff(info Info, installed []Info) InstallPreflightMenuDiff {
	existing := map[string]string{}
	for _, item := range installed {
		for _, menu := range mustMenuNodes(item) {
			existing[menu.Key()] = menu.Source()
		}
	}
	diff := InstallPreflightMenuDiff{}
	for _, menu := range mustMenuNodes(info) {
		record := InstallPreflightMenu{
			Key:                 menu.Key(),
			ParentKey:           menu.ParentKey(),
			Path:                menu.Path(),
			Name:                menu.Name(),
			Source:              menu.Source(),
			RequiredRoles:       append([]string(nil), menu.requiredRoles...),
			RequiredPermissions: append([]string(nil), menu.requiredPermissions...),
		}
		if source, ok := existing[menu.Key()]; ok {
			record.Existing = source
			if source == menu.Source() {
				diff.Update = append(diff.Update, record)
			} else {
				diff.Conflict = append(diff.Conflict, record)
			}
			continue
		}
		diff.Add = append(diff.Add, record)
	}
	sortInstallPreflightMenus(diff.Add)
	sortInstallPreflightMenus(diff.Update)
	sortInstallPreflightMenus(diff.Conflict)
	return diff
}

func buildInstallPreflightConfig(info Info) InstallPreflightConfig {
	if info.ConfigSchema == nil {
		return InstallPreflightConfig{}
	}
	out := InstallPreflightConfig{HasSchema: true, FieldCount: len(info.ConfigSchema.Fields)}
	for _, field := range info.ConfigSchema.Fields {
		key := strings.TrimSpace(field.Key)
		if key == "" {
			continue
		}
		if field.Required {
			out.RequiredFields = append(out.RequiredFields, key)
		}
		if strings.TrimSpace(field.Default) != "" {
			out.DefaultFields = append(out.DefaultFields, key)
		}
	}
	sort.Strings(out.RequiredFields)
	sort.Strings(out.DefaultFields)
	return out
}

func buildInstallPreflightResources(info Info) InstallPreflightResources {
	out := InstallPreflightResources{
		UIMode:           string(info.UIMode),
		FrontendEntry:    ResolveFrontendEntry(info),
		ServiceBaseURL:   strings.TrimSpace(info.ServiceBaseURL),
		ServiceHealthURL: strings.TrimSpace(info.ServiceHealthURL),
	}
	for _, dep := range info.Dependencies {
		id := strings.TrimSpace(dep.ID)
		if id == "" {
			continue
		}
		if strings.TrimSpace(dep.Version) != "" {
			id += "@" + strings.TrimSpace(dep.Version)
		}
		out.Dependencies = append(out.Dependencies, id)
	}
	sort.Strings(out.Dependencies)
	return out
}

func buildInstallPreflightData(info Info) InstallPreflightData {
	if info.DataManifest == nil {
		return InstallPreflightData{}
	}
	data := info.DataManifest
	out := InstallPreflightData{
		Namespace:          strings.TrimSpace(data.Namespace),
		MigrationVersion:   strings.TrimSpace(data.MigrationVersion),
		MigrationDirectory: strings.TrimSpace(data.MigrationDirectory),
		UninstallPolicy:    string(data.UninstallPolicy),
		RollbackPolicy:     string(data.RollbackPolicy),
		Tables:             make([]InstallPreflightDataTable, 0, len(data.Tables)),
	}
	for _, table := range data.Tables {
		record := InstallPreflightDataTable{
			Name:        strings.TrimSpace(table.Name),
			Description: strings.TrimSpace(table.Description),
			PrimaryKey:  strings.TrimSpace(table.PrimaryKey),
			Columns:     append([]string(nil), table.Columns...),
			Indexes:     make([]InstallPreflightDataIndex, 0, len(table.Indexes)),
		}
		for _, index := range table.Indexes {
			record.Indexes = append(record.Indexes, InstallPreflightDataIndex{
				Name:    strings.TrimSpace(index.Name),
				Columns: append([]string(nil), index.Columns...),
				Unique:  index.Unique,
			})
		}
		out.Tables = append(out.Tables, record)
	}
	return out
}

func buildInstallPreflightMigration(info Info, source string) InstallPreflightMigration {
	out := InstallPreflightMigration{Version: strings.TrimSpace(info.MigrationVersion)}
	if out.Version == "" {
		return out
	}
	plan, err := NewMigrator(source).Plan()
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.Pending = migrationStepPreviews(plan.Pending)
	out.Applied = migrationStepPreviews(plan.Applied)
	return out
}

func buildInstallPreflightSignature(info Info) InstallPreflightSignature {
	if info.Signature == nil {
		return InstallPreflightSignature{Status: "unsigned"}
	}
	status := "signed"
	if info.Signature.Algorithm != SigAlgoRSASHA256 {
		status = "unsupported"
	} else if strings.TrimSpace(info.Signature.Value) == "" || info.Signature.Timestamp.IsZero() {
		status = "incomplete"
	}
	signedAt := ""
	if !info.Signature.Timestamp.IsZero() {
		signedAt = info.Signature.Timestamp.UTC().Format("2006-01-02T15:04:05Z")
	}
	return InstallPreflightSignature{
		Status:    status,
		Algorithm: string(info.Signature.Algorithm),
		VendorID:  strings.TrimSpace(info.Signature.VendorID),
		SignedAt:  signedAt,
	}
}

func buildInstallPreflightRisk(result InstallPreflightResult) InstallPreflightRisk {
	rank := 0
	summary := make([]string, 0)
	for _, permission := range append(append([]InstallPreflightPermission{}, result.Permissions.Add...), result.Permissions.Update...) {
		if r := riskRank(permission.Risk); r > rank {
			rank = r
		}
	}
	if len(result.Permissions.Conflict) > 0 || len(result.Menus.Conflict) > 0 || len(result.Blockers) > 0 {
		rank = 3
		summary = append(summary, "blocking conflicts or invalid install inputs")
	}
	if result.Signature.Status != "signed" {
		if rank < 1 {
			rank = 1
		}
		summary = append(summary, "signature:"+result.Signature.Status)
	}
	if len(result.Migration.Pending) > 0 || result.Migration.Error != "" {
		if rank < 1 {
			rank = 1
		}
		summary = append(summary, "migration impact")
	}
	if len(result.Data.Tables) > 0 {
		if rank < 2 {
			rank = 2
		}
		summary = append(summary, "data contract:"+result.Data.Namespace)
	}
	if result.Data.UninstallPolicy == string(DataUninstallDrop) {
		rank = 3
		summary = append(summary, "destructive uninstall policy")
	}
	if result.Resources.ServiceBaseURL != "" || result.Resources.ServiceHealthURL != "" {
		if rank < 2 {
			rank = 2
		}
		summary = append(summary, "network access")
	}
	if result.Resources.FrontendEntry != "" && result.Resources.UIMode != "" && result.Resources.UIMode != string(UIModeBackendOnly) {
		summary = append(summary, "frontend assets")
	}
	sort.Strings(summary)
	return InstallPreflightRisk{Level: riskLevelFromRank(rank), Summary: summary}
}

func samePluginReleaseTarget(left, right Info) bool {
	return strings.EqualFold(strings.TrimSpace(left.ID), strings.TrimSpace(right.ID))
}

func mustCatalogPermissions(info Info) []permissionPreviewSource {
	permissions, err := info.CatalogPermissions()
	if err != nil {
		return nil
	}
	out := make([]permissionPreviewSource, 0, len(permissions))
	for _, permission := range permissions {
		out = append(out, permissionPreviewSource{
			key:    permission.Key(),
			typ:    string(permission.Type()),
			module: permission.Module(),
			name:   permission.Name,
			risk:   string(permission.Risk),
			source: permission.Source(),
		})
	}
	return out
}

type permissionPreviewSource struct {
	key    string
	typ    string
	module string
	name   string
	risk   string
	source string
}

func (p permissionPreviewSource) Key() string    { return p.key }
func (p permissionPreviewSource) Type() string   { return p.typ }
func (p permissionPreviewSource) Module() string { return p.module }
func (p permissionPreviewSource) Source() string { return p.source }

func mustMenuNodes(info Info) []menuPreviewSource {
	menus, err := info.MenuNodes()
	if err != nil {
		return nil
	}
	out := make([]menuPreviewSource, 0, len(menus))
	for _, menu := range menus {
		out = append(out, menuPreviewSource{
			key:                 menu.Key(),
			parentKey:           menu.ParentKey(),
			path:                menu.Path(),
			name:                menu.Name(),
			source:              menu.Source(),
			requiredRoles:       append([]string(nil), menu.RequiredRoles...),
			requiredPermissions: append([]string(nil), menu.RequiredPermissions...),
		})
	}
	return out
}

type menuPreviewSource struct {
	key                 string
	parentKey           string
	path                string
	name                string
	source              string
	requiredRoles       []string
	requiredPermissions []string
}

func (m menuPreviewSource) Key() string       { return m.key }
func (m menuPreviewSource) ParentKey() string { return m.parentKey }
func (m menuPreviewSource) Path() string      { return m.path }
func (m menuPreviewSource) Name() string      { return m.name }
func (m menuPreviewSource) Source() string    { return m.source }

func migrationStepPreviews(steps []MigrationStep) []InstallPreflightMigrationStep {
	out := make([]InstallPreflightMigrationStep, 0, len(steps))
	for _, step := range steps {
		out = append(out, InstallPreflightMigrationStep{
			Version:  step.Version,
			Name:     step.Name,
			UpPath:   step.UpPath,
			DownPath: step.DownPath,
		})
	}
	return out
}

func sortInstallPreflightPermissions(items []InstallPreflightPermission) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Key < items[j].Key
	})
}

func sortInstallPreflightMenus(items []InstallPreflightMenu) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Key < items[j].Key
	})
}
