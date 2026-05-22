package user

import (
	"fmt"
	"strings"

	domainuser "github.com/tinboxw/skoll/internal/domain/user"
)

func validateCreateInput(in CreateUserInput) error {
	if err := domainuser.ValidateAccount(in.Account); err != nil {
		return err
	}
	if err := domainuser.ValidateName(in.Name); err != nil {
		return err
	}
	if _, err := domainuser.NewEmail(in.Email); err != nil {
		return err
	}
	password := strings.TrimSpace(in.PasswordHash)
	if _, err := domainuser.NewPasswordHash(password); err != nil {
		if _, hashErr := domainuser.HashPassword(password); hashErr != nil {
			return hashErr
		}
	}
	if strings.TrimSpace(in.ActorID) == "" {
		return fmt.Errorf("actor id is required")
	}
	return nil
}

func validateListInput(in ListInput) error {
	if in.Offset < 0 {
		return fmt.Errorf("offset must be >= 0")
	}
	if in.Limit < 0 {
		return fmt.Errorf("limit must be >= 0")
	}
	return nil
}
