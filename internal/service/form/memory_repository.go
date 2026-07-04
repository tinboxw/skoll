package form

import (
	"context"
	"fmt"
	"strings"
	"sync"

	domainform "github.com/tinboxw/skoll/internal/domain/form"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type MemoryRepository struct {
	mu    sync.RWMutex
	byID  map[shared.ID]domainform.Schema
	byKey map[string]shared.ID
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byID:  map[shared.ID]domainform.Schema{},
		byKey: map[string]shared.ID{},
	}
}

func (r *MemoryRepository) SaveSchema(_ context.Context, schema domainform.Schema) error {
	if r == nil {
		return fmt.Errorf("form schema repository is required")
	}
	if err := schema.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[schema.ID] = cloneSchema(schema)
	r.byKey[schemaKey(schema.Key, schema.Version)] = schema.ID
	return nil
}

func (r *MemoryRepository) GetSchema(_ context.Context, id shared.ID) (*domainform.Schema, error) {
	if r == nil {
		return nil, fmt.Errorf("form schema repository is required")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	schema, ok := r.byID[id]
	if !ok {
		return nil, fmt.Errorf("form schema not found")
	}
	schema = cloneSchema(schema)
	return &schema, nil
}

func (r *MemoryRepository) GetSchemaByKey(_ context.Context, key string, version int) (*domainform.Schema, error) {
	if r == nil {
		return nil, fmt.Errorf("form schema repository is required")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byKey[schemaKey(key, version)]
	if !ok {
		return nil, fmt.Errorf("form schema not found")
	}
	schema := cloneSchema(r.byID[id])
	return &schema, nil
}

func schemaKey(key string, version int) string {
	return strings.TrimSpace(strings.ToLower(key)) + "#" + fmt.Sprint(version)
}

func cloneSchema(schema domainform.Schema) domainform.Schema {
	schema.Fields = cloneFields(schema.Fields)
	return schema
}

func cloneFields(fields []domainform.Field) []domainform.Field {
	out := append([]domainform.Field(nil), fields...)
	for idx := range out {
		out[idx].Options = append([]domainform.Option(nil), out[idx].Options...)
		out[idx].Validation = append([]domainform.ValidationRule(nil), out[idx].Validation...)
		if out[idx].Attachment != nil {
			cfg := *out[idx].Attachment
			cfg.Accept = append([]string(nil), cfg.Accept...)
			out[idx].Attachment = &cfg
		}
		if out[idx].DetailTable != nil {
			cfg := *out[idx].DetailTable
			cfg.Columns = cloneFields(cfg.Columns)
			out[idx].DetailTable = &cfg
		}
	}
	return out
}
