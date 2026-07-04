package pharmaoa

import (
	"context"
	"strings"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
)

func TestCustomerServiceScopeAndSalesEligibility(t *testing.T) {
	service := NewCustomerService(nil).(*customerService)
	service.nowFn = func() time.Time { return time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC) }
	ctx := context.Background()

	first, err := service.Create(ctx, CustomerWriteInput{
		Code:           "CUST-001",
		Name:           "East Hospital",
		Region:         "East",
		OrganizationID: "org-a",
		OwnerID:        "sales-a",
		Rating:         5,
		Contacts: []domainpharma.CustomerContact{{
			Name:  "Alice",
			Phone: "10086",
			Title: "Buyer",
		}},
		Qualifications: []domainpharma.CustomerQualification{{
			Name:      "Medical Institution License",
			Number:    "LIC-001",
			ExpiresAt: service.nowFn().AddDate(0, 0, 20),
			Attachments: []domainpharma.CustomerAttachment{{
				FileID:   "file-1",
				FileName: "license.pdf",
				Size:     128,
			}},
		}},
		ActorID: "sales-a",
	})
	if err != nil {
		t.Fatalf("create first customer: %v", err)
	}
	if first.Rating != 5 || first.Contacts[0].Name != "Alice" || first.Qualifications[0].Attachments[0].FileID != "file-1" {
		t.Fatalf("unexpected customer: %+v", first)
	}

	second, err := service.Create(ctx, CustomerWriteInput{
		Code:           "CUST-002",
		Name:           "West Pharmacy",
		Region:         "West",
		OrganizationID: "org-b",
		OwnerID:        "sales-b",
		Contacts:       []domainpharma.CustomerContact{{Name: "Bob"}},
		Qualifications: []domainpharma.CustomerQualification{{Name: "Business License", ExpiresAt: service.nowFn().AddDate(0, 0, 90)}},
		ActorID:        "sales-b",
	})
	if err != nil {
		t.Fatalf("create second customer: %v", err)
	}
	if second.Rating != 3 {
		t.Fatalf("expected default rating 3, got %d", second.Rating)
	}

	owned, err := service.List(ctx, CustomerListInput{Scope: CustomerAccessScope{OwnerID: "sales-a"}})
	if err != nil {
		t.Fatalf("list owned: %v", err)
	}
	if len(owned) != 1 || owned[0].ID != first.ID {
		t.Fatalf("expected only sales-a customer, got %+v", owned)
	}

	org, err := service.List(ctx, CustomerListInput{Scope: CustomerAccessScope{OrganizationID: "org-b"}})
	if err != nil {
		t.Fatalf("list org: %v", err)
	}
	if len(org) != 1 || org[0].ID != second.ID {
		t.Fatalf("expected only org-b customer, got %+v", org)
	}

	_, err = service.Update(ctx, first.ID.String(), CustomerWriteInput{
		Code:           "CUST-001",
		Name:           "East Hospital Updated",
		Region:         "East",
		OrganizationID: "org-a",
		OwnerID:        "sales-a",
		Contacts:       []domainpharma.CustomerContact{{Name: "Alice"}},
		Scope:          CustomerAccessScope{OwnerID: "sales-b", OrganizationID: "org-b"},
	})
	if err == nil || !strings.Contains(err.Error(), "access denied") {
		t.Fatalf("expected cross owner/org update denial, got %v", err)
	}

	eligible, err := service.ValidateSalesCustomer(ctx, first.ID.String(), CustomerAccessScope{OwnerID: "sales-a"})
	if err != nil {
		t.Fatalf("validate sales customer: %v", err)
	}
	if !eligible.Allowed {
		t.Fatalf("expected eligible customer, got %+v", eligible)
	}

	_, err = service.Disable(ctx, first.ID.String(), CustomerDisableInput{
		Reason:  "blacklisted",
		ActorID: "sales-a",
		Scope:   CustomerAccessScope{OwnerID: "sales-a"},
	})
	if err != nil {
		t.Fatalf("disable customer: %v", err)
	}
	eligible, err = service.ValidateSalesCustomer(ctx, first.ID.String(), CustomerAccessScope{OwnerID: "sales-a"})
	if err != nil {
		t.Fatalf("validate disabled customer: %v", err)
	}
	if eligible.Allowed || !strings.Contains(eligible.Reason, "disabled") {
		t.Fatalf("expected disabled customer blocked, got %+v", eligible)
	}
}

func TestCustomerServiceQualificationExpiryAndAttachmentValidation(t *testing.T) {
	service := NewCustomerService(nil).(*customerService)
	service.nowFn = func() time.Time { return time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC) }
	ctx := context.Background()

	expired, err := service.Create(ctx, CustomerWriteInput{
		Code:           "CUST-003",
		Name:           "Expired Clinic",
		Region:         "North",
		OrganizationID: "org-a",
		OwnerID:        "sales-a",
		Contacts:       []domainpharma.CustomerContact{{Name: "Cathy"}},
		Qualifications: []domainpharma.CustomerQualification{{
			Name:      "Business License",
			Number:    "LIC-003",
			ExpiresAt: service.nowFn().AddDate(0, 0, -1),
		}},
		ActorID: "sales-a",
	})
	if err != nil {
		t.Fatalf("create expired customer: %v", err)
	}

	reminders, err := service.QualificationReminders(ctx, CustomerReminderInput{Days: 30, Scope: CustomerAccessScope{OwnerID: "sales-a"}})
	if err != nil {
		t.Fatalf("qualification reminders: %v", err)
	}
	if len(reminders) != 1 || reminders[0].CustomerID != expired.ID.String() {
		t.Fatalf("expected expired customer reminder, got %+v", reminders)
	}

	eligible, err := service.ValidateSalesCustomer(ctx, expired.ID.String(), CustomerAccessScope{OwnerID: "sales-a"})
	if err != nil {
		t.Fatalf("validate expired customer: %v", err)
	}
	if eligible.Allowed || !strings.Contains(eligible.Reason, "expired") {
		t.Fatalf("expected expired qualification blocked, got %+v", eligible)
	}

	_, err = service.Create(ctx, CustomerWriteInput{
		Code:           "CUST-004",
		Name:           "Unsafe Clinic",
		Region:         "South",
		OrganizationID: "org-a",
		OwnerID:        "sales-a",
		Contacts:       []domainpharma.CustomerContact{{Name: "Dan"}},
		Qualifications: []domainpharma.CustomerQualification{{
			Name: "Business License",
			Attachments: []domainpharma.CustomerAttachment{{
				FileID:   "file-unsafe",
				FileName: "../license.exe",
				Size:     1,
			}},
		}},
		ActorID: "sales-a",
	})
	if err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("expected unsafe attachment rejection, got %v", err)
	}
}
