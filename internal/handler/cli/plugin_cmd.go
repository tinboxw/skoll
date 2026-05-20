package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/tinboxw/skoll/internal/plugin"
)

type PluginCommand struct {
	manager plugin.Manager
	loader  plugin.MetadataLoader
	logRoot string
}

func NewPluginCommand(manager plugin.Manager, loader plugin.MetadataLoader, logRoot string) *PluginCommand {
	if loader == nil {
		loader = plugin.NewFileLoader()
	}
	if strings.TrimSpace(logRoot) == "" {
		logRoot = "log"
	}
	return &PluginCommand{
		manager: manager,
		loader:  loader,
		logRoot: logRoot,
	}
}

func (c *PluginCommand) Handle(_ context.Context, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("missing subcommand")
	}

	switch strings.ToLower(args[0]) {
	case "list":
		if c == nil || c.manager == nil {
			return "", fmt.Errorf("plugin manager not configured")
		}
		return c.handleList(), nil
	case "debug":
		if c == nil || c.manager == nil {
			return "", fmt.Errorf("plugin manager not configured")
		}
		if len(args) < 2 {
			return "", fmt.Errorf("usage: debug <pluginId>")
		}
		return c.handleDebug(args[1])
	case "logs":
		if len(args) < 2 {
			return "", fmt.Errorf("usage: logs <pluginId>")
		}
		return c.handleLogs(args[1])
	case "validate":
		if c == nil || c.loader == nil {
			return "", fmt.Errorf("plugin metadata loader not configured")
		}
		if len(args) < 2 {
			return "", fmt.Errorf("usage: validate <pluginPathOrManifest>")
		}
		return c.handleValidate(args[1])
	case "validate-all":
		if c == nil || c.loader == nil {
			return "", fmt.Errorf("plugin metadata loader not configured")
		}
		if len(args) < 2 {
			return "", fmt.Errorf("usage: validate-all <pluginsRootDir>")
		}
		return c.handleValidateAll(args[1])
	case "scaffold":
		if len(args) < 4 {
			return "", fmt.Errorf("usage: scaffold <pluginsRootDir> <pluginId> <pluginName> [appId]")
		}
		appID := ""
		if len(args) > 4 {
			appID = args[4]
		}
		return c.handleScaffold(args[1], args[2], args[3], appID)
	case "migrate":
		if len(args) < 3 {
			return "", fmt.Errorf("usage: migrate <pluginDir> <plan|apply|rollback> [steps]")
		}
		steps := 0
		if len(args) > 3 {
			v, err := strconv.Atoi(strings.TrimSpace(args[3]))
			if err != nil || v < 0 {
				return "", fmt.Errorf("invalid steps: %s", args[3])
			}
			steps = v
		}
		return c.handleMigrate(args[1], args[2], steps)
	default:
		return "", fmt.Errorf("unsupported plugin subcommand: %s", args[0])
	}
}

func (c *PluginCommand) handleList() string {
	items := c.manager.List()
	if len(items) == 0 {
		return "plugins=0"
	}

	rows := make([]string, 0, len(items)+1)
	rows = append(rows, fmt.Sprintf("plugins=%d", len(items)))
	for _, item := range items {
		rows = append(rows, fmt.Sprintf("- %s@%s state=%s", item.ID, item.Version, item.State))
	}

	return strings.Join(rows, "\n")
}

func (c *PluginCommand) handleDebug(pluginID string) (string, error) {
	info, err := c.manager.Get(pluginID)
	if err != nil {
		return "", err
	}

	deps := make([]string, 0, len(info.Dependencies))
	for _, dep := range info.Dependencies {
		version := dep.Version
		if version == "" {
			version = "*"
		}
		deps = append(deps, dep.ID+"@"+version)
	}
	sort.Strings(deps)

	perms := append([]string(nil), info.Permissions...)
	sort.Strings(perms)

	return strings.Join([]string{
		"id=" + info.ID,
		"name=" + info.Name,
		"version=" + info.Version,
		"state=" + string(info.State),
		"deps=" + strings.Join(deps, ","),
		"perms=" + strings.Join(perms, ","),
		"source=" + info.Source,
	}, "\n"), nil
}

func (c *PluginCommand) handleLogs(pluginID string) (string, error) {
	file := filepath.Join(c.logRoot, pluginID+".log")
	content, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

func (c *PluginCommand) handleValidate(path string) (string, error) {
	info, err := c.loader.Load(path)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("valid id=%s version=%s deps=%d perms=%d", info.ID, info.Version, len(info.Dependencies), len(info.Permissions)), nil
}

func (c *PluginCommand) handleValidateAll(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", fmt.Errorf("plugins root is required")
	}

	pluginPaths, err := discoverPluginManifestDirs(root)
	if err != nil {
		return "", err
	}
	if len(pluginPaths) == 0 {
		return "", fmt.Errorf("no plugin manifest found under %s", root)
	}

	rows := make([]string, 0, len(pluginPaths)+1)
	rows = append(rows, fmt.Sprintf("validated=%d", len(pluginPaths)))
	errs := make([]string, 0)

	for _, path := range pluginPaths {
		info, loadErr := c.loader.Load(path)
		if loadErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, loadErr))
			continue
		}
		rows = append(rows, fmt.Sprintf("- ok %s id=%s version=%s", path, info.ID, info.Version))
	}

	if len(errs) > 0 {
		return strings.Join(rows, "\n"), fmt.Errorf("manifest validation failed: %s", strings.Join(errs, "; "))
	}

	return strings.Join(rows, "\n"), nil
}

func discoverPluginManifestDirs(root string) ([]string, error) {
	paths := make([]string, 0)

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d == nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		name := d.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" {
			if path == root {
				return nil
			}
			return filepath.SkipDir
		}

		manifestPath := filepath.Join(path, "plugin.yaml")
		if info, err := os.Stat(manifestPath); err == nil && !info.IsDir() {
			paths = append(paths, filepath.Clean(path))
			return filepath.SkipDir
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(paths)
	return paths, nil
}

func (c *PluginCommand) handleScaffold(root, pluginID, pluginName, appID string) (string, error) {
	root = strings.TrimSpace(root)
	pluginID = strings.TrimSpace(strings.ToLower(pluginID))
	pluginName = strings.TrimSpace(pluginName)
	appID = strings.TrimSpace(strings.ToLower(appID))

	if root == "" || pluginID == "" || pluginName == "" {
		return "", fmt.Errorf("plugins root, plugin id, and plugin name are required")
	}

	info := plugin.Info{
		ID:                 pluginID,
		Name:               pluginName,
		Version:            "0.1.0",
		APIVersion:         "v1",
		CompatibilitySkoll: ">=1.0.0 <2.0.0",
		MigrationVersion:   "v0.1.0",
		UIMode:             plugin.UIModeSeparated,
		MountPolicy:        plugin.MountPolicyAdmin,
		Level:              plugin.LevelSystem,
		Permissions:        []string{pluginID + ".read"},
	}
	if appID != "" {
		info.Level = plugin.LevelApp
		info.AppID = appID
	}
	if err := info.ValidateManifest(); err != nil {
		return "", fmt.Errorf("invalid scaffold inputs: %w", err)
	}

	pluginDir := filepath.Join(root, pluginID)
	manifestPath := filepath.Join(pluginDir, "plugin.yaml")
	if _, err := os.Stat(manifestPath); err == nil {
		return "", fmt.Errorf("plugin manifest already exists: %s", manifestPath)
	}

	dirs := []string{
		pluginDir,
		filepath.Join(pluginDir, "backend"),
		filepath.Join(pluginDir, "frontend"),
		filepath.Join(pluginDir, "migrations"),
		filepath.Join(pluginDir, "docs"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}

	manifest := renderScaffoldManifest(info)
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		return "", err
	}

	readme := strings.Join([]string{
		"# " + pluginName,
		"",
		"Generated by: plugin scaffold",
		"",
		"## Next Steps",
		"",
		"1. Implement backend APIs in ./backend",
		"2. Build frontend app in ./frontend",
		"3. Add DB migrations in ./migrations",
		"4. Validate with: plugin validate " + pluginDir,
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(pluginDir, "README.md"), []byte(readme), 0o644); err != nil {
		return "", err
	}

	return fmt.Sprintf("scaffolded plugin at %s", pluginDir), nil
}

func renderScaffoldManifest(info plugin.Info) string {
	lines := []string{
		"id: " + info.ID,
		"name: " + quoteYAML(info.Name),
		"version: " + info.Version,
		"api_version: " + info.APIVersion,
		"compatibility_skoll: " + quoteYAML(info.CompatibilitySkoll),
		"migration_version: " + info.MigrationVersion,
		"ui_mode: " + string(info.UIMode),
		"level: " + string(info.Level),
		"mount_policy: " + string(info.MountPolicy),
	}
	if strings.TrimSpace(info.AppID) != "" {
		lines = append(lines, "app_id: "+info.AppID)
	}
	lines = append(lines,
		"permissions:",
		"  - "+quoteYAML(info.Permissions[0]),
	)
	return strings.Join(lines, "\n") + "\n"
}

func quoteYAML(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.ReplaceAll(trimmed, `"`, `\"`)
	return `"` + trimmed + `"`
}

func (c *PluginCommand) handleMigrate(pluginDir, action string, steps int) (string, error) {
	migrator := plugin.NewMigrator(pluginDir)
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "plan":
		plan, err := migrator.Plan()
		if err != nil {
			return "", err
		}
		rows := []string{fmt.Sprintf("applied=%d pending=%d", len(plan.Applied), len(plan.Pending))}
		for _, step := range plan.Pending {
			rows = append(rows, fmt.Sprintf("- pending %03d", step.Version))
		}
		return strings.Join(rows, "\n"), nil
	case "apply":
		applied, err := migrator.Apply(steps)
		if err != nil {
			return "", err
		}
		if len(applied) == 0 {
			return "applied=0", nil
		}
		rows := []string{fmt.Sprintf("applied=%d", len(applied))}
		for _, step := range applied {
			rows = append(rows, fmt.Sprintf("- applied %03d", step.Version))
		}
		return strings.Join(rows, "\n"), nil
	case "rollback":
		rolled, err := migrator.Rollback(steps)
		if err != nil {
			return "", err
		}
		if len(rolled) == 0 {
			return "rolled_back=0", nil
		}
		rows := []string{fmt.Sprintf("rolled_back=%d", len(rolled))}
		for _, step := range rolled {
			rows = append(rows, fmt.Sprintf("- rolled_back %03d", step.Version))
		}
		return strings.Join(rows, "\n"), nil
	default:
		return "", fmt.Errorf("unsupported migrate action: %s", action)
	}
}
