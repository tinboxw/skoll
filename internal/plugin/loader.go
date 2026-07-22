package plugin

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type MetadataLoader interface {
	Load(path string) (Info, error)
}

type FileLoader struct{}

const (
	sectionRoot               = "root"
	sectionDeps               = "deps"
	sectionPerm               = "perm"
	sectionI18n               = "i18n"
	sectionUIMenu             = "ui_menu"
	sectionUIMenuRoles        = "ui_menu_roles"
	sectionUIMenuPermissions  = "ui_menu_permissions"
	sectionConfigSchema       = "config_schema"
	sectionConfigFields       = "config_schema_fields"
	sectionConfigOptions      = "config_schema_options"
	sectionData               = "data"
	sectionDataTables         = "data_tables"
	sectionAPI                = "api"
	sectionAPIRoutes          = "api_routes"
	sectionEvents             = "events"
	sectionEventSubscriptions = "event_subscriptions"
)

func NewFileLoader() *FileLoader {
	return &FileLoader{}
}

func (l *FileLoader) Load(path string) (Info, error) {
	manifestPath, err := resolveManifestPath(path)
	if err != nil {
		return Info{}, err
	}

	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return Info{}, fmt.Errorf("read plugin manifest: %w", err)
	}

	info, err := parseManifest(raw)
	if err != nil {
		return Info{}, err
	}

	if err := info.ValidateManifest(); err != nil {
		return Info{}, err
	}

	return info, nil
}

func resolveManifestPath(path string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if fi.IsDir() {
		p := filepath.Join(path, "plugin.yaml")
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("plugin.yaml not found under %s", path)
		}
		return p, nil
	}

	if filepath.Base(path) != "plugin.yaml" {
		return "", errors.New("manifest file must be named plugin.yaml")
	}

	return path, nil
}

func parseManifest(raw []byte) (Info, error) {
	var info Info

	section := sectionRoot
	var currentDep *Dependency
	var currentPermission *PermissionDeclaration
	var currentConfigField *ConfigField
	var currentConfigOption *ConfigOption
	var currentDataTable *DataTable
	var currentAPIRoute *APIRoute
	var currentEventSubscription *EventSubscription

	flushPermission := func() {
		if currentPermission == nil {
			return
		}
		info.PermissionResources = append(info.PermissionResources, *currentPermission)
		currentPermission = nil
	}
	flushDependency := func() {
		if currentDep == nil {
			return
		}
		info.Dependencies = append(info.Dependencies, *currentDep)
		currentDep = nil
	}
	flushConfigField := func() {
		if currentConfigOption != nil && currentConfigField != nil {
			currentConfigField.Options = append(currentConfigField.Options, *currentConfigOption)
			currentConfigOption = nil
		}
		if currentConfigField != nil {
			if info.ConfigSchema == nil {
				info.ConfigSchema = &ConfigSchema{}
			}
			info.ConfigSchema.Fields = append(info.ConfigSchema.Fields, *currentConfigField)
			currentConfigField = nil
		}
	}
	flushDataTable := func() {
		if currentDataTable == nil {
			return
		}
		if info.DataManifest == nil {
			info.DataManifest = &DataManifest{}
		}
		info.DataManifest.Tables = append(info.DataManifest.Tables, *currentDataTable)
		currentDataTable = nil
	}
	flushAPIRoute := func() {
		if currentAPIRoute == nil {
			return
		}
		if info.APIContract == nil {
			info.APIContract = &APIContract{}
		}
		info.APIContract.Routes = append(info.APIContract.Routes, *currentAPIRoute)
		currentAPIRoute = nil
	}
	flushEventSubscription := func() {
		if currentEventSubscription == nil {
			return
		}
		if info.EventContract == nil {
			info.EventContract = &EventContract{}
		}
		info.EventContract.Subscriptions = append(info.EventContract.Subscriptions, *currentEventSubscription)
		currentEventSubscription = nil
	}

	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	for scanner.Scan() {
		rawLine := scanner.Text()
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		isTopLevel := rawLine == line

		switch line {
		case "dependencies:":
			flushDependency()
			flushPermission()
			section = sectionDeps
			continue
		case "permissions:":
			flushDependency()
			flushPermission()
			section = sectionPerm
			continue
		case "i18n_locales:":
			flushDependency()
			flushPermission()
			section = sectionI18n
			continue
		case "ui_menu:":
			flushDependency()
			flushPermission()
			if currentConfigOption != nil {
				currentConfigField.Options = append(currentConfigField.Options, *currentConfigOption)
				currentConfigOption = nil
			}
			if currentConfigField != nil {
				info.ConfigSchema.Fields = append(info.ConfigSchema.Fields, *currentConfigField)
				currentConfigField = nil
			}
			if info.UIMenu == nil {
				info.UIMenu = &UIMenu{}
			}
			section = sectionUIMenu
			continue
		case "config_schema:":
			flushDependency()
			flushPermission()
			flushDataTable()
			flushConfigField()
			if info.ConfigSchema == nil {
				info.ConfigSchema = &ConfigSchema{}
			}
			section = sectionConfigSchema
			continue
		case "data:":
			flushDependency()
			flushPermission()
			flushConfigField()
			flushDataTable()
			flushAPIRoute()
			if info.DataManifest == nil {
				info.DataManifest = &DataManifest{}
			}
			section = sectionData
			continue
		case "api:":
			flushDependency()
			flushPermission()
			flushConfigField()
			flushDataTable()
			flushAPIRoute()
			flushEventSubscription()
			if info.APIContract == nil {
				info.APIContract = &APIContract{}
			}
			section = sectionAPI
			continue
		case "events:":
			flushDependency()
			flushPermission()
			flushConfigField()
			flushDataTable()
			flushAPIRoute()
			flushEventSubscription()
			if info.EventContract == nil {
				info.EventContract = &EventContract{}
			}
			section = sectionEvents
			continue
		}

		if strings.HasPrefix(line, "-") {
			item := strings.TrimSpace(strings.TrimPrefix(line, "-"))
			switch section {
			case sectionDeps:
				if currentDep != nil {
					info.Dependencies = append(info.Dependencies, *currentDep)
				}
				currentDep = &Dependency{}
				if strings.HasPrefix(item, "id:") {
					currentDep.ID = parseScalar(strings.TrimSpace(strings.TrimPrefix(item, "id:")))
				} else {
					currentDep.ID = parseScalar(item)
				}
			case sectionPerm:
				flushPermission()
				if strings.HasPrefix(item, "key:") {
					currentPermission = &PermissionDeclaration{Key: parseScalar(strings.TrimSpace(strings.TrimPrefix(item, "key:")))}
				} else {
					appendPermissionDeclaration(&info, PermissionDeclaration{Key: parseScalar(item)})
				}
			case sectionI18n:
				info.I18nLocales = append(info.I18nLocales, parseScalar(item))
			case sectionUIMenuRoles:
				if info.UIMenu == nil {
					info.UIMenu = &UIMenu{}
				}
				info.UIMenu.RequiredRoles = append(info.UIMenu.RequiredRoles, parseScalar(item))
			case sectionUIMenuPermissions:
				if info.UIMenu == nil {
					info.UIMenu = &UIMenu{}
				}
				info.UIMenu.RequiredPermissions = append(info.UIMenu.RequiredPermissions, parseScalar(item))
			case sectionConfigFields:
				if info.ConfigSchema == nil {
					info.ConfigSchema = &ConfigSchema{}
				}
				if currentConfigOption != nil {
					currentConfigField.Options = append(currentConfigField.Options, *currentConfigOption)
					currentConfigOption = nil
				}
				if currentConfigField != nil {
					info.ConfigSchema.Fields = append(info.ConfigSchema.Fields, *currentConfigField)
				}
				currentConfigField = &ConfigField{}
				if strings.HasPrefix(item, "key:") {
					currentConfigField.Key = parseScalar(strings.TrimSpace(strings.TrimPrefix(item, "key:")))
				} else {
					currentConfigField.Key = parseScalar(item)
				}
			case sectionConfigOptions:
				if currentConfigField == nil {
					currentConfigField = &ConfigField{}
				}
				if strings.HasPrefix(item, "key:") {
					if currentConfigOption != nil {
						currentConfigField.Options = append(currentConfigField.Options, *currentConfigOption)
						currentConfigOption = nil
					}
					info.ConfigSchema.Fields = append(info.ConfigSchema.Fields, *currentConfigField)
					currentConfigField = &ConfigField{Key: parseScalar(strings.TrimSpace(strings.TrimPrefix(item, "key:")))}
					section = sectionConfigFields
					continue
				}
				if currentConfigOption != nil {
					currentConfigField.Options = append(currentConfigField.Options, *currentConfigOption)
				}
				currentConfigOption = &ConfigOption{}
				if strings.HasPrefix(item, "value:") {
					currentConfigOption.Value = parseScalar(strings.TrimSpace(strings.TrimPrefix(item, "value:")))
				} else {
					currentConfigOption.Value = parseScalar(item)
				}
			case sectionDataTables:
				flushDataTable()
				currentDataTable = &DataTable{}
				if strings.HasPrefix(item, "name:") {
					currentDataTable.Name = parseScalar(strings.TrimSpace(strings.TrimPrefix(item, "name:")))
				} else {
					currentDataTable.Name = parseScalar(item)
				}
			case sectionAPIRoutes:
				flushAPIRoute()
				currentAPIRoute = &APIRoute{}
				if strings.HasPrefix(item, "method:") {
					currentAPIRoute.Method = parseScalar(strings.TrimSpace(strings.TrimPrefix(item, "method:")))
				} else {
					currentAPIRoute.Method = parseScalar(item)
				}
			case sectionEventSubscriptions:
				flushEventSubscription()
				currentEventSubscription = &EventSubscription{}
				if strings.HasPrefix(item, "name:") {
					currentEventSubscription.Name = parseScalar(strings.TrimSpace(strings.TrimPrefix(item, "name:")))
				} else {
					currentEventSubscription.Name = parseScalar(item)
				}
			}
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = parseScalar(strings.TrimSpace(value))
		if section == sectionPerm {
			if !isTopLevel && applyPermissionField(currentPermission, key, value) {
				continue
			}
			if isTopLevel {
				flushPermission()
				section = sectionRoot
			}
		}
		if section == sectionUIMenu || section == sectionUIMenuRoles || section == sectionUIMenuPermissions {
			if !isTopLevel && applyUIMenuField(info.UIMenu, key, value, &section) {
				continue
			}
			if isTopLevel {
				section = sectionRoot
			}
		}
		if section == sectionConfigSchema || section == sectionConfigFields || section == sectionConfigOptions {
			if !isTopLevel && applyConfigSchemaField(info.ConfigSchema, currentConfigField, currentConfigOption, key, value, &section) {
				continue
			}
			if isTopLevel {
				flushConfigField()
				section = sectionRoot
			}
		}
		if section == sectionData || section == sectionDataTables {
			if !isTopLevel && applyDataManifestField(info.DataManifest, currentDataTable, key, value, &section) {
				continue
			}
			if isTopLevel {
				flushDataTable()
				section = sectionRoot
			}
		}
		if section == sectionAPI || section == sectionAPIRoutes {
			if !isTopLevel && applyAPIContractField(info.APIContract, currentAPIRoute, key, value, &section) {
				continue
			}
			if isTopLevel {
				flushAPIRoute()
				section = sectionRoot
			}
		}
		if section == sectionEvents || section == sectionEventSubscriptions {
			if !isTopLevel && applyEventContractField(info.EventContract, currentEventSubscription, key, value, &section) {
				continue
			}
			if isTopLevel {
				flushEventSubscription()
				section = sectionRoot
			}
		}

		switch key {
		case "id":
			if section == sectionDeps {
				if currentDep == nil {
					currentDep = &Dependency{}
				}
				currentDep.ID = value
			} else {
				info.ID = value
			}
		case "name":
			info.Name = value
		case "name_zh_cn":
			info.NameZhCN = value
		case "name_en_us":
			info.NameEnUS = value
		case "version":
			if section == sectionDeps {
				if currentDep == nil {
					currentDep = &Dependency{}
				}
				currentDep.Version = value
			} else {
				info.Version = value
			}
		case "api_version":
			info.APIVersion = value
		case "service_base_url":
			info.ServiceBaseURL = value
		case "service_health_url":
			info.ServiceHealthURL = value
		case "migration_version":
			info.MigrationVersion = value
		case "description":
			info.Description = value
		case "ui_mode":
			info.UIMode = UIMode(value)
		case "level":
			info.Level = Level(value)
		case "app_id":
			info.AppID = value
		case "mount_policy":
			info.MountPolicy = MountPolicy(value)
		case "frontend_entry":
			info.FrontendEntry = value
		case "ui_nav_position":
			info.UINavPosition = UINavPosition(value)
		case "ui_open_mode":
			info.UIOpenMode = UIOpenMode(value)
		case "ui_tab_mode":
			info.UITabMode = UITabMode(value)
		case "i18n_locales":
			if value != "" {
				for _, locale := range strings.Split(value, ",") {
					trimmed := strings.TrimSpace(locale)
					if trimmed != "" {
						info.I18nLocales = append(info.I18nLocales, trimmed)
					}
				}
			}
		case "vendor":
			info.Vendor = value
		case "vendor_url":
			info.VendorURL = value
		case "vendor_id":
			if info.Signature == nil {
				info.Signature = &Signature{}
			}
			info.Signature.VendorID = value
		case "sign_algo":
			if info.Signature == nil {
				info.Signature = &Signature{}
			}
			info.Signature.Algorithm = SignatureAlgorithm(value)
		case "sign_timestamp":
			if info.Signature == nil {
				info.Signature = &Signature{}
			}
			if t, err := time.Parse(time.RFC3339, value); err == nil {
				info.Signature.Timestamp = t
			}
		case "sign_value":
			if info.Signature == nil {
				info.Signature = &Signature{}
			}
			info.Signature.Value = value
		case "vendor_pubkey":
			if info.Signature == nil {
				info.Signature = &Signature{}
			}
			info.Signature.PublicKey = value
			if info.Signature.VendorID == "" {
				info.Signature.VendorID = info.Vendor
			}
		default:
			if isTopLevel {
				return Info{}, fmt.Errorf("unsupported plugin manifest field: %s", key)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return Info{}, fmt.Errorf("scan manifest: %w", err)
	}

	flushDependency()
	flushPermission()
	flushConfigField()
	flushDataTable()
	flushAPIRoute()
	flushEventSubscription()
	if info.UIMode == "" {
		info.UIMode = UIModeBackendOnly
	}
	if info.Level == "" {
		info.Level = LevelSystem
	}
	if info.MountPolicy == "" {
		info.MountPolicy = MountPolicyAdmin
	}
	if info.UINavPosition == "" {
		info.UINavPosition = UINavPositionNone
	}
	if info.UIOpenMode == "" {
		info.UIOpenMode = UIOpenModeIntegrated
	}
	if info.UITabMode == "" && info.UIOpenMode == UIOpenModeStandalone {
		info.UITabMode = UITabModeDisabled
	}
	if info.UITabMode == "" {
		info.UITabMode = UITabModeOptional
	}
	if info.UIMode != UIModeBackendOnly && len(info.I18nLocales) == 0 {
		info.I18nLocales = []string{"zh-CN", "en-US"}
	}
	normalizePermissionDeclarations(&info)
	normalizeDataManifest(&info)
	normalizeAPIContract(&info)
	normalizeEventContract(&info)
	info.FrontendEntry = ResolveFrontendEntry(info)

	return info, nil
}

func appendPermissionDeclaration(info *Info, declaration PermissionDeclaration) {
	if info == nil {
		return
	}
	declaration = normalizePermissionDeclaration(declaration)
	info.PermissionResources = append(info.PermissionResources, declaration)
	info.Permissions = append(info.Permissions, declaration.Key)
}

func applyPermissionField(declaration *PermissionDeclaration, key, value string) bool {
	if declaration == nil {
		return false
	}
	switch key {
	case "key":
		declaration.Key = value
	case "type":
		declaration.Type = value
	case "module":
		declaration.Module = value
	case "name":
		declaration.Name = value
	case "risk":
		declaration.Risk = value
	default:
		if strings.HasPrefix(key, "metadata.") {
			if declaration.Metadata == nil {
				declaration.Metadata = map[string]string{}
			}
			declaration.Metadata[strings.TrimPrefix(key, "metadata.")] = value
			return true
		}
		return false
	}
	return true
}

func normalizePermissionDeclarations(info *Info) {
	if info == nil {
		return
	}
	out := make([]PermissionDeclaration, 0, len(info.PermissionResources))
	keys := make([]string, 0, len(info.PermissionResources))
	for _, declaration := range info.PermissionResources {
		normalized := normalizePermissionDeclaration(declaration)
		if normalized.Key == "" {
			out = append(out, normalized)
			continue
		}
		out = append(out, normalized)
		keys = append(keys, normalized.Key)
	}
	info.PermissionResources = out
	info.Permissions = keys
}

func normalizePermissionDeclaration(declaration PermissionDeclaration) PermissionDeclaration {
	declaration.Key = strings.TrimSpace(strings.ToLower(declaration.Key))
	declaration.Type = strings.TrimSpace(strings.ToLower(declaration.Type))
	if declaration.Type == "" {
		declaration.Type = "api"
	}
	declaration.Module = strings.TrimSpace(strings.ToLower(declaration.Module))
	if declaration.Module == "" {
		declaration.Module = inferPermissionModule(declaration.Key)
	}
	declaration.Name = strings.TrimSpace(declaration.Name)
	if declaration.Name == "" {
		declaration.Name = declaration.Key
	}
	declaration.Risk = strings.TrimSpace(strings.ToLower(declaration.Risk))
	if declaration.Risk == "" {
		declaration.Risk = "low"
	}
	if len(declaration.Metadata) > 0 {
		metadata := make(map[string]string, len(declaration.Metadata))
		for key, value := range declaration.Metadata {
			metadata[strings.TrimSpace(strings.ToLower(key))] = strings.TrimSpace(value)
		}
		declaration.Metadata = metadata
	}
	return declaration
}

func inferPermissionModule(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	for _, sep := range []string{".", ":", "_", "-"} {
		if idx := strings.Index(key, sep); idx > 0 {
			return key[:idx]
		}
	}
	return "plugin"
}

func applyConfigSchemaField(schema *ConfigSchema, field *ConfigField, option *ConfigOption, key, value string, section *string) bool {
	if schema == nil {
		return false
	}
	switch *section {
	case sectionConfigSchema:
		switch key {
		case "title":
			schema.Title = value
		case "title_zh_cn":
			schema.TitleZhCN = value
		case "title_en_us":
			schema.TitleEnUS = value
		case "description":
			schema.Description = value
		case "fields":
			*section = sectionConfigFields
		default:
			return false
		}
	case sectionConfigFields:
		if field == nil {
			return false
		}
		switch key {
		case "key":
			field.Key = value
		case "label":
			field.Label = value
		case "label_zh_cn":
			field.LabelZhCN = value
		case "label_en_us":
			field.LabelEnUS = value
		case "type":
			field.Type = strings.ToLower(strings.TrimSpace(value))
		case "required":
			field.Required = parseBoolScalar(value)
		case "default":
			field.Default = value
		case "placeholder":
			field.Placeholder = value
		case "help":
			field.Help = value
		case "min":
			if parsed, err := strconv.ParseFloat(value, 64); err == nil {
				field.Min = &parsed
			}
		case "max":
			if parsed, err := strconv.ParseFloat(value, 64); err == nil {
				field.Max = &parsed
			}
		case "min_length":
			if parsed, err := strconv.Atoi(value); err == nil {
				field.MinLength = &parsed
			}
		case "max_length":
			if parsed, err := strconv.Atoi(value); err == nil {
				field.MaxLength = &parsed
			}
		case "pattern":
			field.Pattern = value
		case "options":
			if value != "" {
				for _, part := range splitScalarList(value) {
					field.Options = append(field.Options, ConfigOption{Label: part, Value: part})
				}
			} else {
				*section = sectionConfigOptions
			}
		default:
			return false
		}
	case sectionConfigOptions:
		if option == nil {
			return false
		}
		switch key {
		case "value":
			option.Value = value
		case "label":
			option.Label = value
		case "label_zh_cn":
			option.LabelZhCN = value
		case "label_en_us":
			option.LabelEnUS = value
		default:
			return false
		}
	default:
		return false
	}
	return true
}

func applyDataManifestField(data *DataManifest, table *DataTable, key, value string, section *string) bool {
	if data == nil {
		return false
	}
	switch *section {
	case sectionData:
		switch key {
		case "namespace":
			data.Namespace = value
		case "migration_version":
			data.MigrationVersion = value
		case "migration_directory":
			data.MigrationDirectory = value
		case "uninstall_policy":
			data.UninstallPolicy = DataUninstallPolicy(strings.ToLower(strings.TrimSpace(value)))
		case "rollback_policy":
			data.RollbackPolicy = DataRollbackPolicy(strings.ToLower(strings.TrimSpace(value)))
		case "tables":
			*section = sectionDataTables
		default:
			return false
		}
	case sectionDataTables:
		if table == nil {
			return false
		}
		switch key {
		case "name":
			table.Name = value
		case "description":
			table.Description = value
		case "primary_key":
			table.PrimaryKey = value
		case "columns":
			table.Columns = splitScalarList(value)
		case "indexes":
			table.Indexes = parseDataIndexes(value)
		default:
			return false
		}
	default:
		return false
	}
	return true
}

func applyAPIContractField(contract *APIContract, route *APIRoute, key, value string, section *string) bool {
	if contract == nil {
		return false
	}
	switch *section {
	case sectionAPI:
		switch key {
		case "routes":
			*section = sectionAPIRoutes
		default:
			return false
		}
	case sectionAPIRoutes:
		if route == nil {
			return false
		}
		switch key {
		case "method":
			route.Method = value
		case "path":
			route.Path = NormalizeEntryPath(value)
		case "summary":
			route.Summary = value
		case "permission":
			route.Permission = value
		case "audit_action":
			route.AuditAction = value
		default:
			return false
		}
	default:
		return false
	}
	return true
}

func applyEventContractField(contract *EventContract, subscription *EventSubscription, key, value string, section *string) bool {
	if contract == nil {
		return false
	}
	switch *section {
	case sectionEvents:
		switch key {
		case "subscriptions":
			*section = sectionEventSubscriptions
		default:
			return false
		}
	case sectionEventSubscriptions:
		if subscription == nil {
			return false
		}
		switch key {
		case "name":
			subscription.Name = value
		case "handler":
			subscription.Handler = value
		case "retry_policy":
			subscription.RetryPolicy = value
		default:
			return false
		}
	default:
		return false
	}
	return true
}

func normalizeAPIContract(info *Info) {
	if info == nil || info.APIContract == nil {
		return
	}
	for i := range info.APIContract.Routes {
		route := &info.APIContract.Routes[i]
		route.Method = strings.ToUpper(strings.TrimSpace(route.Method))
		route.Path = NormalizeEntryPath(route.Path)
		route.Summary = strings.TrimSpace(route.Summary)
		route.Permission = strings.TrimSpace(strings.ToLower(route.Permission))
		route.AuditAction = strings.TrimSpace(strings.ToLower(route.AuditAction))
	}
}

func normalizeEventContract(info *Info) {
	if info == nil || info.EventContract == nil {
		return
	}
	for i := range info.EventContract.Subscriptions {
		subscription := &info.EventContract.Subscriptions[i]
		subscription.Name = strings.TrimSpace(strings.ToLower(subscription.Name))
		subscription.Handler = strings.TrimSpace(subscription.Handler)
		subscription.RetryPolicy = strings.TrimSpace(strings.ToLower(subscription.RetryPolicy))
		if subscription.RetryPolicy == "" {
			subscription.RetryPolicy = "standard"
		}
	}
}

func normalizeDataManifest(info *Info) {
	if info == nil || info.DataManifest == nil {
		return
	}
	data := info.DataManifest
	if strings.TrimSpace(data.Namespace) == "" {
		data.Namespace = normalizeDataNamespace(info.ID)
	} else {
		data.Namespace = normalizeDataNamespace(data.Namespace)
	}
	if strings.TrimSpace(data.MigrationVersion) == "" {
		data.MigrationVersion = strings.TrimSpace(info.MigrationVersion)
	}
	data.MigrationDirectory = strings.Trim(strings.TrimSpace(data.MigrationDirectory), "/\\")
	if data.MigrationDirectory == "" {
		data.MigrationDirectory = "migrations"
	}
	for i := range data.Tables {
		table := &data.Tables[i]
		table.Name = normalizeDataIdentifier(table.Name)
		table.PrimaryKey = normalizeDataIdentifier(table.PrimaryKey)
		table.Columns = normalizeDataIdentifierList(table.Columns)
		for j := range table.Indexes {
			index := &table.Indexes[j]
			index.Name = normalizeDataIdentifier(index.Name)
			index.Columns = normalizeDataIdentifierList(index.Columns)
		}
	}
}

func normalizeDataNamespace(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, ".", "_")
	return normalized
}

func normalizeDataIdentifier(value string) string {
	return normalizeDataNamespace(value)
}

func normalizeDataIdentifierList(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		normalized := normalizeDataIdentifier(item)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func parseDataIndexes(value string) []DataIndex {
	parts := strings.Split(value, ";")
	out := make([]DataIndex, 0, len(parts))
	for _, part := range parts {
		spec := strings.TrimSpace(part)
		if spec == "" {
			continue
		}
		index := DataIndex{}
		if strings.HasPrefix(strings.ToLower(spec), "unique ") {
			index.Unique = true
			spec = strings.TrimSpace(spec[len("unique "):])
		}
		name, columnsRaw, hasColumns := strings.Cut(spec, "(")
		index.Name = parseScalar(strings.TrimSpace(name))
		if hasColumns {
			columnsRaw = strings.TrimSuffix(columnsRaw, ")")
			index.Columns = splitScalarList(columnsRaw)
		}
		out = append(out, index)
	}
	return out
}

func parseBoolScalar(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func applyUIMenuField(menu *UIMenu, key, value string, section *string) bool {
	if menu == nil {
		return false
	}
	switch key {
	case "key":
		menu.Key = value
	case "parent_key":
		menu.ParentKey = value
	case "label":
		menu.Label = value
	case "label_zh_cn":
		menu.LabelZhCN = value
	case "label_en_us":
		menu.LabelEnUS = value
	case "path":
		menu.Path = NormalizeEntryPath(value)
	case "component":
		menu.Component = value
	case "icon":
		menu.Icon = value
	case "order":
		if order, err := strconv.Atoi(value); err == nil {
			menu.Order = order
		}
	case "visible":
		visible := parseBoolScalar(value)
		menu.Visible = &visible
	case "required_roles":
		if value != "" {
			menu.RequiredRoles = append(menu.RequiredRoles, splitScalarList(value)...)
		} else {
			*section = sectionUIMenuRoles
		}
	case "required_permissions":
		if value != "" {
			menu.RequiredPermissions = append(menu.RequiredPermissions, splitScalarList(value)...)
		} else {
			*section = sectionUIMenuPermissions
		}
	default:
		return false
	}
	return true
}

func splitScalarList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := parseScalar(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseScalar(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Trim(v, "\"")
	v = strings.Trim(v, "'")
	return v
}
