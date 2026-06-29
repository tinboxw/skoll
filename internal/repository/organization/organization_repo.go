package organization

import (
	"context"

	"github.com/tinboxw/skoll/internal/domain/organization"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type OrganizationRepository interface {
	GetDepartmentByID(ctx context.Context, id shared.ID) (*organization.Department, error)
	GetDepartmentByCode(ctx context.Context, code string) (*organization.Department, error)
	ListDepartments(ctx context.Context, offset, limit int) ([]organization.Department, error)
	SaveDepartment(ctx context.Context, department *organization.Department) error
	DeleteDepartment(ctx context.Context, id shared.ID) error

	GetPositionByID(ctx context.Context, id shared.ID) (*organization.Position, error)
	GetPositionByCode(ctx context.Context, code string) (*organization.Position, error)
	ListPositions(ctx context.Context, offset, limit int) ([]organization.Position, error)
	SavePosition(ctx context.Context, position *organization.Position) error
	DeletePosition(ctx context.Context, id shared.ID) error

	GetUserAssignment(ctx context.Context, userID shared.ID) (*organization.UserAssignment, error)
	ListUserAssignmentsByDepartment(ctx context.Context, departmentIDs []shared.ID, offset, limit int) ([]organization.UserAssignment, error)
	SaveUserAssignment(ctx context.Context, assignment *organization.UserAssignment) error
	DeleteUserAssignment(ctx context.Context, userID shared.ID) error
}
