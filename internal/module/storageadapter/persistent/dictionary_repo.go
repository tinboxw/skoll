package persistent

import (
	"context"
	"errors"
	"sort"

	"gorm.io/gorm"

	"github.com/tinboxw/skoll/internal/module/dictionary"
	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
)

// DictionaryItemModel is the gorm representation of dictionary.Item.
type DictionaryItemModel struct {
	ID      int64  `gorm:"primaryKey;autoIncrement"`
	Type    string `gorm:"column:dict_type;size:128;index:idx_dict_type_sort,priority:1"`
	Label   string `gorm:"size:255"`
	Value   string `gorm:"size:255"`
	Sort    int    `gorm:"index:idx_dict_type_sort,priority:2"`
	Enabled bool
}

func (DictionaryItemModel) TableName() string { return "skoll_dictionary_items" }

func (m DictionaryItemModel) toDomain() dictionary.Item {
	return dictionary.Item{
		ID:      m.ID,
		Type:    m.Type,
		Label:   m.Label,
		Value:   m.Value,
		Sort:    m.Sort,
		Enabled: m.Enabled,
	}
}

// DictionaryRepository is a SQL-backed implementation of contracts.DictionaryRepository.
type DictionaryRepository struct {
	db *gorm.DB
}

// NewDictionaryRepository constructs a SQL-backed dictionary repository.
func NewDictionaryRepository(db *gorm.DB) *DictionaryRepository {
	return &DictionaryRepository{db: db}
}

func (r *DictionaryRepository) ctx() context.Context { return context.Background() }

// Models returns gorm models owned by this repo for migration registration.
func (r *DictionaryRepository) Models() []any { return []any{&DictionaryItemModel{}} }

// Create inserts a new dictionary item and returns the stored value.
func (r *DictionaryRepository) Create(itemType, label, value string, sortOrder int, enabled bool) dictionary.Item {
	row := DictionaryItemModel{
		Type:    itemType,
		Label:   label,
		Value:   value,
		Sort:    sortOrder,
		Enabled: enabled,
	}
	if err := r.db.WithContext(r.ctx()).Create(&row).Error; err != nil {
		panic(err)
	}
	return row.toDomain()
}

// Get returns the dictionary item with the supplied id.
func (r *DictionaryRepository) Get(id int64) (dictionary.Item, error) {
	var row DictionaryItemModel
	if err := r.db.WithContext(r.ctx()).First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dictionary.Item{}, dictionary.ErrItemNotFound
		}
		return dictionary.Item{}, err
	}
	return row.toDomain(), nil
}

// List returns all dictionary items.
func (r *DictionaryRepository) List() []dictionary.Item {
	return r.fetch(func(tx *gorm.DB) *gorm.DB { return tx })
}

// ListByType returns dictionary items filtered by type.
func (r *DictionaryRepository) ListByType(itemType string) []dictionary.Item {
	return r.fetch(func(tx *gorm.DB) *gorm.DB { return tx.Where("dict_type = ?", itemType) })
}

func (r *DictionaryRepository) fetch(filter func(*gorm.DB) *gorm.DB) []dictionary.Item {
	var rows []DictionaryItemModel
	tx := r.db.WithContext(r.ctx()).Model(&DictionaryItemModel{})
	if filter != nil {
		tx = filter(tx)
	}
	if err := tx.Find(&rows).Error; err != nil {
		panic(err)
	}
	out := make([]dictionary.Item, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	sortDictionaryItems(out)
	return out
}

func sortDictionaryItems(items []dictionary.Item) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Type == items[j].Type {
			if items[i].Sort == items[j].Sort {
				return items[i].ID < items[j].ID
			}
			return items[i].Sort < items[j].Sort
		}
		return items[i].Type < items[j].Type
	})
}

// Compile-time interface check.
var _ contracts.DictionaryRepository = (*DictionaryRepository)(nil)
