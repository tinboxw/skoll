package user

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/audit"
	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainuser "github.com/tinboxw/skoll/internal/domain/user"
	"github.com/tinboxw/skoll/internal/repository"
	auditrepo "github.com/tinboxw/skoll/internal/repository/audit"
	userrepo "github.com/tinboxw/skoll/internal/repository/user"
	servicecommon "github.com/tinboxw/skoll/internal/service/common"
	rbacservice "github.com/tinboxw/skoll/internal/service/rbac"
)

type DataScopeResolver interface {
	ResolveDataScope(ctx context.Context, in rbacservice.ResolveDataScopeInput) (rbacservice.DataScopeDecision, error)
}

type serviceImpl struct {
	repo              userrepo.UserRepository
	audit             auditrepo.AuditRepository
	tx                *servicecommon.TransactionManager
	dataScopeResolver DataScopeResolver
	nowFn             func() time.Time
	idFn              func(prefix string) shared.ID
}

var sha256HexPattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

func NewService(repo userrepo.UserRepository, auditRepo auditrepo.AuditRepository, tx repository.UnitOfWork) Service {
	return NewServiceWithDataScope(repo, auditRepo, tx, nil)
}

func NewServiceWithDataScope(repo userrepo.UserRepository, auditRepo auditrepo.AuditRepository, tx repository.UnitOfWork, resolver DataScopeResolver) Service {
	return &serviceImpl{
		repo:              repo,
		audit:             auditRepo,
		tx:                servicecommon.NewTransactionManager(tx),
		dataScopeResolver: resolver,
		nowFn:             func() time.Time { return time.Now().UTC() },
		idFn: func(prefix string) shared.ID {
			if prefix == "audit" {
				return shared.ID(strconv.FormatInt(time.Now().UTC().UnixNano(), 10))
			}
			return shared.ID("new")
		},
	}
}

func (s *serviceImpl) Create(ctx context.Context, in CreateUserInput) (*domainuser.User, error) {
	if err := validateCreateInput(in); err != nil {
		return nil, err
	}

	now := s.nowFn()
	entity, err := domainuser.New(s.idFn("user"), in.Account, in.Name, in.Email, now)
	if err != nil {
		return nil, err
	}
	passwordValue := strings.TrimSpace(in.PasswordHash)
	if !strings.HasPrefix(strings.ToLower(passwordValue), "sha256:") && !sha256HexPattern.MatchString(passwordValue) {
		hash, err := domainuser.HashPassword(passwordValue)
		if err != nil {
			return nil, err
		}
		passwordValue = hash.String()
	}
	if err := entity.SetPasswordHash(passwordValue); err != nil {
		return nil, err
	}
	entity.SetOrganization(in.DepartmentID, in.PositionID, now)

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
				map[string]any{"account": entity.Account},
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

func (s *serviceImpl) CreateBatch(ctx context.Context, in BatchCreateInput) ([]BatchCreateResult, error) {
	if len(in.Items) == 0 {
		return []BatchCreateResult{}, nil
	}

	results := make([]BatchCreateResult, 0, len(in.Items))
	if !in.Atomic {
		for idx, item := range in.Items {
			entity, err := s.Create(ctx, item)
			account := strings.TrimSpace(item.Account)
			if err != nil {
				results = append(results, BatchCreateResult{Index: idx, Account: account, Success: false, Message: err.Error()})
				continue
			}
			createdID := ""
			if entity != nil {
				createdID = entity.ID.String()
			}
			results = append(results, BatchCreateResult{Index: idx, Account: account, Success: true, Message: "ok", ID: createdID})
		}
		return results, nil
	}

	err := s.tx.InTx(ctx, func(_ repository.Tx) error {
		tmp := make([]BatchCreateResult, 0, len(in.Items))
		for idx, item := range in.Items {
			account := strings.TrimSpace(item.Account)
			entity, err := s.createWithoutTx(ctx, item)
			if err != nil {
				tmp = append(tmp, BatchCreateResult{Index: idx, Account: account, Success: false, Message: err.Error()})
				results = tmp
				return err
			}
			createdID := ""
			if entity != nil {
				createdID = entity.ID.String()
			}
			tmp = append(tmp, BatchCreateResult{Index: idx, Account: account, Success: true, Message: "ok", ID: createdID})
		}
		results = tmp
		return nil
	})
	if err != nil {
		if len(results) == 0 {
			results = []BatchCreateResult{{Index: 0, Account: "", Success: false, Message: err.Error()}}
		}
		return results, nil
	}

	return results, nil
}

func (s *serviceImpl) createWithoutTx(ctx context.Context, in CreateUserInput) (*domainuser.User, error) {
	if err := validateCreateInput(in); err != nil {
		return nil, err
	}

	now := s.nowFn()
	entity, err := domainuser.New(s.idFn("user"), in.Account, in.Name, in.Email, now)
	if err != nil {
		return nil, err
	}
	passwordValue := strings.TrimSpace(in.PasswordHash)
	if !strings.HasPrefix(strings.ToLower(passwordValue), "sha256:") && !sha256HexPattern.MatchString(passwordValue) {
		hash, err := domainuser.HashPassword(passwordValue)
		if err != nil {
			return nil, err
		}
		passwordValue = hash.String()
	}
	if err := entity.SetPasswordHash(passwordValue); err != nil {
		return nil, err
	}
	entity.SetOrganization(in.DepartmentID, in.PositionID, now)

	if err := s.repo.Save(ctx, entity); err != nil {
		return nil, err
	}
	if s.audit != nil {
		rec, recErr := audit.NewRecord(
			s.idFn("audit"),
			shared.ID(in.ActorID),
			"create",
			"user",
			entity.ID.String(),
			map[string]any{"account": entity.Account},
			now,
		)
		if recErr != nil {
			return nil, recErr
		}
		if err := s.audit.Append(ctx, rec); err != nil {
			return nil, err
		}
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
	if in.SuperAdmin {
		return s.repo.List(ctx, in.Offset, in.Limit)
	}

	scope := domainrbac.NormalizeDataScope(in.DataScope)
	if scope == "" {
		return s.repo.List(ctx, in.Offset, in.Limit)
	}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if scope == domainrbac.DataScopeAll {
		return s.repo.List(ctx, in.Offset, in.Limit)
	}
	if s.dataScopeResolver == nil {
		return nil, fmt.Errorf("data scope resolver is required")
	}

	decision, err := s.dataScopeResolver.ResolveDataScope(ctx, rbacservice.ResolveDataScopeInput{
		Scope:               scope,
		ActorUserID:         in.ActorUserID,
		ActorDepartmentID:   in.ActorDepartmentID,
		DepartmentTreeIDs:   in.DepartmentTreeIDs,
		CustomDepartmentIDs: in.CustomDepartmentIDs,
	})
	if err != nil {
		return nil, err
	}
	if decision.All {
		return s.repo.List(ctx, in.Offset, in.Limit)
	}

	filter := userrepo.ListFilter{
		UserIDs:       make([]shared.ID, 0, len(decision.UserIDs)),
		DepartmentIDs: decision.DepartmentIDs,
	}
	for _, id := range decision.UserIDs {
		filter.UserIDs = append(filter.UserIDs, shared.ID(id))
	}
	if filter.Empty() {
		return []*domainuser.User{}, nil
	}
	return s.repo.ListFiltered(ctx, filter, in.Offset, in.Limit)
}

func (s *serviceImpl) Update(ctx context.Context, in UpdateUserInput) (*domainuser.User, error) {
	if strings.TrimSpace(in.ID) == "" {
		return nil, fmt.Errorf("id is required")
	}
	now := s.nowFn()
	entity, err := s.repo.GetByID(ctx, shared.ID(strings.TrimSpace(in.ID)))
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, fmt.Errorf("user not found")
	}

	if name := strings.TrimSpace(in.Name); name != "" && name != entity.Name {
		if err := entity.Rename(name, now); err != nil {
			return nil, err
		}
	}
	if email := strings.TrimSpace(in.Email); email != "" && email != entity.Email.String() {
		if err := entity.ChangeEmail(email, now); err != nil {
			return nil, err
		}
	}
	if status := strings.TrimSpace(strings.ToLower(in.Status)); status != "" {
		switch domainuser.Status(status) {
		case domainuser.StatusActive:
			entity.Activate(now)
		case domainuser.StatusDisabled:
			entity.Disable(now)
		default:
			return nil, fmt.Errorf("unsupported status: %s", status)
		}
	}
	if strings.TrimSpace(in.DepartmentID) != entity.DepartmentID || strings.TrimSpace(in.PositionID) != entity.PositionID {
		entity.SetOrganization(in.DepartmentID, in.PositionID, now)
	}

	if err := s.repo.Save(ctx, entity); err != nil {
		return nil, err
	}
	return entity, nil
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

func (s *serviceImpl) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id is required")
	}
	entity, err := s.repo.GetByID(ctx, shared.ID(id))
	if err != nil {
		return err
	}
	if entity == nil {
		return fmt.Errorf("user not found")
	}
	return s.repo.Delete(ctx, shared.ID(id))
}
