package system

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type DictionaryStatus string

const (
	DictionaryStatusEnabled  DictionaryStatus = "enabled"
	DictionaryStatusDisabled DictionaryStatus = "disabled"
)

var dictionaryCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,127}$`)

type DictionaryTypeInput struct {
	ID          shared.ID
	Code        string
	Name        string
	Description string
	Status      DictionaryStatus
	Sort        int
	Builtin     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type DictionaryItemInput struct {
	ID        shared.ID
	TypeCode  string
	Label     string
	Value     string
	Status    DictionaryStatus
	Sort      int
	Builtin   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DictionaryType struct {
	ID          shared.ID
	Code        string
	Name        string
	Description string
	Status      DictionaryStatus
	Sort        int
	Builtin     bool
	Meta        shared.AuditMeta
}

type DictionaryItem struct {
	ID       shared.ID
	TypeCode string
	Label    string
	Value    string
	Status   DictionaryStatus
	Sort     int
	Builtin  bool
	Meta     shared.AuditMeta
}

func NewDictionaryType(in DictionaryTypeInput) (*DictionaryType, error) {
	in = normalizeDictionaryTypeInput(in)
	if err := validateDictionaryTypeInput(in); err != nil {
		return nil, err
	}
	item := &DictionaryType{
		ID:          in.ID,
		Code:        in.Code,
		Name:        in.Name,
		Description: in.Description,
		Status:      in.Status,
		Sort:        in.Sort,
		Builtin:     in.Builtin,
	}
	item.Meta.CreatedAt = in.CreatedAt
	item.Meta.UpdatedAt = in.UpdatedAt
	return item, nil
}

func NewDictionaryItem(in DictionaryItemInput) (*DictionaryItem, error) {
	in = normalizeDictionaryItemInput(in)
	if err := validateDictionaryItemInput(in); err != nil {
		return nil, err
	}
	item := &DictionaryItem{
		ID:       in.ID,
		TypeCode: in.TypeCode,
		Label:    in.Label,
		Value:    in.Value,
		Status:   in.Status,
		Sort:     in.Sort,
		Builtin:  in.Builtin,
	}
	item.Meta.CreatedAt = in.CreatedAt
	item.Meta.UpdatedAt = in.UpdatedAt
	return item, nil
}

func (d *DictionaryType) Enable(now time.Time) {
	d.Status = DictionaryStatusEnabled
	d.Meta.Touch(now)
}

func (d *DictionaryType) Disable(now time.Time) {
	d.Status = DictionaryStatusDisabled
	d.Meta.Touch(now)
}

func (d *DictionaryType) Reorder(sortValue int, now time.Time) error {
	if sortValue < 0 {
		return fmt.Errorf("dictionary sort must be non-negative")
	}
	d.Sort = sortValue
	d.Meta.Touch(now)
	return nil
}

func (d DictionaryType) RequireMutable() error {
	if d.Builtin {
		return fmt.Errorf("builtin dictionary type cannot be modified")
	}
	return nil
}

func (i *DictionaryItem) Enable(now time.Time) {
	i.Status = DictionaryStatusEnabled
	i.Meta.Touch(now)
}

func (i *DictionaryItem) Disable(now time.Time) {
	i.Status = DictionaryStatusDisabled
	i.Meta.Touch(now)
}

func (i *DictionaryItem) Reorder(sortValue int, now time.Time) error {
	if sortValue < 0 {
		return fmt.Errorf("dictionary item sort must be non-negative")
	}
	i.Sort = sortValue
	i.Meta.Touch(now)
	return nil
}

func (i DictionaryItem) RequireMutable() error {
	if i.Builtin {
		return fmt.Errorf("builtin dictionary item cannot be modified")
	}
	return nil
}

func SortDictionaryItems(items []DictionaryItem) []DictionaryItem {
	out := append([]DictionaryItem(nil), items...)
	sort.SliceStable(out, func(left, right int) bool {
		if out[left].Sort != out[right].Sort {
			return out[left].Sort < out[right].Sort
		}
		return out[left].Value < out[right].Value
	})
	return out
}

func normalizeDictionaryTypeInput(in DictionaryTypeInput) DictionaryTypeInput {
	in.Code = normalizeDictionaryCode(in.Code)
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	if in.Status == "" {
		in.Status = DictionaryStatusEnabled
	}
	if in.UpdatedAt.IsZero() {
		in.UpdatedAt = in.CreatedAt
	}
	return in
}

func normalizeDictionaryItemInput(in DictionaryItemInput) DictionaryItemInput {
	in.TypeCode = normalizeDictionaryCode(in.TypeCode)
	in.Label = strings.TrimSpace(in.Label)
	in.Value = strings.TrimSpace(in.Value)
	if in.Status == "" {
		in.Status = DictionaryStatusEnabled
	}
	if in.UpdatedAt.IsZero() {
		in.UpdatedAt = in.CreatedAt
	}
	return in
}

func normalizeDictionaryCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

func validateDictionaryTypeInput(in DictionaryTypeInput) error {
	if in.ID.IsZero() {
		return fmt.Errorf("dictionary type id is required")
	}
	if err := validateDictionaryCode(in.Code, "type code"); err != nil {
		return err
	}
	if err := validateDictionaryName(in.Name, "type name"); err != nil {
		return err
	}
	if err := validateDictionaryStatus(in.Status); err != nil {
		return err
	}
	if in.Sort < 0 {
		return fmt.Errorf("dictionary sort must be non-negative")
	}
	return validateDictionaryTimestamps(in.CreatedAt, in.UpdatedAt)
}

func validateDictionaryItemInput(in DictionaryItemInput) error {
	if in.ID.IsZero() {
		return fmt.Errorf("dictionary item id is required")
	}
	if err := validateDictionaryCode(in.TypeCode, "type code"); err != nil {
		return err
	}
	if err := validateDictionaryName(in.Label, "item label"); err != nil {
		return err
	}
	if err := validateDictionaryName(in.Value, "item value"); err != nil {
		return err
	}
	if err := validateDictionaryStatus(in.Status); err != nil {
		return err
	}
	if in.Sort < 0 {
		return fmt.Errorf("dictionary item sort must be non-negative")
	}
	return validateDictionaryTimestamps(in.CreatedAt, in.UpdatedAt)
}

func validateDictionaryCode(code, label string) error {
	if !dictionaryCodePattern.MatchString(code) {
		return fmt.Errorf("dictionary %s must match %s", label, dictionaryCodePattern.String())
	}
	return nil
}

func validateDictionaryName(value, label string) error {
	if value == "" {
		return fmt.Errorf("dictionary %s is required", label)
	}
	if len([]rune(value)) > 255 {
		return fmt.Errorf("dictionary %s is too long", label)
	}
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("dictionary %s must not contain line breaks", label)
	}
	return nil
}

func validateDictionaryStatus(status DictionaryStatus) error {
	switch status {
	case DictionaryStatusEnabled, DictionaryStatusDisabled:
		return nil
	default:
		return fmt.Errorf("dictionary status is invalid")
	}
}

func validateDictionaryTimestamps(createdAt, updatedAt time.Time) error {
	if createdAt.IsZero() {
		return fmt.Errorf("dictionary created time is required")
	}
	if updatedAt.IsZero() {
		return fmt.Errorf("dictionary updated time is required")
	}
	if updatedAt.Before(createdAt) {
		return fmt.Errorf("dictionary updated time must not be before created time")
	}
	return nil
}
