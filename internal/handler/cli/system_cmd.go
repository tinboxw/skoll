package cli

import (
	"context"
	"fmt"
	"runtime"
	"strings"
)

type SystemCommand struct{}

func NewSystemCommand() *SystemCommand {
	return &SystemCommand{}
}

func (c *SystemCommand) Handle(_ context.Context, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("missing subcommand")
	}
	switch strings.ToLower(args[0]) {
	case "health":
		return "ok", nil
	case "version":
		return "go=" + runtime.Version(), nil
	default:
		return "", fmt.Errorf("unsupported system subcommand: %s", args[0])
	}
}
