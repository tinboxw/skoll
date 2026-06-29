package gormrepo

import (
	"strings"
	"time"

	domainorg "github.com/tinboxw/skoll/internal/domain/organization"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type DepartmentModel struct {
	ID           string `gorm:"size:64;primaryKey"`
	ParentID     string `gorm:"size:64;index:idx_departments_parent_sort"`
	Code         string `gorm:"size:128;uniqueIndex"`
	Name         string `gorm:"size:255"`
	LeaderUserID string `gorm:"size:64;index"`
	Status       string `gorm:"size:32;index:idx_departments_status_sort"`
	Sort         int    `gorm:"index:idx_departments_parent_sort;index:idx_departments_status_sort"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (DepartmentModel) TableName() string { return "sk_departments" }

type PositionModel struct {
	ID          string `gorm:"size:64;primaryKey"`
	Code        string `gorm:"size:128;uniqueIndex"`
	Name        string `gorm:"size:255"`
	Description string `gorm:"size:512"`
	Status      string `gorm:"size:32;index:idx_positions_status_sort"`
	Sort        int    `gorm:"index:idx_positions_status_sort"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (PositionModel) TableName() string { return "sk_positions" }

type UserAssignmentModel struct {
	UserID            string `gorm:"size:64;primaryKey"`
	DepartmentID      string `gorm:"size:64;index:idx_user_assignments_department"`
	PositionID        string `gorm:"size:64;index:idx_user_assignments_position"`
	PrimaryAssignment bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (UserAssignmentModel) TableName() string { return "sk_user_organization_assignments" }

func DepartmentModelFromDomain(item domainorg.Department) DepartmentModel {
	return DepartmentModel{
		ID:           strings.TrimSpace(item.ID.String()),
		ParentID:     strings.TrimSpace(item.ParentID.String()),
		Code:         normalizeOrganizationCode(item.Code),
		Name:         strings.TrimSpace(item.Name),
		LeaderUserID: strings.TrimSpace(item.LeaderUserID.String()),
		Status:       string(item.Status),
		Sort:         item.Sort,
		CreatedAt:    item.Meta.CreatedAt,
		UpdatedAt:    item.Meta.UpdatedAt,
	}
}

func (m DepartmentModel) ToDomain() (*domainorg.Department, error) {
	return domainorg.NewDepartment(domainorg.DepartmentInput{
		ID:           shared.ID(strings.TrimSpace(m.ID)),
		ParentID:     shared.ID(strings.TrimSpace(m.ParentID)),
		Code:         m.Code,
		Name:         m.Name,
		LeaderUserID: shared.ID(strings.TrimSpace(m.LeaderUserID)),
		Status:       domainorg.Status(strings.TrimSpace(m.Status)),
		Sort:         m.Sort,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	})
}

func PositionModelFromDomain(item domainorg.Position) PositionModel {
	return PositionModel{
		ID:          strings.TrimSpace(item.ID.String()),
		Code:        normalizeOrganizationCode(item.Code),
		Name:        strings.TrimSpace(item.Name),
		Description: strings.TrimSpace(item.Description),
		Status:      string(item.Status),
		Sort:        item.Sort,
		CreatedAt:   item.Meta.CreatedAt,
		UpdatedAt:   item.Meta.UpdatedAt,
	}
}

func (m PositionModel) ToDomain() (*domainorg.Position, error) {
	return domainorg.NewPosition(domainorg.PositionInput{
		ID:          shared.ID(strings.TrimSpace(m.ID)),
		Code:        m.Code,
		Name:        m.Name,
		Description: m.Description,
		Status:      domainorg.Status(strings.TrimSpace(m.Status)),
		Sort:        m.Sort,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	})
}

func UserAssignmentModelFromDomain(item domainorg.UserAssignment) UserAssignmentModel {
	return UserAssignmentModel{
		UserID:            strings.TrimSpace(item.UserID.String()),
		DepartmentID:      strings.TrimSpace(item.DepartmentID.String()),
		PositionID:        strings.TrimSpace(item.PositionID.String()),
		PrimaryAssignment: item.Primary,
		CreatedAt:         item.Meta.CreatedAt,
		UpdatedAt:         item.Meta.UpdatedAt,
	}
}

func (m UserAssignmentModel) ToDomain() (*domainorg.UserAssignment, error) {
	return domainorg.NewUserAssignment(domainorg.UserAssignmentInput{
		UserID:       shared.ID(strings.TrimSpace(m.UserID)),
		DepartmentID: shared.ID(strings.TrimSpace(m.DepartmentID)),
		PositionID:   shared.ID(strings.TrimSpace(m.PositionID)),
		Primary:      m.PrimaryAssignment,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	})
}

func normalizeOrganizationCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
