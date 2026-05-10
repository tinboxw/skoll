package user

import (
	"fmt"
	"regexp"
	"strings"
)

var accountPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{2,31}$`)

func ValidateAccount(account string) error {
	v := strings.TrimSpace(account)
	if v == "" {
		return fmt.Errorf("account is required")
	}
	if !accountPattern.MatchString(v) {
		return fmt.Errorf("account must match %s", accountPattern.String())
	}
	return nil
}

func ValidateName(name string) error {
	v := strings.TrimSpace(name)
	if v == "" {
		return fmt.Errorf("name is required")
	}
	if len([]rune(v)) > 64 {
		return fmt.Errorf("name is too long")
	}
	return nil
}
