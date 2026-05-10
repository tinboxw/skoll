package model

import (
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainuser "github.com/tinboxw/skoll/internal/domain/user"
)

type UserModel struct {
	ID          string `gorm:"primaryKey;size:128"`
	Username    string `gorm:"size:128;index"`
	DisplayName string `gorm:"size:128"`
	Email       string `gorm:"size:191;uniqueIndex"`
	Status      string `gorm:"size:32"`
	Password    string `gorm:"size:256"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (UserModel) TableName() string { return "sk_users" }

func UserModelFromDomain(entity *domainuser.User) UserModel {
	return UserModel{
		ID:          entity.ID.String(),
		Username:    entity.Username,
		DisplayName: entity.DisplayName,
		Email:       entity.Email.String(),
		Status:      string(entity.Status),
		Password:    entity.Password.String(),
		CreatedAt:   entity.Meta.CreatedAt,
		UpdatedAt:   entity.Meta.UpdatedAt,
	}
}

func (m UserModel) ToDomain() *domainuser.User {
	return &domainuser.User{
		ID:          shared.ID(m.ID),
		Username:    m.Username,
		DisplayName: m.DisplayName,
		Email:       domainuser.Email(m.Email),
		Status:      domainuser.Status(m.Status),
		Password:    domainuser.PasswordHash(m.Password),
		Meta: shared.AuditMeta{
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
	}
}
