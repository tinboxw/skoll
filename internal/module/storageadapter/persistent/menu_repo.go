package persistent

import (
	"context"
	"errors"
	"sort"

	"gorm.io/gorm"

	"github.com/tinboxw/skoll/internal/module/menu"
	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
)

// MenuItemModel is the gorm representation of menu.Item.
type MenuItemModel struct {
	ID     int64  `gorm:"primaryKey;autoIncrement"`
	Title  string `gorm:"size:255"`
	Path   string `gorm:"size:255"`
	Order  int    `gorm:"column:menu_order;index"`
	Hidden bool
}

func (MenuItemModel) TableName() string { return "skoll_menu_items" }

func (m MenuItemModel) toDomain() menu.Item {
	return menu.Item{
		ID:     m.ID,
		Title:  m.Title,
		Path:   m.Path,
		Order:  m.Order,
		Hidden: m.Hidden,
	}
}

// MenuRepository is a SQL-backed implementation of contracts.MenuRepository.
type MenuRepository struct {
	db *gorm.DB
}

// NewMenuRepository constructs a SQL-backed menu repository.
func NewMenuRepository(db *gorm.DB) *MenuRepository { return &MenuRepository{db: db} }

func (r *MenuRepository) ctx() context.Context { return context.Background() }

// Models returns gorm models owned by this repo for migration registration.
func (r *MenuRepository) Models() []any { return []any{&MenuItemModel{}} }

// Create inserts a new menu item.
func (r *MenuRepository) Create(title, path string, order int) menu.Item {
	row := MenuItemModel{Title: title, Path: path, Order: order}
	if err := r.db.WithContext(r.ctx()).Create(&row).Error; err != nil {
		panic(err)
	}
	return row.toDomain()
}

// Get returns the menu item with the supplied id.
func (r *MenuRepository) Get(id int64) (menu.Item, error) {
	var row MenuItemModel
	if err := r.db.WithContext(r.ctx()).First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return menu.Item{}, menu.ErrMenuNotFound
		}
		return menu.Item{}, err
	}
	return row.toDomain(), nil
}

// List returns all menu items ordered by (order asc, id asc) to match memory.
func (r *MenuRepository) List() []menu.Item {
	var rows []MenuItemModel
	if err := r.db.WithContext(r.ctx()).Find(&rows).Error; err != nil {
		panic(err)
	}
	out := make([]menu.Item, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Order == out[j].Order {
			return out[i].ID < out[j].ID
		}
		return out[i].Order < out[j].Order
	})
	return out
}

// Compile-time interface check.
var _ contracts.MenuRepository = (*MenuRepository)(nil)
