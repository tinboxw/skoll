package user

import (
	"errors"
	"unicode/utf8"
)

func ValidateUsername(username string) error {
	n := utf8.RuneCountInString(username)
	if n < 3 || n > 64 {
		return errors.New("username length must be between 3 and 64")
	}
	return nil
}
