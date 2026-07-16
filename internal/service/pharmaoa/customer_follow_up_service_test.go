package pharmaoa

import (
	"context"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
)

func TestCustomerFollowUpDataScopeAndLifecycle(t *testing.T) {
	ctx := context.Background()
	customers := NewCustomerService(nil)
	ownerOne, _ := customers.Create(ctx, CustomerWriteInput{Code: "CUS-1", Name: "First Pharmacy", Region: "East", OrganizationID: "org-a", OwnerID: "sales-1", Contacts: []domainpharma.CustomerContact{{Name: "Amy"}}})
	ownerTwo, _ := customers.Create(ctx, CustomerWriteInput{Code: "CUS-2", Name: "Second Pharmacy", Region: "East", OrganizationID: "org-a", OwnerID: "sales-2", Contacts: []domainpharma.CustomerContact{{Name: "Bob"}}})
	otherOrg, _ := customers.Create(ctx, CustomerWriteInput{Code: "CUS-3", Name: "Third Pharmacy", Region: "West", OrganizationID: "org-b", OwnerID: "sales-3", Contacts: []domainpharma.CustomerContact{{Name: "Cara"}}})
	service := NewCustomerFollowUpService(customers, nil)
	now := time.Now().UTC().Add(time.Hour)
	first, err := service.Create(ctx, CustomerFollowUpCreateInput{CustomerID: ownerOne.ID.String(), ContactName: "Amy", Channel: domainpharma.CustomerFollowUpOnsite, ScheduledAt: now, ActorID: "sales-1"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Attachments == nil {
		t.Fatal("empty attachments must serialize as an array")
	}
	_, _ = service.Create(ctx, CustomerFollowUpCreateInput{CustomerID: ownerTwo.ID.String(), Channel: domainpharma.CustomerFollowUpPhone, ScheduledAt: now, ActorID: "sales-2"})
	_, _ = service.Create(ctx, CustomerFollowUpCreateInput{CustomerID: otherOrg.ID.String(), Channel: domainpharma.CustomerFollowUpOnline, ScheduledAt: now, ActorID: "sales-3"})
	if _, err = service.Create(ctx, CustomerFollowUpCreateInput{CustomerID: ownerTwo.ID.String(), Channel: domainpharma.CustomerFollowUpPhone, ScheduledAt: now, ActorID: "sales-1"}); err == nil {
		t.Fatal("sales user created a follow-up for an unauthorized customer")
	}
	own, _ := service.List(ctx, CustomerFollowUpListInput{ActorID: "sales-1"})
	org, _ := service.List(ctx, CustomerFollowUpListInput{ActorID: "manager-a", Scope: CustomerFollowUpAccessScope{OrganizationID: "org-a"}})
	all, _ := service.List(ctx, CustomerFollowUpListInput{ActorID: "admin", Scope: CustomerFollowUpAccessScope{IncludeAll: true}})
	if len(own) != 1 || own[0].OwnerID != "sales-1" || len(org) != 2 || len(all) != 3 {
		t.Fatalf("scope mismatch own=%d org=%d all=%d", len(own), len(org), len(all))
	}
	if _, err = service.Complete(ctx, first.ID.String(), CustomerFollowUpCompleteInput{Summary: "Order intent confirmed", NextAction: "Send quotation", ActorID: "sales-2"}); err == nil {
		t.Fatal("unauthorized user completed another owner's follow-up")
	}
	completed, err := service.Complete(ctx, first.ID.String(), CustomerFollowUpCompleteInput{Summary: "Order intent confirmed", NextAction: "Send quotation", ActorID: "sales-1"})
	if err != nil || completed.Status != domainpharma.CustomerFollowUpCompleted || completed.CompletedAt == nil {
		t.Fatalf("complete failed: %+v %v", completed, err)
	}
	if _, err = service.Cancel(ctx, first.ID.String(), CustomerFollowUpCancelInput{Reason: "late", ActorID: "sales-1"}); err == nil {
		t.Fatal("completed follow-up was cancelled")
	}
}

func TestCustomerFollowUpWritesLifecycleAudit(t *testing.T) {
	ctx := context.Background()
	audit := auditsvc.NewService(clickhouse.NewAuditStore())
	customers := NewCustomerService(nil)
	customer, _ := customers.Create(ctx, CustomerWriteInput{Code: "CUS-AUDIT", Name: "Audit Pharmacy", Region: "East", OrganizationID: "org-a", OwnerID: "sales-1", Contacts: []domainpharma.CustomerContact{{Name: "Amy"}}})
	service := NewCustomerFollowUpService(customers, audit)
	item, err := service.Create(ctx, CustomerFollowUpCreateInput{CustomerID: customer.ID.String(), Channel: domainpharma.CustomerFollowUpPhone, ScheduledAt: time.Now().UTC(), ActorID: "sales-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.List(ctx, CustomerFollowUpListInput{ActorID: "sales-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Complete(ctx, item.ID.String(), CustomerFollowUpCompleteInput{Summary: "Completed", ActorID: "sales-1"}); err != nil {
		t.Fatal(err)
	}
	records, err := audit.ListByActor(ctx, "sales-1", 10)
	if err != nil || len(records) != 3 {
		t.Fatalf("audit records=%+v err=%v", records, err)
	}
	actions := map[string]bool{}
	for _, record := range records {
		actions[record.Action] = true
	}
	for _, action := range []string{"pharma_oa.customer_follow_up.create", "pharma_oa.customer_follow_up.read_list", "pharma_oa.customer_follow_up.complete"} {
		if !actions[action] {
			t.Fatalf("audit action %q is missing from %+v", action, records)
		}
	}
}

func TestCustomerFollowUpRejectsUnsafeAttachment(t *testing.T) {
	ctx := context.Background()
	customers := NewCustomerService(nil)
	customer, _ := customers.Create(ctx, CustomerWriteInput{Code: "CUS-A", Name: "Attachment Pharmacy", Region: "East", OrganizationID: "org-a", OwnerID: "sales-1", Contacts: []domainpharma.CustomerContact{{Name: "Amy"}}})
	service := NewCustomerFollowUpService(customers, nil)
	_, err := service.Create(ctx, CustomerFollowUpCreateInput{CustomerID: customer.ID.String(), Channel: domainpharma.CustomerFollowUpEmail, ScheduledAt: time.Now().UTC(), ActorID: "sales-1", Attachments: []domainpharma.CustomerFollowUpAttachment{{FileID: "file-1", FileName: "payload.exe"}}})
	if err == nil {
		t.Fatal("unsafe attachment was accepted")
	}
}
