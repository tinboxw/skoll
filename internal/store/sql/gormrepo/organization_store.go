package gormrepo

import (
	"context"
	"fmt"
	"sort"
	"strings"

	domainorg "github.com/tinboxw/skoll/internal/domain/organization"
	"github.com/tinboxw/skoll/internal/domain/shared"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OrganizationStore struct {
	db *gorm.DB
}

func NewOrganizationStore(db *gorm.DB) *OrganizationStore {
	return &OrganizationStore{db: db}
}

func (s *OrganizationStore) GetDepartmentByID(ctx context.Context, id shared.ID) (*domainorg.Department, error) {
	if id.IsZero() {
		return nil, nil
	}
	var row DepartmentModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Where("id = ?", id.String()).First(&row).Error
	})
	return departmentFromRow(row, err)
}

func (s *OrganizationStore) GetDepartmentByCode(ctx context.Context, code string) (*domainorg.Department, error) {
	code = normalizeOrganizationCode(code)
	if code == "" {
		return nil, nil
	}
	var row DepartmentModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Where("code = ?", code).First(&row).Error
	})
	return departmentFromRow(row, err)
}

func (s *OrganizationStore) ListDepartments(ctx context.Context, offset, limit int) ([]domainorg.Department, error) {
	q := s.db.WithContext(ctx).Model(&DepartmentModel{}).Order("sort asc").Order("code asc")
	if offset > 0 {
		q = q.Offset(offset)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []DepartmentModel
	if err := withDBRetry(func() error { return q.Find(&rows).Error }); err != nil {
		return nil, err
	}
	return departmentsFromRows(rows)
}

func (s *OrganizationStore) SaveDepartment(ctx context.Context, department *domainorg.Department) error {
	if department == nil {
		return fmt.Errorf("department is required")
	}
	row := DepartmentModelFromDomain(*department)
	if _, err := row.ToDomain(); err != nil {
		return err
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if row.ParentID != "" {
				var parentCount int64
				if err := tx.Model(&DepartmentModel{}).Where("id = ?", row.ParentID).Count(&parentCount).Error; err != nil {
					return err
				}
				if parentCount == 0 {
					return fmt.Errorf("department parent does not exist")
				}
			}
			var rows []DepartmentModel
			if err := tx.Find(&rows).Error; err != nil {
				return err
			}
			departments, err := departmentsFromRows(rows)
			if err != nil {
				return err
			}
			next := make([]domainorg.Department, 0, len(departments)+1)
			for _, item := range departments {
				if item.ID != shared.ID(row.ID) {
					next = append(next, item)
				}
			}
			normalized, err := row.ToDomain()
			if err != nil {
				return err
			}
			next = append(next, *normalized)
			if err := domainorg.ValidateDepartmentTree(next); err != nil {
				return err
			}
			return tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"parent_id",
					"code",
					"name",
					"leader_user_id",
					"status",
					"sort",
					"updated_at",
				}),
			}).Create(&row).Error
		})
	})
}

func (s *OrganizationStore) DeleteDepartment(ctx context.Context, id shared.ID) error {
	if id.IsZero() {
		return nil
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			ids, err := departmentSubtreeIDs(ctx, tx, id.String())
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				return nil
			}
			if err := tx.Where("department_id IN ?", ids).Delete(&UserAssignmentModel{}).Error; err != nil {
				return err
			}
			return tx.Where("id IN ?", ids).Delete(&DepartmentModel{}).Error
		})
	})
}

func (s *OrganizationStore) GetPositionByID(ctx context.Context, id shared.ID) (*domainorg.Position, error) {
	if id.IsZero() {
		return nil, nil
	}
	var row PositionModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Where("id = ?", id.String()).First(&row).Error
	})
	return positionFromRow(row, err)
}

func (s *OrganizationStore) GetPositionByCode(ctx context.Context, code string) (*domainorg.Position, error) {
	code = normalizeOrganizationCode(code)
	if code == "" {
		return nil, nil
	}
	var row PositionModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Where("code = ?", code).First(&row).Error
	})
	return positionFromRow(row, err)
}

func (s *OrganizationStore) ListPositions(ctx context.Context, offset, limit int) ([]domainorg.Position, error) {
	q := s.db.WithContext(ctx).Model(&PositionModel{}).Order("sort asc").Order("code asc")
	if offset > 0 {
		q = q.Offset(offset)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []PositionModel
	if err := withDBRetry(func() error { return q.Find(&rows).Error }); err != nil {
		return nil, err
	}
	out := make([]domainorg.Position, 0, len(rows))
	for _, row := range rows {
		item, err := row.ToDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, nil
}

func (s *OrganizationStore) SavePosition(ctx context.Context, position *domainorg.Position) error {
	if position == nil {
		return fmt.Errorf("position is required")
	}
	row := PositionModelFromDomain(*position)
	if _, err := row.ToDomain(); err != nil {
		return err
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"code",
				"name",
				"description",
				"status",
				"sort",
				"updated_at",
			}),
		}).Create(&row).Error
	})
}

func (s *OrganizationStore) DeletePosition(ctx context.Context, id shared.ID) error {
	if id.IsZero() {
		return nil
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&UserAssignmentModel{}).Where("position_id = ?", id.String()).UpdateColumn("position_id", "").Error; err != nil {
				return err
			}
			return tx.Delete(&PositionModel{}, "id = ?", id.String()).Error
		})
	})
}

func (s *OrganizationStore) GetUserAssignment(ctx context.Context, userID shared.ID) (*domainorg.UserAssignment, error) {
	if userID.IsZero() {
		return nil, nil
	}
	var row UserAssignmentModel
	err := withDBRetry(func() error {
		return s.db.WithContext(ctx).Where("user_id = ?", userID.String()).First(&row).Error
	})
	return userAssignmentFromRow(row, err)
}

func (s *OrganizationStore) ListUserAssignmentsByDepartment(ctx context.Context, departmentIDs []shared.ID, offset, limit int) ([]domainorg.UserAssignment, error) {
	ids := make([]string, 0, len(departmentIDs))
	for _, id := range departmentIDs {
		if !id.IsZero() {
			ids = append(ids, id.String())
		}
	}
	q := s.db.WithContext(ctx).Model(&UserAssignmentModel{}).Order("user_id asc")
	if len(ids) > 0 {
		q = q.Where("department_id IN ?", ids)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []UserAssignmentModel
	if err := withDBRetry(func() error { return q.Find(&rows).Error }); err != nil {
		return nil, err
	}
	out := make([]domainorg.UserAssignment, 0, len(rows))
	for _, row := range rows {
		item, err := row.ToDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, nil
}

func (s *OrganizationStore) SaveUserAssignment(ctx context.Context, assignment *domainorg.UserAssignment) error {
	if assignment == nil {
		return fmt.Errorf("user assignment is required")
	}
	row := UserAssignmentModelFromDomain(*assignment)
	if _, err := row.ToDomain(); err != nil {
		return err
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var departmentCount int64
			if err := tx.Model(&DepartmentModel{}).Where("id = ?", row.DepartmentID).Count(&departmentCount).Error; err != nil {
				return err
			}
			if departmentCount == 0 {
				return fmt.Errorf("department does not exist")
			}
			if row.PositionID != "" {
				var positionCount int64
				if err := tx.Model(&PositionModel{}).Where("id = ?", row.PositionID).Count(&positionCount).Error; err != nil {
					return err
				}
				if positionCount == 0 {
					return fmt.Errorf("position does not exist")
				}
			}
			return tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "user_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"department_id",
					"position_id",
					"primary_assignment",
					"updated_at",
				}),
			}).Create(&row).Error
		})
	})
}

func (s *OrganizationStore) DeleteUserAssignment(ctx context.Context, userID shared.ID) error {
	if userID.IsZero() {
		return nil
	}
	return withDBRetry(func() error {
		return s.db.WithContext(ctx).Delete(&UserAssignmentModel{}, "user_id = ?", userID.String()).Error
	})
}

func departmentsFromRows(rows []DepartmentModel) ([]domainorg.Department, error) {
	out := make([]domainorg.Department, 0, len(rows))
	for _, row := range rows {
		item, err := row.ToDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, nil
}

func departmentFromRow(row DepartmentModel, err error) (*domainorg.Department, error) {
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return row.ToDomain()
}

func positionFromRow(row PositionModel, err error) (*domainorg.Position, error) {
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return row.ToDomain()
}

func userAssignmentFromRow(row UserAssignmentModel, err error) (*domainorg.UserAssignment, error) {
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return row.ToDomain()
}

func departmentSubtreeIDs(ctx context.Context, tx *gorm.DB, rootID string) ([]string, error) {
	var rows []DepartmentModel
	if err := tx.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	children := map[string][]string{}
	known := map[string]struct{}{}
	for _, row := range rows {
		known[row.ID] = struct{}{}
		children[row.ParentID] = append(children[row.ParentID], row.ID)
	}
	if _, ok := known[rootID]; !ok {
		return nil, nil
	}
	out := []string{}
	seen := map[string]struct{}{}
	var visit func(string)
	visit = func(id string) {
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
		sort.Strings(children[id])
		for _, childID := range children[id] {
			visit(childID)
		}
	}
	visit(strings.TrimSpace(rootID))
	return out, nil
}
