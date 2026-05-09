package user

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainuser "github.com/tinboxw/skoll/internal/domain/user"
	"github.com/tinboxw/skoll/internal/repository"
	servicecommon "github.com/tinboxw/skoll/internal/service/common"
)

type serviceImpl struct {
	repo  repository.UserRepository
	audit repository.AuditRepository
	tx    *servicecommon.TransactionManager
	nowFn func() time.Time
	idFn  func(prefix string) shared.ID
}

func NewService(repo repository.UserRepository, auditRepo repository.AuditRepository, tx repository.UnitOfWork) Service {
	return &serviceImpl{
		repo:  repo,
		audit: auditRepo,
		tx:    servicecommon.NewTransactionManager(tx),
		nowFn: func() time.Time { return time.Now().UTC() },
		idFn: func(prefix string) shared.ID {
			return shared.ID(prefix + "-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10))
		},
	}
}

func (s *serviceImpl) Create(ctx context.Context, in CreateUserInput) (*domainuser.User, error) {
	if err := validateCreateInput(in); err != nil {
		return nil, err
	}

	now := s.nowFn()
	entity, err := domainuser.New(s.idFn("user"), in.Username, in.DisplayName, in.Email, now)
	if err != nil {
		return nil, err
	}
	if err := entity.SetPasswordHash(in.PasswordHash); err != nil {
		return nil, err
	}

	err = s.tx.InTx(ctx, func(_ repository.Tx) error {
		if err := s.repo.Save(ctx, entity); err != nil {
			return err
		}
		if s.audit != nil {
			rec, recErr := audit.NewRecord(
				s.idFn("audit"),
				shared.ID(in.ActorID),
				"create",
				"user",
				entity.ID.String(),
				map[string]any{"username": entity.Username},
				now,
			)
			if recErr != nil {
				return recErr
			}
			if err := s.audit.Append(ctx, rec); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *serviceImpl) Get(ctx context.Context, id string) (*domainuser.User, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	return s.repo.GetByID(ctx, shared.ID(id))
}

func (s *serviceImpl) List(ctx context.Context, in ListInput) ([]*domainuser.User, error) {
	if err := validateListInput(in); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, in.Offset, in.Limit)
}

func (s *serviceImpl) UpdateEmail(ctx context.Context, in UpdateEmailInput) (*domainuser.User, error) {
	if in.ID == "" {
		return nil, fmt.Errorf("id is required")
	}
	now := s.nowFn()
	entity, err := s.repo.GetByID(ctx, shared.ID(in.ID))
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, fmt.Errorf("user not found")
	}
	if err := entity.ChangeEmail(in.Email, now); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, entity); err != nil {
		return nil, err
	}
	return entity, nil
}

func (s *serviceImpl) Disable(ctx context.Context, id, actorID string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}
	now := s.nowFn()
	entity, err := s.repo.GetByID(ctx, shared.ID(id))
	if err != nil {
		return err
	}
	if entity == nil {
		return fmt.Errorf("user not found")
	}
	entity.Disable(now)
	if err := s.repo.Save(ctx, entity); err != nil {
		return err
	}
	if s.audit != nil {
		rec, recErr := audit.NewRecord(s.idFn("audit"), shared.ID(actorID), "disable", "user", entity.ID.String(), nil, now)
		if recErr == nil {
			_ = s.audit.Append(ctx, rec)
		}
	}
	return nil
}
