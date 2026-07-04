package form

import (
	"context"
	"fmt"
	"strings"

	domainform "github.com/tinboxw/skoll/internal/domain/form"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type serviceImpl struct {
	repo Repository
}

func NewService(repo Repository) Service {
	if repo == nil {
		repo = NewMemoryRepository()
	}
	return &serviceImpl{repo: repo}
}

func (s *serviceImpl) CreateSchema(ctx context.Context, in CreateSchemaInput) (*domainform.Schema, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("form schema repository is required")
	}
	schema, err := domainform.NewSchema(domainform.SchemaInput{
		ID:           in.ID,
		Key:          in.Key,
		Name:         in.Name,
		Version:      in.Version,
		BusinessType: in.BusinessType,
		Description:  in.Description,
		Fields:       in.Fields,
		Now:          in.Now,
	})
	if err != nil {
		return nil, err
	}
	if err := s.repo.SaveSchema(ctx, *schema); err != nil {
		return nil, err
	}
	return schema, nil
}

func (s *serviceImpl) GetSchema(ctx context.Context, id shared.ID) (*domainform.Schema, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("form schema repository is required")
	}
	if id.IsZero() {
		return nil, fmt.Errorf("form schema id is required")
	}
	return s.repo.GetSchema(ctx, id)
}

func (s *serviceImpl) GetSchemaByKey(ctx context.Context, key string, version int) (*domainform.Schema, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("form schema repository is required")
	}
	if strings.TrimSpace(key) == "" || version <= 0 {
		return nil, fmt.Errorf("form schema key and version are required")
	}
	return s.repo.GetSchemaByKey(ctx, key, version)
}
