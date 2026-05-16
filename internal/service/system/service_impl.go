package system

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
	systemrepo "github.com/tinboxw/skoll/internal/repository/system"
)

type serviceImpl struct {
	repo  systemrepo.SystemRepository
	nowFn func() time.Time
	idFn  func(prefix string) shared.ID
}

func NewService(repo systemrepo.SystemRepository) Service {
	return &serviceImpl{
		repo:  repo,
		nowFn: func() time.Time { return time.Now().UTC() },
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
