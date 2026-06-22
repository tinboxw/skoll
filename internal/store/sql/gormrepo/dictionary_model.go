package gormrepo

import (
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
)

type DictionaryTypeModel struct {
	ID          string `gorm:"size:64;primaryKey"`
	Code        string `gorm:"size:128;uniqueIndex"`
	Name        string `gorm:"size:255"`
	Description string `gorm:"size:512"`
	Status      string `gorm:"size:32;index:idx_dictionary_types_status_sort"`
	Sort        int    `gorm:"index:idx_dictionary_types_status_sort"`
	Builtin     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (DictionaryTypeModel) TableName() string { return "sk_dictionary_types" }

type DictionaryItemModel struct {
	ID        string `gorm:"size:64;primaryKey"`
	TypeCode  string `gorm:"size:128;uniqueIndex:uk_dictionary_items_type_value;index:idx_dictionary_items_type_sort"`
	Label     string `gorm:"size:255"`
	Value     string `gorm:"size:255;uniqueIndex:uk_dictionary_items_type_value"`
	Status    string `gorm:"size:32;index:idx_dictionary_items_status_sort"`
	Sort      int    `gorm:"index:idx_dictionary_items_type_sort;index:idx_dictionary_items_status_sort"`
	Builtin   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (DictionaryItemModel) TableName() string { return "sk_dictionary_items" }

func DictionaryTypeModelFromDomain(item domainsystem.DictionaryType) DictionaryTypeModel {
	return DictionaryTypeModel{
		ID:          strings.TrimSpace(item.ID.String()),
		Code:        normalizeDictionaryCode(item.Code),
		Name:        strings.TrimSpace(item.Name),
		Description: strings.TrimSpace(item.Description),
		Status:      string(item.Status),
		Sort:        item.Sort,
		Builtin:     item.Builtin,
		CreatedAt:   item.Meta.CreatedAt,
		UpdatedAt:   item.Meta.UpdatedAt,
	}
}

func (m DictionaryTypeModel) ToDomain() (*domainsystem.DictionaryType, error) {
	return domainsystem.NewDictionaryType(domainsystem.DictionaryTypeInput{
		ID:          shared.ID(strings.TrimSpace(m.ID)),
		Code:        m.Code,
		Name:        m.Name,
		Description: m.Description,
		Status:      domainsystem.DictionaryStatus(strings.TrimSpace(m.Status)),
		Sort:        m.Sort,
		Builtin:     m.Builtin,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	})
}

func DictionaryItemModelFromDomain(item domainsystem.DictionaryItem) DictionaryItemModel {
	return DictionaryItemModel{
		ID:        strings.TrimSpace(item.ID.String()),
		TypeCode:  normalizeDictionaryCode(item.TypeCode),
		Label:     strings.TrimSpace(item.Label),
		Value:     strings.TrimSpace(item.Value),
		Status:    string(item.Status),
		Sort:      item.Sort,
		Builtin:   item.Builtin,
		CreatedAt: item.Meta.CreatedAt,
		UpdatedAt: item.Meta.UpdatedAt,
	}
}

func (m DictionaryItemModel) ToDomain() (*domainsystem.DictionaryItem, error) {
	return domainsystem.NewDictionaryItem(domainsystem.DictionaryItemInput{
		ID:        shared.ID(strings.TrimSpace(m.ID)),
		TypeCode:  m.TypeCode,
		Label:     m.Label,
		Value:     m.Value,
		Status:    domainsystem.DictionaryStatus(strings.TrimSpace(m.Status)),
		Sort:      m.Sort,
		Builtin:   m.Builtin,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	})
}

func normalizeDictionaryCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
