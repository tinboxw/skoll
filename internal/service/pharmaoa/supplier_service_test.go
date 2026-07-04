package pharmaoa

import (
	"context"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
)

func TestSupplierServiceCreateReminderAndDisablePurchaseBlock(t *testing.T) {
	service := NewSupplierService(nil).(*supplierService)
	service.nowFn = func() time.Time { return time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC) }

	created, err := service.Create(context.Background(), supplierFixture("SUP001", service.nowFn().AddDate(0, 0, 10)))
	if err != nil {
		t.Fatalf("create supplier: %v", err)
	}
	if created.Status != domainpharma.SupplierStatusActive || created.Rating != 5 || len(created.Qualifications[0].Attachments) != 1 {
		t.Fatalf("unexpected supplier: %+v", created)
	}

	eligible, err := service.ValidatePurchaseSupplier(context.Background(), created.ID.String())
	if err != nil {
		t.Fatalf("validate purchase supplier: %v", err)
	}
	if !eligible.Allowed {
		t.Fatalf("expected active supplier to be allowed: %+v", eligible)
	}

	reminders, err := service.QualificationReminders(context.Background(), 30)
	if err != nil {
		t.Fatalf("qualification reminders: %v", err)
	}
	if len(reminders) != 1 || reminders[0].SupplierID != created.ID.String() {
		t.Fatalf("unexpected reminders: %+v", reminders)
	}

	disabled, err := service.Disable(context.Background(), created.ID.String(), SupplierDisableInput{Reason: "blacklisted", ActorID: "admin"})
	if err != nil {
		t.Fatalf("disable supplier: %v", err)
	}
	if disabled.Status != domainpharma.SupplierStatusDisabled || disabled.DisableReason != "blacklisted" {
		t.Fatalf("unexpected disabled supplier: %+v", disabled)
	}
	eligible, err = service.ValidatePurchaseSupplier(context.Background(), created.ID.String())
	if err != nil {
		t.Fatalf("validate disabled supplier: %v", err)
	}
	if eligible.Allowed || eligible.Reason != "supplier is disabled" {
		t.Fatalf("expected disabled supplier blocked: %+v", eligible)
	}
}

func TestSupplierRejectsUnsafeAttachmentAndExpiredQualification(t *testing.T) {
	service := NewSupplierService(nil).(*supplierService)
	service.nowFn = func() time.Time { return time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC) }

	bad := supplierFixture("SUP001", service.nowFn().AddDate(0, 0, 10))
	bad.Qualifications[0].Attachments[0].FileName = "..\\run.ps1"
	if _, err := service.Create(context.Background(), bad); err == nil {
		t.Fatal("expected unsafe attachment to be rejected")
	}

	expired, err := service.Create(context.Background(), supplierFixture("SUP002", service.nowFn().AddDate(0, 0, -1)))
	if err != nil {
		t.Fatalf("create expired supplier: %v", err)
	}
	eligible, err := service.ValidatePurchaseSupplier(context.Background(), expired.ID.String())
	if err != nil {
		t.Fatalf("validate expired supplier: %v", err)
	}
	if eligible.Allowed || eligible.Reason == "" {
		t.Fatalf("expected expired qualification to block purchase: %+v", eligible)
	}
}

func supplierFixture(code string, expiresAt time.Time) SupplierWriteInput {
	return SupplierWriteInput{
		Code:   code,
		Name:   "Skoll Medical Supply",
		Rating: 5,
		Contacts: []domainpharma.SupplierContact{{
			ID:      "contact-primary",
			Name:    "Jane",
			Phone:   "13800000000",
			Email:   "jane@skoll.local",
			Title:   "Account Manager",
			Primary: true,
		}},
		Qualifications: []domainpharma.SupplierQualification{{
			ID:        "gsp-license",
			Name:      "GSP License",
			Number:    "GSP-SUP-001",
			ExpiresAt: expiresAt,
			Attachments: []domainpharma.SupplierAttachment{{
				FileID:   "file-001",
				FileName: "gsp-license.pdf",
				MimeType: "application/pdf",
				Size:     1024,
			}},
		}},
		ActorID: "admin",
	}
}
