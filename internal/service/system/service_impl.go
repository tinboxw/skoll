package system

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tinboxw/skoll/internal/cache"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
	systemrepo "github.com/tinboxw/skoll/internal/repository/system"
)

const defaultDictionaryCacheTTL = 5 * time.Minute

type serviceImpl struct {
	repo             systemrepo.SystemRepository
	nowFn            func() time.Time
	idFn             func(prefix string) shared.ID
	dictionaryCache  cache.BytesCache
	dictionaryTTL    time.Duration
	cacheMu          sync.Mutex
	dictListKeys     map[string]struct{}
	dictItemKeys     map[string]struct{}
	dictItemListKeys map[string]struct{}
}

func NewService(repo systemrepo.SystemRepository) Service {
	return newServiceImpl(repo)
}

func newServiceImpl(repo systemrepo.SystemRepository) *serviceImpl {
	return &serviceImpl{
		repo:             repo,
		dictionaryTTL:    defaultDictionaryCacheTTL,
		dictListKeys:     map[string]struct{}{},
		dictItemKeys:     map[string]struct{}{},
		dictItemListKeys: map[string]struct{}{},
		nowFn:            func() time.Time { return time.Now().UTC() },
		idFn: func(prefix string) shared.ID {
			return shared.ID("new")
		},
	}
}

func (s *serviceImpl) Upsert(ctx context.Context, in UpsertInput) (*domainsystem.Setting, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("system repository is not configured")
	}
	key := strings.TrimSpace(in.Key)
	if key == "" {
		return nil, fmt.Errorf("setting key is required")
	}

	now := s.nowFn()
	existing, err := s.repo.GetSettingByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		existing.UpdateValue(in.Value, now)
		existing.Encrypted = in.Encrypted
		if err := s.repo.SaveSetting(ctx, existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	created, err := domainsystem.NewSetting(s.idFn("setting"), key, in.Value, in.Encrypted, now)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SaveSetting(ctx, created); err != nil {
		return nil, err
	}
	return created, nil
}

func (s *serviceImpl) GetByKey(ctx context.Context, key string) (*domainsystem.Setting, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("system repository is not configured")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("setting key is required")
	}
	return s.repo.GetSettingByKey(ctx, key)
}

func (s *serviceImpl) List(ctx context.Context, in ListInput) ([]*domainsystem.Setting, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("system repository is not configured")
	}
	if in.Offset < 0 || in.Limit < 0 {
		return nil, fmt.Errorf("invalid pagination")
	}
	return s.repo.ListSettings(ctx, in.Offset, in.Limit)
}

func (s *serviceImpl) Reset(ctx context.Context) (int, error) {
	if s == nil || s.repo == nil {
		return 0, fmt.Errorf("system repository is not configured")
	}
	items, err := s.repo.ListSettings(ctx, 0, 0)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, item := range items {
		if item == nil || item.ID.IsZero() {
			continue
		}
		if err := s.repo.DeleteSetting(ctx, item.ID); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}

func (s *serviceImpl) SaveDictionaryType(ctx context.Context, in DictionaryTypeInput) (*domainsystem.DictionaryType, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	now := s.nowFn()
	id := shared.ID(strings.TrimSpace(in.ID))
	if id.IsZero() {
		id = s.idFn("dict_type")
	}
	item, err := domainsystem.NewDictionaryType(domainsystem.DictionaryTypeInput{
		ID:          id,
		Code:        in.Code,
		Name:        in.Name,
		Description: in.Description,
		Status:      domainsystem.DictionaryStatus(strings.TrimSpace(in.Status)),
		Sort:        in.Sort,
		Builtin:     in.Builtin,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return nil, err
	}
	var oldTypeCode string
	if existing, err := s.repo.GetDictionaryTypeByID(ctx, item.ID); err != nil {
		return nil, err
	} else if existing != nil && existing.Meta.CreatedAt.IsZero() == false {
		item.Meta.CreatedAt = existing.Meta.CreatedAt
		oldTypeCode = existing.Code
	}
	if err := s.repo.SaveDictionaryType(ctx, item); err != nil {
		return nil, err
	}
	if oldTypeCode != "" && oldTypeCode != item.Code {
		s.invalidateDictionaryType(oldTypeCode)
		s.invalidateDictionaryItemsForType(oldTypeCode)
	}
	s.invalidateDictionaryType(item.Code)
	return item, nil
}

func (s *serviceImpl) GetDictionaryTypeByCode(ctx context.Context, code string) (*domainsystem.DictionaryType, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	code = normalizeDictionaryCode(code)
	if code == "" {
		return nil, fmt.Errorf("dictionary type code is required")
	}
	key := dictionaryTypeCacheKey(code)
	var cached domainsystem.DictionaryType
	if s.cacheGet(key, &cached) {
		return &cached, nil
	}
	item, err := s.repo.GetDictionaryTypeByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if item != nil {
		s.cacheSet(key, item)
	}
	return item, nil
}

func (s *serviceImpl) ListDictionaryTypes(ctx context.Context, in DictionaryTypeListInput) ([]domainsystem.DictionaryType, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	if in.Offset < 0 || in.Limit < 0 {
		return nil, fmt.Errorf("invalid pagination")
	}
	key := dictionaryTypeListCacheKey(in.Offset, in.Limit)
	var cached []domainsystem.DictionaryType
	if s.cacheGet(key, &cached) {
		return cached, nil
	}
	items, err := s.repo.ListDictionaryTypes(ctx, in.Offset, in.Limit)
	if err != nil {
		return nil, err
	}
	s.cacheSet(key, items)
	s.trackDictListKey(key)
	return items, nil
}

func (s *serviceImpl) DeleteDictionaryType(ctx context.Context, id string) error {
	if err := s.requireRepo(); err != nil {
		return err
	}
	dictID := shared.ID(strings.TrimSpace(id))
	if dictID.IsZero() {
		return fmt.Errorf("dictionary type id is required")
	}
	existing, err := s.repo.GetDictionaryTypeByID(ctx, dictID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteDictionaryType(ctx, dictID); err != nil {
		return err
	}
	if existing != nil {
		s.invalidateDictionaryType(existing.Code)
	} else {
		s.invalidateDictionaryLists()
	}
	s.invalidateDictionaryItemsForType("")
	return nil
}

func (s *serviceImpl) SaveDictionaryItem(ctx context.Context, in DictionaryItemInput) (*domainsystem.DictionaryItem, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	now := s.nowFn()
	id := shared.ID(strings.TrimSpace(in.ID))
	if id.IsZero() {
		id = s.idFn("dict_item")
	}
	item, err := domainsystem.NewDictionaryItem(domainsystem.DictionaryItemInput{
		ID:        id,
		TypeCode:  in.TypeCode,
		Label:     in.Label,
		Value:     in.Value,
		Status:    domainsystem.DictionaryStatus(strings.TrimSpace(in.Status)),
		Sort:      in.Sort,
		Builtin:   in.Builtin,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return nil, err
	}
	var oldItemTypeCode string
	if existing, err := s.repo.GetDictionaryItemByID(ctx, item.ID); err != nil {
		return nil, err
	} else if existing != nil {
		item.Meta.CreatedAt = existing.Meta.CreatedAt
		oldItemTypeCode = existing.TypeCode
		s.cacheDelete(dictionaryItemCacheKey(existing.TypeCode, existing.Value))
	}
	if err := s.repo.SaveDictionaryItem(ctx, item); err != nil {
		return nil, err
	}
	if oldItemTypeCode != "" && oldItemTypeCode != item.TypeCode {
		s.invalidateDictionaryItemsForType(oldItemTypeCode)
	}
	s.invalidateDictionaryItemsForType(item.TypeCode)
	return item, nil
}

func (s *serviceImpl) GetDictionaryItemByTypeAndValue(ctx context.Context, typeCode, value string) (*domainsystem.DictionaryItem, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	typeCode = normalizeDictionaryCode(typeCode)
	value = strings.TrimSpace(value)
	if typeCode == "" || value == "" {
		return nil, fmt.Errorf("dictionary item type code and value are required")
	}
	key := dictionaryItemCacheKey(typeCode, value)
	var cached domainsystem.DictionaryItem
	if s.cacheGet(key, &cached) {
		return &cached, nil
	}
	item, err := s.repo.GetDictionaryItemByTypeAndValue(ctx, typeCode, value)
	if err != nil {
		return nil, err
	}
	if item != nil {
		s.cacheSet(key, item)
		s.trackDictItemKey(key)
	}
	return item, nil
}

func (s *serviceImpl) ListDictionaryItems(ctx context.Context, in DictionaryItemListInput) ([]domainsystem.DictionaryItem, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	if in.Offset < 0 || in.Limit < 0 {
		return nil, fmt.Errorf("invalid pagination")
	}
	typeCode := normalizeDictionaryCode(in.TypeCode)
	if typeCode == "" {
		return nil, fmt.Errorf("dictionary type code is required")
	}
	key := dictionaryItemListCacheKey(typeCode, in.Offset, in.Limit)
	var cached []domainsystem.DictionaryItem
	if s.cacheGet(key, &cached) {
		return cached, nil
	}
	items, err := s.repo.ListDictionaryItems(ctx, typeCode, in.Offset, in.Limit)
	if err != nil {
		return nil, err
	}
	s.cacheSet(key, items)
	s.trackDictItemListKey(key)
	return items, nil
}

func (s *serviceImpl) DeleteDictionaryItem(ctx context.Context, id string) error {
	if err := s.requireRepo(); err != nil {
		return err
	}
	itemID := shared.ID(strings.TrimSpace(id))
	if itemID.IsZero() {
		return fmt.Errorf("dictionary item id is required")
	}
	existing, err := s.repo.GetDictionaryItemByID(ctx, itemID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteDictionaryItem(ctx, itemID); err != nil {
		return err
	}
	if existing != nil {
		s.invalidateDictionaryItemsForType(existing.TypeCode)
	} else {
		s.invalidateDictionaryItemsForType("")
	}
	return nil
}

func (s *serviceImpl) requireRepo() error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("system repository is not configured")
	}
	return nil
}

func (s *serviceImpl) cacheGet(key string, out any) bool {
	if s == nil || s.dictionaryCache == nil {
		return false
	}
	data, ok := s.dictionaryCache.Get(key)
	if !ok {
		return false
	}
	return json.Unmarshal(data, out) == nil
}

func (s *serviceImpl) cacheSet(key string, value any) {
	if s == nil || s.dictionaryCache == nil {
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	s.dictionaryCache.Set(key, data, s.dictionaryTTL)
}

func (s *serviceImpl) cacheDelete(key string) {
	if s == nil || s.dictionaryCache == nil || key == "" {
		return
	}
	s.dictionaryCache.Delete(key)
}

func (s *serviceImpl) trackDictListKey(key string) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.dictListKeys[key] = struct{}{}
}

func (s *serviceImpl) trackDictItemKey(key string) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.dictItemKeys[key] = struct{}{}
}

func (s *serviceImpl) trackDictItemListKey(key string) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.dictItemListKeys[key] = struct{}{}
}

func (s *serviceImpl) invalidateDictionaryType(code string) {
	s.cacheDelete(dictionaryTypeCacheKey(code))
	s.invalidateDictionaryLists()
}

func (s *serviceImpl) invalidateDictionaryLists() {
	s.cacheMu.Lock()
	keys := make([]string, 0, len(s.dictListKeys))
	for key := range s.dictListKeys {
		keys = append(keys, key)
		delete(s.dictListKeys, key)
	}
	s.cacheMu.Unlock()
	for _, key := range keys {
		s.cacheDelete(key)
	}
}

func (s *serviceImpl) invalidateDictionaryItemsForType(typeCode string) {
	typeCode = normalizeDictionaryCode(typeCode)
	s.cacheMu.Lock()
	itemKeys := make([]string, 0, len(s.dictItemKeys))
	for key := range s.dictItemKeys {
		if typeCode == "" || strings.Contains(key, ":"+typeCode+":") {
			itemKeys = append(itemKeys, key)
			delete(s.dictItemKeys, key)
		}
	}
	listKeys := make([]string, 0, len(s.dictItemListKeys))
	for key := range s.dictItemListKeys {
		if typeCode == "" || strings.Contains(key, ":"+typeCode+":") {
			listKeys = append(listKeys, key)
			delete(s.dictItemListKeys, key)
		}
	}
	s.cacheMu.Unlock()
	for _, key := range itemKeys {
		s.cacheDelete(key)
	}
	for _, key := range listKeys {
		s.cacheDelete(key)
	}
}

func normalizeDictionaryCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func dictionaryTypeCacheKey(code string) string {
	return "dictionary:type:code:" + normalizeDictionaryCode(code)
}

func dictionaryTypeListCacheKey(offset, limit int) string {
	return fmt.Sprintf("dictionary:types:list:%d:%d", offset, limit)
}

func dictionaryItemCacheKey(typeCode, value string) string {
	return "dictionary:item:" + normalizeDictionaryCode(typeCode) + ":" + strings.TrimSpace(value)
}

func dictionaryItemListCacheKey(typeCode string, offset, limit int) string {
	return fmt.Sprintf("dictionary:items:list:%s:%d:%d", normalizeDictionaryCode(typeCode), offset, limit)
}
