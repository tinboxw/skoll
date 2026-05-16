package gormrepo

import (
	"strconv"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainuser "github.com/tinboxw/skoll/internal/domain/user"
)

type UserModel struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	Account   string `gorm:"column:account;size:128;uniqueIndex"`
	Name      string `gorm:"column:name;size:128"`
	Email     string `gorm:"size:191;uniqueIndex"`
	Status    string `gorm:"size:32"`
	Password  string `gorm:"size:256"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (UserModel) TableName() string { return "sk_users" }

func UserModelFromDomain(entity *domainuser.User) UserModel {
	return UserModel{
		ID:        parseUserID(entity.ID.String()),
		Account:   entity.Account,
		Name:      entity.Name,
		Email:     entity.Email.String(),
		Status:    string(entity.Status),
		Password:  entity.Password.String(),
		CreatedAt: entity.Meta.CreatedAt,
		UpdatedAt: entity.Meta.UpdatedAt,
	}
}

func (m UserModel) ToDomain() *domainuser.User {
	return &domainuser.User{
		ID:       shared.ID(strconv.FormatUint(m.ID, 10)),
		Account:  m.Account,
		Name:     m.Name,
		Email:    domainuser.Email(m.Email),
		Status:   domainuser.Status(m.Status),
		Password: domainuser.PasswordHash(m.Password),
		Meta: shared.AuditMeta{
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
	}
}

func parseUserID(raw string) uint64 {
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return v
}
