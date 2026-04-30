package persistent

import (
	"context"
	"errors"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/tinboxw/skoll/internal/module/role"
	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
)

// RoleModel is the gorm representation of role.Role. Permissions is stored
// as a newline-separated string to stay portable across mysql/pg/sqlite
// without requiring JSON column support; the (de)serialization is local to
// the repository.
type RoleModel struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	Name        string `gorm:"size:128;uniqueIndex"`
	Permissions string `gorm:"type:text"`
}

func (RoleModel) TableName() string { return "skoll_roles" }

func (m RoleModel) toDomain() role.Role {
	return role.Role{
		ID:          m.ID,
		Name:        m.Name,
		Permissions: splitNonEmpty(m.Permissions, "\n"),
	}
}

// RoleRepository is a SQL-backed implementation of contracts.RoleRepository.
type RoleRepository struct {
	db *gorm.DB
}

// NewRoleRepository constructs a SQL-backed role repository.
func NewRoleRepository(db *gorm.DB) *RoleRepository { return &RoleRepository{db: db} }

func (r *RoleRepository) ctx() context.Context { return context.Background() }

// Models returns gorm models owned by this repo for migration registration.
func (r *RoleRepository) Models() []any { return []any{&RoleModel{}} }

// Create persists a new role.
func (r *RoleRepository) Create(name string, permissions []string) role.Role {
	row := RoleModel{
		Name:        name,
		Permissions: strings.Join(permissions, "\n"),
	}
	if err := r.db.WithContext(r.ctx()).Create(&row).Error; err != nil {
		panic(err)
	}
	return row.toDomain()
}

// Get returns the role with the supplied id.
func (r *RoleRepository) Get(id int64) (role.Role, error) {
	var row RoleModel
	if err := r.db.WithContext(r.ctx()).First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return role.Role{}, role.ErrRoleNotFound
		}
		return role.Role{}, err
	}
	return row.toDomain(), nil
}

// List returns all roles ordered by id ascending.
func (r *RoleRepository) List() []role.Role {
	var rows []RoleModel
	if err := r.db.WithContext(r.ctx()).Find(&rows).Error; err != nil {
		panic(err)
	}
	out := make([]role.Role, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func splitNonEmpty(s, sep string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// Compile-time interface check.
var _ contracts.RoleRepository = (*RoleRepository)(nil)
