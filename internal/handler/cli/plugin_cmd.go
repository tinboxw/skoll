package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
		logRoot = filepath.Join("plugins", "logs")
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
