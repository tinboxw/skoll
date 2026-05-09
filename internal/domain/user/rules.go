package user

import (
	"fmt"
	"regexp"
	"strings"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{2,31}$`)

func ValidateUsername(username string) error {
	v := strings.TrimSpace(username)
	if v == "" {
		return fmt.Errorf("username is required")
	}
	if !usernamePattern.MatchString(v) {
		return fmt.Errorf("username must match %s", usernamePattern.String())
	}
	return nil
}

func ValidateDisplayName(name string) error {
	v := strings.TrimSpace(name)
	if v == "" {
		return fmt.Errorf("display name is required")
	}
	if len([]rune(v)) > 64 {
		return fmt.Errorf("display name is too long")
	}
	return nil
}
