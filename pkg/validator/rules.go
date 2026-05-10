package validator

import (
	"fmt"
	"regexp"
	"strings"
)

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// Rule validates one field value.
type Rule func(value any) error

func Required() Rule {
	return func(value any) error {
		if value == nil {
			return fmt.Errorf("is required")
		}
		s, ok := value.(string)
		if ok && strings.TrimSpace(s) == "" {
			return fmt.Errorf("is required")
		}
		return nil
	}
}

func MinLen(min int) Rule {
	return func(value any) error {
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be a string")
		}
		if len([]rune(s)) < min {
			return fmt.Errorf("length must be >= %d", min)
		}
		return nil
	}
}

func MaxLen(max int) Rule {
	return func(value any) error {
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be a string")
		}
		if len([]rune(s)) > max {
			return fmt.Errorf("length must be <= %d", max)
		}
		return nil
	}
}

func Email() Rule {
	return func(value any) error {
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be a string")
		}
		if !emailPattern.MatchString(strings.TrimSpace(s)) {
			return fmt.Errorf("must be a valid email")
		}
		return nil
	}
}
