package pharmaoa

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
)

type EmployeeService interface {
	Create(ctx context.Context, in EmployeeWriteInput) (*domainpharma.Employee, error)
	List(ctx context.Context, in EmployeeListInput) ([]*domainpharma.Employee, error)
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
	mu        sync.RWMutex
	items     map[string]*domainpharma.Employee
	audit     auditsvc.Service
	nowFn     func() time.Time
	idCounter int64
}

func NewEmployeeService(audit auditsvc.Service) EmployeeService {
	return &employeeService{
		items: map[string]*domainpharma.Employee{},
		audit: audit,
		nowFn: func() time.Time { return time.Now().UTC() },
	}
}

func (s *employeeService) Create(ctx context.Context, in EmployeeWriteInput) (*domainpharma.Employee, error) {
	now := s.nowFn()
	entity, err := domainpharma.NewEmployee(s.nextID(), toDomainInput(in), now)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if existing := s.findByCodeLocked(entity.Code, ""); existing != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("employee code already exists")
	}
	s.items[entity.ID.String()] = cloneEmployee(entity)
	s.mu.Unlock()
	s.appendAudit(ctx, in.ActorID, "pharma_oa.employee.create", entity.ID.String(), map[string]any{"code": entity.Code})
	return cloneEmployee(entity), nil
}

func (s *employeeService) List(_ context.Context, in EmployeeListInput) ([]*domainpharma.Employee, error) {
	limit := normalizeLimit(in.Limit)
	offset := in.Offset
	if offset < 0 {
		offset = 0
	}
	keyword := strings.ToLower(strings.TrimSpace(in.Keyword))
	status := strings.ToLower(strings.TrimSpace(in.Status))
	s.mu.RLock()
	items := make([]*domainpharma.Employee, 0, len(s.items))
	for _, item := range s.items {
		if status != "" && string(item.Status) != status {
			continue
		}
		if keyword != "" && !employeeMatches(item, keyword) {
			continue
		}
		items = append(items, cloneEmployee(item))
	}
	s.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool {
		return items[i].Code < items[j].Code
	})
	if offset >= len(items) {
		return []*domainpharma.Employee{}, nil
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end], nil
}

func (s *employeeService) Update(ctx context.Context, id string, in EmployeeWriteInput) (*domainpharma.Employee, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	now := s.nowFn()
	s.mu.Lock()
	current := s.items[id]
	if current == nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("employee not found")
	}
	if existing := s.findByCodeLocked(in.Code, id); existing != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("employee code already exists")
	}
	next := cloneEmployee(current)
	if err := next.Update(toDomainInput(in), now); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.items[id] = cloneEmployee(next)
	s.mu.Unlock()
	s.appendAudit(ctx, in.ActorID, "pharma_oa.employee.update", id, map[string]any{"code": next.Code})
	return cloneEmployee(next), nil
}

func (s *employeeService) MarkLeft(ctx context.Context, id string, in EmployeeLeaveInput) (*domainpharma.Employee, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	now := s.nowFn()
	s.mu.Lock()
	current := s.items[id]
	if current == nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("employee not found")
	}
	next := cloneEmployee(current)
	if err := next.MarkLeft(in.Reason, now); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.items[id] = cloneEmployee(next)
	s.mu.Unlock()
	s.appendAudit(ctx, in.ActorID, "pharma_oa.employee.leave", id, map[string]any{"reason": strings.TrimSpace(in.Reason)})
	return cloneEmployee(next), nil
}

func (s *employeeService) QualificationReminders(_ context.Context, days int) ([]EmployeeQualificationReminder, error) {
	if days <= 0 {
		days = 30
	}
	deadline := s.nowFn().AddDate(0, 0, days)
	s.mu.RLock()
	out := make([]EmployeeQualificationReminder, 0)
	for _, item := range s.items {
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
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		return out[i].ExpiresAt.Before(out[j].ExpiresAt)
	})
	return out, nil
}

func (s *employeeService) findByCodeLocked(code string, exceptID string) *domainpharma.Employee {
	code = strings.TrimSpace(code)
	for id, item := range s.items {
		if id == exceptID {
			continue
		}
		if strings.EqualFold(item.Code, code) {
			return item
		}
	}
	return nil
}

func (s *employeeService) nextID() shared.ID {
	s.idCounter++
	return shared.ID("pharma-employee-" + strconv.FormatInt(s.idCounter, 10))
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
	out.Certificates = append([]domainpharma.EmployeeCertificate(nil), item.Certificates...)
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
