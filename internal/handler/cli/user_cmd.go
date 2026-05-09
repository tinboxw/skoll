package cli

import (
	"context"
	"fmt"
	"strings"

	usersvc "github.com/tinboxw/skoll/internal/service/user"
)

type UserCommand struct {
	service usersvc.Service
}

func NewUserCommand(service usersvc.Service) *UserCommand {
	return &UserCommand{service: service}
}

func (c *UserCommand) Handle(ctx context.Context, args []string) (string, error) {
	if c == nil || c.service == nil {
		return "", fmt.Errorf("user service not configured")
	}
	if len(args) == 0 {
		return "", fmt.Errorf("missing subcommand")
	}

	switch strings.ToLower(args[0]) {
	case "list":
		items, err := c.service.List(ctx, usersvc.ListInput{Offset: 0, Limit: 50})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("users=%d", len(items)), nil
	case "disable":
		if len(args) < 3 {
			return "", fmt.Errorf("usage: disable <userId> <actorId>")
		}
		if err := c.service.Disable(ctx, args[1], args[2]); err != nil {
			return "", err
		}
		return "disabled", nil
	default:
		return "", fmt.Errorf("unsupported user subcommand: %s", args[0])
	}
}
