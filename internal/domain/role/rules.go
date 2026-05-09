package role

import (
	"errors"
	"unicode/utf8"
)

func ValidateName(name string) error {
	n := utf8.RuneCountInString(name)
	if n < 2 || n > 64 {
		return errors.New("role name length must be between 2 and 64")
	}
	return nil
}
