package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/domain/system"
)

type SystemStore struct {
	mu            sync.RWMutex
	items         map[shared.ID]*system.Setting
	byKey         map[string]shared.ID
	dictTypes     map[shared.ID]system.DictionaryType
	dictTypeCodes map[string]shared.ID
	dictItems     map[shared.ID]system.DictionaryItem
	dictItemKeys  map[string]shared.ID
}

func NewSystemStore() *SystemStore {
	return &SystemStore{
		items:         make(map[shared.ID]*system.Setting),
		byKey:         make(map[string]shared.ID),
		dictTypes:     make(map[shared.ID]system.DictionaryType),
		dictTypeCodes: make(map[string]shared.ID),
		dictItems:     make(map[shared.ID]system.DictionaryItem),
		dictItemKeys:  make(map[string]shared.ID),
	}
}

func (s *SystemStore) GetSettingByID(_ context.Context, id shared.ID) (*system.Setting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.items[id], nil
}

func (s *SystemStore) GetSettingByKey(_ context.Context, key string) (*system.Setting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.byKey[normalizeSettingKey(key)]
	if !ok {
		return nil, nil
	}
	return s.items[id], nil
}

func (s *SystemStore) ListSettings(_ context.Context, offset, limit int) ([]*system.Setting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.byKey))
	for key := range s.byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = len(keys)
	}
	if offset > len(keys) {
		return []*system.Setting{}, nil
	}
	end := offset + limit
	if end > len(keys) {
		end = len(keys)
	}

	out := make([]*system.Setting, 0, end-offset)
	for _, key := range keys[offset:end] {
		out = append(out, s.items[s.byKey[key]])
	}
	return out, nil
}

func (s *SystemStore) SaveSetting(_ context.Context, setting *system.Setting) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[setting.ID] = setting
	s.byKey[normalizeSettingKey(setting.Key)] = setting.ID
	return nil
}

func (s *SystemStore) DeleteSetting(_ context.Context, id shared.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if ok && item != nil {
		delete(s.byKey, normalizeSettingKey(item.Key))
	}
	delete(s.items, id)
	return nil
}

func normalizeSettingKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

func (s *SystemStore) GetDictionaryTypeByID(_ context.Context, id shared.ID) (*system.DictionaryType, error) {
	if id.IsZero() {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.dictTypes[id]
	if !ok {
		return nil, nil
	}
	return cloneDictionaryType(item), nil
}

func (s *SystemStore) GetDictionaryTypeByCode(_ context.Context, code string) (*system.DictionaryType, error) {
	code = normalizeDictionaryCode(code)
	if code == "" {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.dictTypeCodes[code]
	if !ok {
		return nil, nil
	}
	item := s.dictTypes[id]
	return cloneDictionaryType(item), nil
}

func (s *SystemStore) ListDictionaryTypes(_ context.Context, offset, limit int) ([]system.DictionaryType, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]system.DictionaryType, 0, len(s.dictTypes))
	for _, item := range s.dictTypes {
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Sort != out[j].Sort {
			return out[i].Sort < out[j].Sort
		}
		return out[i].Code < out[j].Code
	})
	return paginateDictionaryTypes(out, offset, limit), nil
}

func (s *SystemStore) SaveDictionaryType(_ context.Context, dictType *system.DictionaryType) error {
	if dictType == nil {
		return fmt.Errorf("dictionary type is required")
	}
	normalized, err := system.NewDictionaryType(system.DictionaryTypeInput{
		ID:          dictType.ID,
		Code:        dictType.Code,
		Name:        dictType.Name,
		Description: dictType.Description,
		Status:      dictType.Status,
		Sort:        dictType.Sort,
		Builtin:     dictType.Builtin,
		CreatedAt:   dictType.Meta.CreatedAt,
		UpdatedAt:   dictType.Meta.UpdatedAt,
	})
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if existingID, ok := s.dictTypeCodes[normalized.Code]; ok && existingID != normalized.ID {
		return fmt.Errorf("dictionary type code already exists")
	}
	if old, ok := s.dictTypes[normalized.ID]; ok && old.Code != normalized.Code {
		delete(s.dictTypeCodes, old.Code)
	}
	s.dictTypes[normalized.ID] = *normalized
	s.dictTypeCodes[normalized.Code] = normalized.ID
	return nil
}

func (s *SystemStore) DeleteDictionaryType(_ context.Context, id shared.ID) error {
	if id.IsZero() {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.dictTypes[id]
	if !ok {
		return nil
	}
	delete(s.dictTypes, id)
	delete(s.dictTypeCodes, item.Code)
	for itemID, dictItem := range s.dictItems {
		if dictItem.TypeCode == item.Code {
			delete(s.dictItems, itemID)
			delete(s.dictItemKeys, dictionaryItemKey(dictItem.TypeCode, dictItem.Value))
		}
	}
	return nil
}

func (s *SystemStore) GetDictionaryItemByID(_ context.Context, id shared.ID) (*system.DictionaryItem, error) {
	if id.IsZero() {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.dictItems[id]
	if !ok {
		return nil, nil
	}
	return cloneDictionaryItem(item), nil
}

func (s *SystemStore) GetDictionaryItemByTypeAndValue(_ context.Context, typeCode, value string) (*system.DictionaryItem, error) {
	key := dictionaryItemKey(typeCode, value)
	if key == "" {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.dictItemKeys[key]
	if !ok {
		return nil, nil
	}
	item := s.dictItems[id]
	return cloneDictionaryItem(item), nil
}

func (s *SystemStore) ListDictionaryItems(_ context.Context, typeCode string, offset, limit int) ([]system.DictionaryItem, error) {
	typeCode = normalizeDictionaryCode(typeCode)
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]system.DictionaryItem, 0, len(s.dictItems))
	for _, item := range s.dictItems {
		if typeCode == "" || item.TypeCode == typeCode {
			out = append(out, item)
		}
	}
	out = system.SortDictionaryItems(out)
	return paginateDictionaryItems(out, offset, limit), nil
}

func (s *SystemStore) SaveDictionaryItem(_ context.Context, item *system.DictionaryItem) error {
	if item == nil {
		return fmt.Errorf("dictionary item is required")
	}
	normalized, err := system.NewDictionaryItem(system.DictionaryItemInput{
		ID:        item.ID,
		TypeCode:  item.TypeCode,
		Label:     item.Label,
		Value:     item.Value,
		Status:    item.Status,
		Sort:      item.Sort,
		Builtin:   item.Builtin,
		CreatedAt: item.Meta.CreatedAt,
		UpdatedAt: item.Meta.UpdatedAt,
	})
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.dictTypeCodes[normalized.TypeCode]; !ok {
		return fmt.Errorf("dictionary type does not exist")
	}
	key := dictionaryItemKey(normalized.TypeCode, normalized.Value)
	if existingID, ok := s.dictItemKeys[key]; ok && existingID != normalized.ID {
		return fmt.Errorf("dictionary item value already exists")
	}
	if old, ok := s.dictItems[normalized.ID]; ok {
		oldKey := dictionaryItemKey(old.TypeCode, old.Value)
		if oldKey != key {
			delete(s.dictItemKeys, oldKey)
		}
	}
	s.dictItems[normalized.ID] = *normalized
	s.dictItemKeys[key] = normalized.ID
	return nil
}

func (s *SystemStore) DeleteDictionaryItem(_ context.Context, id shared.ID) error {
	if id.IsZero() {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.dictItems[id]
	if !ok {
		return nil
	}
	delete(s.dictItems, id)
	delete(s.dictItemKeys, dictionaryItemKey(item.TypeCode, item.Value))
	return nil
}

func normalizeDictionaryCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func dictionaryItemKey(typeCode, value string) string {
	typeCode = normalizeDictionaryCode(typeCode)
	value = strings.TrimSpace(value)
	if typeCode == "" || value == "" {
		return ""
	}
	return typeCode + "\x00" + value
}

func cloneDictionaryType(item system.DictionaryType) *system.DictionaryType {
	clone := item
	return &clone
}

func cloneDictionaryItem(item system.DictionaryItem) *system.DictionaryItem {
	clone := item
	return &clone
}

func paginateDictionaryTypes(items []system.DictionaryType, offset, limit int) []system.DictionaryType {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = len(items)
	}
	if offset > len(items) {
		return []system.DictionaryType{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return append([]system.DictionaryType(nil), items[offset:end]...)
}

func paginateDictionaryItems(items []system.DictionaryItem, offset, limit int) []system.DictionaryItem {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = len(items)
	}
	if offset > len(items) {
		return []system.DictionaryItem{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return append([]system.DictionaryItem(nil), items[offset:end]...)
}
