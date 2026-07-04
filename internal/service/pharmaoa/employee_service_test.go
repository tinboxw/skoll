package pharmaoa

import (
	"context"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
)

func TestEmployeeServiceCreateUpdateLeaveAndReminders(t *testing.T) {
	service := NewEmployeeService(nil).(*employeeService)
	service.nowFn = func() time.Time { return time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC) }

	created, err := service.Create(context.Background(), EmployeeWriteInput{
		Code:         "EMP001",
		Name:         "Alice",
		DepartmentID: "quality",
		PositionID:   "qa",
		ActorID:      "admin",
		Certificates: []domainpharma.EmployeeCertificate{{
			ID:        "cert-gsp",
			Name:      "GSP",
			Number:    "GSP-001",
			ExpiresAt: service.nowFn().AddDate(0, 0, 10),
		}},
	})
	if err != nil {
		t.Fatalf("create employee: %v", err)
	}
	if created.Status != domainpharma.EmployeeStatusActive || created.Code != "EMP001" {
		t.Fatalf("unexpected created employee: %+v", created)
	}

	updated, err := service.Update(context.Background(), created.ID.String(), EmployeeWriteInput{
		Code:         "EMP001",
		Name:         "Alice Zhang",
		DepartmentID: "quality",
		PositionID:   "qa-lead",
		ActorID:      "admin",
		Certificates: created.Certificates,
	})
	if err != nil {
		t.Fatalf("update employee: %v", err)
	}
	if updated.Name != "Alice Zhang" || updated.PositionID != "qa-lead" {
		t.Fatalf("unexpected updated employee: %+v", updated)
	}

	reminders, err := service.QualificationReminders(context.Background(), 30)
	if err != nil {
		t.Fatalf("qualification reminders: %v", err)
	}
	if len(reminders) != 1 || reminders[0].EmployeeID != created.ID.String() {
		t.Fatalf("unexpected reminders: %+v", reminders)
	}

	left, err := service.MarkLeft(context.Background(), created.ID.String(), EmployeeLeaveInput{Reason: "resigned", ActorID: "admin"})
	if err != nil {
		t.Fatalf("mark left: %v", err)
	}
	if left.Status != domainpharma.EmployeeStatusLeft || left.LeaveReason != "resigned" {
		t.Fatalf("unexpected left employee: %+v", left)
	}
	reminders, err = service.QualificationReminders(context.Background(), 30)
	if err != nil {
		t.Fatalf("qualification reminders after leave: %v", err)
	}
	if len(reminders) != 0 {
		t.Fatalf("left employee should not produce reminders: %+v", reminders)
	}
}
