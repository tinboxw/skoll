package pharmaoa

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
)

type EmployeeService interface {
	Create(ctx context.Context, in EmployeeWriteInput) (*domainpharma.Employee, error)
	List(ctx context.Context, in EmployeeListInput) ([]*domainpharma.Employee, error)
	ListPage(ctx context.Context, in EmployeeListInput) (ListPage[*domainpharma.Employee], error)
	Update(ctx context.Context, id string, in EmployeeWriteInput) (*domainpharma.Employee, error)
	MarkLeft(ctx context.Context, id string, in EmployeeLeaveInput) (*domainpharma.Employee, error)
	QualificationReminders(ctx context.Context, days int) ([]EmployeeQualificationReminder, error)
}

type EmployeeWriteInput struct {
	Code         string
	Name         string
	DepartmentID string
	PositionID   string
	Phone        string
	Email        string
	Certificates []domainpharma.EmployeeCertificate
	ActorID      string
}

type EmployeeListInput struct {
	Keyword string
	Status  string
	Offset  int
	Limit   int
}

type EmployeeLeaveInput struct {
	Reason  string
	ActorID string
}

type EmployeeQualificationReminder struct {
	EmployeeID   string    `json:"employeeId"`
	EmployeeCode string    `json:"employeeCode"`
	EmployeeName string    `json:"employeeName"`
	Certificate  string    `json:"certificate"`
	Number       string    `json:"number"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

type employeeService struct {
	repo      pharmaoarepo.EmployeeRepository
	audit     auditsvc.Service
	nowFn     func() time.Time
	idCounter atomic.Int64
}

func NewEmployeeService(audit auditsvc.Service, repositories ...pharmaoarepo.EmployeeRepository) EmployeeService {
	repo := pharmaoarepo.EmployeeRepository(pharmaoarepo.NewMemoryEmployeeRepository())
	if len(repositories) > 0 && repositories[0] != nil {
		repo = repositories[0]
	}
	return &employeeService{
		repo:  repo,
		audit: audit,
		nowFn: func() time.Time { return time.Now().UTC() },
	}
}

func (s *employeeService) Create(ctx context.Context, in EmployeeWriteInput) (*domainpharma.Employee, error) {
	now := s.nowFn()
	id, err := s.nextAvailableID(ctx)
	if err != nil {
		return nil, err
	}
	entity, err := domainpharma.NewEmployee(id, toDomainInput(in), now)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetByCode(ctx, entity.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("employee code already exists")
	}
	if err := s.repo.Create(ctx, entity); err != nil {
		return nil, err
	}
	s.appendAudit(ctx, in.ActorID, "pharma_oa.employee.create", entity.ID.String(), map[string]any{"code": entity.Code})
	return cloneEmployee(entity), nil
}

func (s *employeeService) List(ctx context.Context, in EmployeeListInput) ([]*domainpharma.Employee, error) {
	page, err := s.ListPage(ctx, in)
	return page.Items, err
}

func (s *employeeService) ListPage(ctx context.Context, in EmployeeListInput) (ListPage[*domainpharma.Employee], error) {
	page, err := s.repo.ListPage(ctx, pharmaoarepo.ListFilter{Keyword: in.Keyword, Status: in.Status, Offset: in.Offset, Limit: normalizeLimit(in.Limit)})
	if err != nil {
		return ListPage[*domainpharma.Employee]{}, err
	}
	items := make([]*domainpharma.Employee, 0, len(page.Items))
	for index := range page.Items {
		items = append(items, cloneEmployee(&page.Items[index]))
	}
	return ListPage[*domainpharma.Employee]{Items: items, Total: page.Total}, nil
}

func (s *employeeService) Update(ctx context.Context, id string, in EmployeeWriteInput) (*domainpharma.Employee, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	current, err := s.repo.Get(ctx, shared.ID(id))
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("employee not found")
	}
	existing, err := s.repo.GetByCode(ctx, in.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.ID.String() != id {
		return nil, fmt.Errorf("employee code already exists")
	}
	next := cloneEmployee(current)
	if err := next.Update(toDomainInput(in), s.nowFn()); err != nil {
		return nil, err
	}
	if err := s.repo.Upsert(ctx, next); err != nil {
		return nil, err
	}
	s.appendAudit(ctx, in.ActorID, "pharma_oa.employee.update", id, map[string]any{"code": next.Code})
	return cloneEmployee(next), nil
}

func (s *employeeService) MarkLeft(ctx context.Context, id string, in EmployeeLeaveInput) (*domainpharma.Employee, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	current, err := s.repo.Get(ctx, shared.ID(id))
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("employee not found")
	}
	next := cloneEmployee(current)
	if err := next.MarkLeft(in.Reason, s.nowFn()); err != nil {
		return nil, err
	}
	if err := s.repo.Upsert(ctx, next); err != nil {
		return nil, err
	}
	s.appendAudit(ctx, in.ActorID, "pharma_oa.employee.leave", id, map[string]any{"reason": strings.TrimSpace(in.Reason)})
	return cloneEmployee(next), nil
}

func (s *employeeService) QualificationReminders(ctx context.Context, days int) ([]EmployeeQualificationReminder, error) {
	if days <= 0 {
		days = 30
	}
	deadline := s.nowFn().AddDate(0, 0, days)
	items, err := s.repo.List(ctx, pharmaoarepo.ListFilter{})
	if err != nil {
		return nil, err
	}
	out := make([]EmployeeQualificationReminder, 0)
	for index := range items {
		item := &items[index]
		for _, cert := range item.CertificateExpiringBefore(deadline) {
			out = append(out, EmployeeQualificationReminder{
				EmployeeID:   item.ID.String(),
				EmployeeCode: item.Code,
				EmployeeName: item.Name,
				Certificate:  cert.Name,
				Number:       cert.Number,
				ExpiresAt:    cert.ExpiresAt,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ExpiresAt.Before(out[j].ExpiresAt)
	})
	return out, nil
}

func (s *employeeService) nextID() shared.ID {
	return shared.ID("pharma-employee-" + strconv.FormatInt(s.idCounter.Add(1), 10))
}

func (s *employeeService) nextAvailableID(ctx context.Context) (shared.ID, error) {
	for {
		id := s.nextID()
		item, err := s.repo.Get(ctx, id)
		if err != nil {
			return "", err
		}
		if item == nil {
			return id, nil
		}
	}
}

func (s *employeeService) appendAudit(ctx context.Context, actorID, action, resourceID string, detail map[string]any) {
	if s.audit == nil {
		return
	}
	actor := strings.TrimSpace(actorID)
	if actor == "" {
		actor = "system"
	}
	_, _ = s.audit.Append(ctx, actor, action, "pharma_oa_employee", resourceID, detail)
}

func toDomainInput(in EmployeeWriteInput) domainpharma.EmployeeInput {
	return domainpharma.EmployeeInput{
		Code:         in.Code,
		Name:         in.Name,
		DepartmentID: in.DepartmentID,
		PositionID:   in.PositionID,
		Phone:        in.Phone,
		Email:        in.Email,
		Certificates: in.Certificates,
	}
}

func employeeMatches(item *domainpharma.Employee, keyword string) bool {
	if item == nil {
		return false
	}
	return strings.Contains(strings.ToLower(item.ID.String()), keyword) ||
		strings.Contains(strings.ToLower(item.Code), keyword) ||
		strings.Contains(strings.ToLower(item.Name), keyword) ||
		strings.Contains(strings.ToLower(item.DepartmentID), keyword) ||
		strings.Contains(strings.ToLower(item.PositionID), keyword)
}

func cloneEmployee(item *domainpharma.Employee) *domainpharma.Employee {
	if item == nil {
		return nil
	}
	out := *item
	out.Certificates = append(make([]domainpharma.EmployeeCertificate, 0, len(item.Certificates)), item.Certificates...)
	return &out
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}
