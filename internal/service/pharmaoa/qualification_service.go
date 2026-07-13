package pharmaoa

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
)

type QualificationSubjectType string
type QualificationStatus string

const (
	QualificationSubjectEmployee QualificationSubjectType = "employee"
	QualificationSubjectSupplier QualificationSubjectType = "supplier"
	QualificationSubjectCustomer QualificationSubjectType = "customer"

	QualificationStatusValid     QualificationStatus = "valid"
	QualificationStatusExpiring  QualificationStatus = "expiring"
	QualificationStatusExpired   QualificationStatus = "expired"
	QualificationStatusPermanent QualificationStatus = "permanent"
)

type QualificationService interface {
	List(ctx context.Context, in QualificationListInput) ([]QualificationRecord, error)
	ScanExpiry(ctx context.Context, in QualificationScanInput) (*QualificationScanResult, error)
}

type QualificationListInput struct {
	Keyword     string
	SubjectType QualificationSubjectType
	Status      QualificationStatus
	Days        int
}

type QualificationRecord struct {
	ID                     string                   `json:"id"`
	SubjectType            QualificationSubjectType `json:"subjectType"`
	SubjectID              string                   `json:"subjectId"`
	SubjectCode            string                   `json:"subjectCode"`
	SubjectName            string                   `json:"subjectName"`
	SubjectStatus          string                   `json:"subjectStatus"`
	QualificationID        string                   `json:"qualificationId"`
	QualificationName      string                   `json:"qualificationName"`
	Number                 string                   `json:"number"`
	ExpiresAt              time.Time                `json:"expiresAt"`
	Status                 QualificationStatus      `json:"status"`
	AttachmentCount        int                      `json:"attachmentCount"`
	RecipientID            string                   `json:"recipientId"`
	TargetPath             string                   `json:"targetPath"`
	ReminderNotificationID string                   `json:"reminderNotificationId,omitempty"`
}

type QualificationScanInput struct {
	Days    int
	ActorID string
}

type QualificationScanResult struct {
	MatchedCount int                   `json:"matchedCount"`
	CreatedCount int                   `json:"createdCount"`
	Reminders    []QualificationRecord `json:"reminders"`
}

type qualificationService struct {
	employees     EmployeeService
	suppliers     SupplierService
	customers     CustomerService
	notifications *notificationsvc.Service
	audit         auditsvc.Service
	nowFn         func() time.Time
	scanMu        sync.Mutex
	mu            sync.RWMutex
	reminders     map[string]string
}

func NewQualificationService(employees EmployeeService, suppliers SupplierService, customers CustomerService, notifications *notificationsvc.Service, audit auditsvc.Service) QualificationService {
	return &qualificationService{employees: employees, suppliers: suppliers, customers: customers, notifications: notifications, audit: audit, nowFn: func() time.Time { return time.Now().UTC() }, reminders: map[string]string{}}
}

func (s *qualificationService) List(ctx context.Context, in QualificationListInput) ([]QualificationRecord, error) {
	if s == nil || s.employees == nil || s.suppliers == nil || s.customers == nil {
		return nil, fmt.Errorf("qualification service dependencies are required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	days := in.Days
	if days <= 0 {
		days = 30
	}
	now := s.nowFn()
	deadline := now.AddDate(0, 0, days)
	records, err := s.collect(ctx, now, deadline)
	if err != nil {
		return nil, err
	}
	keyword := strings.ToLower(strings.TrimSpace(in.Keyword))
	out := make([]QualificationRecord, 0, len(records))
	for _, record := range records {
		if in.SubjectType != "" && record.SubjectType != in.SubjectType || in.Status != "" && record.Status != in.Status {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(strings.Join([]string{record.SubjectCode, record.SubjectName, record.QualificationName, record.Number}, " ")), keyword) {
			continue
		}
		s.mu.RLock()
		record.ReminderNotificationID = s.reminders[qualificationReminderKey(record)]
		s.mu.RUnlock()
		out = append(out, record)
	}
	sort.Slice(out, func(i, j int) bool {
		left, right := qualificationSortTime(out[i].ExpiresAt), qualificationSortTime(out[j].ExpiresAt)
		if left.Equal(right) {
			return out[i].ID < out[j].ID
		}
		return left.Before(right)
	})
	return out, nil
}

func (s *qualificationService) ScanExpiry(ctx context.Context, in QualificationScanInput) (*QualificationScanResult, error) {
	if s == nil || s.notifications == nil {
		return nil, fmt.Errorf("qualification notification service is required")
	}
	if in.Days <= 0 {
		in.Days = 30
	}
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	records, err := s.List(ctx, QualificationListInput{Days: in.Days})
	if err != nil {
		return nil, err
	}
	result := &QualificationScanResult{Reminders: []QualificationRecord{}}
	for _, record := range records {
		if record.Status != QualificationStatusExpired && record.Status != QualificationStatusExpiring || !qualificationSubjectActive(record) {
			continue
		}
		result.MatchedCount++
		key := qualificationReminderKey(record)
		s.mu.RLock()
		notificationID, exists := s.reminders[key]
		s.mu.RUnlock()
		if !exists {
			notification, createErr := s.notifications.Create(ctx, notificationsvc.CreateInput{
				ID:       qualificationNotificationID(record),
				Category: notificationsvc.CategoryReminder,
				Title:    "Qualification expiry reminder",
				Body:     fmt.Sprintf("%s %s qualification %s expires on %s.", record.SubjectType, record.SubjectName, record.QualificationName, record.ExpiresAt.Format("2006-01-02")),
				ActorID:  qualificationRecipient(record),
				Target:   notificationsvc.Target{Type: "pharma_qualification", ID: record.ID, Path: record.TargetPath},
				DueAt:    record.ExpiresAt,
			})
			if createErr != nil {
				return nil, createErr
			}
			notificationID = notification.ID
			s.mu.Lock()
			s.reminders[key] = notificationID
			s.mu.Unlock()
			result.CreatedCount++
			s.appendAudit(ctx, in.ActorID, "pharma_oa.qualification.expiry_remind", record.ID, map[string]any{"subjectType": record.SubjectType, "subjectId": record.SubjectID, "expiresAt": record.ExpiresAt, "recipientId": qualificationRecipient(record), "notificationId": notificationID})
		}
		record.ReminderNotificationID = notificationID
		result.Reminders = append(result.Reminders, record)
	}
	s.appendAudit(ctx, in.ActorID, "pharma_oa.qualification.expiry_scan", "qualifications", map[string]any{"days": in.Days, "matched": result.MatchedCount, "created": result.CreatedCount})
	return result, nil
}

func (s *qualificationService) collect(ctx context.Context, now, deadline time.Time) ([]QualificationRecord, error) {
	out := make([]QualificationRecord, 0)
	for offset := 0; ; offset += 100 {
		items, err := s.employees.List(ctx, EmployeeListInput{Offset: offset, Limit: 100})
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			for _, qualification := range item.Certificates {
				out = append(out, newQualificationRecord(QualificationSubjectEmployee, item.ID.String(), item.Code, item.Name, string(item.Status), qualification.ID, qualification.Name, qualification.Number, qualification.ExpiresAt, 0, item.ID.String(), "/skoll/pharma-oa/employees?employeeId="+url.QueryEscape(item.ID.String()), now, deadline))
			}
		}
		if len(items) < 100 {
			break
		}
	}
	for offset := 0; ; offset += 100 {
		items, err := s.suppliers.List(ctx, SupplierListInput{Offset: offset, Limit: 100})
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			for _, qualification := range item.Qualifications {
				out = append(out, newQualificationRecord(QualificationSubjectSupplier, item.ID.String(), item.Code, item.Name, string(item.Status), qualification.ID, qualification.Name, qualification.Number, qualification.ExpiresAt, len(qualification.Attachments), "system", "/skoll/pharma-oa/suppliers?supplierId="+url.QueryEscape(item.ID.String()), now, deadline))
			}
		}
		if len(items) < 100 {
			break
		}
	}
	for offset := 0; ; offset += 100 {
		items, err := s.customers.List(ctx, CustomerListInput{Offset: offset, Limit: 100, Scope: CustomerAccessScope{IncludeAll: true}})
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			for _, qualification := range item.Qualifications {
				out = append(out, newQualificationRecord(QualificationSubjectCustomer, item.ID.String(), item.Code, item.Name, string(item.Status), qualification.ID, qualification.Name, qualification.Number, qualification.ExpiresAt, len(qualification.Attachments), item.OwnerID, "/skoll/pharma-oa/customers?customerId="+url.QueryEscape(item.ID.String()), now, deadline))
			}
		}
		if len(items) < 100 {
			break
		}
	}
	return out, nil
}

func newQualificationRecord(subjectType QualificationSubjectType, subjectID, subjectCode, subjectName, subjectStatus, qualificationID, qualificationName, number string, expiresAt time.Time, attachmentCount int, recipientID, targetPath string, now, deadline time.Time) QualificationRecord {
	return QualificationRecord{ID: string(subjectType) + ":" + subjectID + ":" + qualificationID, SubjectType: subjectType, SubjectID: subjectID, SubjectCode: subjectCode, SubjectName: subjectName, SubjectStatus: subjectStatus, QualificationID: qualificationID, QualificationName: qualificationName, Number: number, ExpiresAt: expiresAt.UTC(), Status: qualificationStatus(expiresAt, now, deadline), AttachmentCount: attachmentCount, RecipientID: strings.TrimSpace(recipientID), TargetPath: targetPath}
}

func qualificationStatus(expiresAt, now, deadline time.Time) QualificationStatus {
	if expiresAt.IsZero() {
		return QualificationStatusPermanent
	}
	if !expiresAt.After(now) {
		return QualificationStatusExpired
	}
	if !expiresAt.After(deadline) {
		return QualificationStatusExpiring
	}
	return QualificationStatusValid
}

func qualificationSubjectActive(record QualificationRecord) bool {
	if record.SubjectType == QualificationSubjectEmployee {
		return record.SubjectStatus != string(domainpharma.EmployeeStatusLeft)
	}
	return record.SubjectStatus == "active"
}

func qualificationRecipient(record QualificationRecord) string {
	if strings.TrimSpace(record.RecipientID) == "" {
		return "system"
	}
	return strings.TrimSpace(record.RecipientID)
}

func qualificationReminderKey(record QualificationRecord) string {
	return record.ID + "|" + record.ExpiresAt.UTC().Format(time.RFC3339Nano)
}

func qualificationNotificationID(record QualificationRecord) string {
	cleanID := strings.NewReplacer(":", "-", "/", "-", "\\", "-").Replace(record.ID)
	return "qualification-expiry-" + cleanID + "-" + record.ExpiresAt.UTC().Format("20060102")
}

func qualificationSortTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	}
	return value
}

func (s *qualificationService) appendAudit(ctx context.Context, actorID, action, resourceID string, detail map[string]any) {
	if s.audit == nil {
		return
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		actorID = "system"
	}
	_, _ = s.audit.Append(ctx, actorID, action, "pharma_oa_qualification", resourceID, detail)
}
