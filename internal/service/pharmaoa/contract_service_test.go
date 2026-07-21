package pharmaoa

import (
	"context"
	"fmt"
	"testing"
	"time"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	filesvc "github.com/tinboxw/skoll/internal/service/file"
	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
)

type contractFileFixture struct {
	items map[string]*domainfile.FileObject
	deny  bool
}

func (f contractFileFixture) Get(_ context.Context, in filesvc.GetInput) (*domainfile.FileObject, filesvc.AccessDecision, error) {
	if f.deny {
		return nil, filesvc.AccessDecision{Allowed: false}, fmt.Errorf("permission denied")
	}
	item := f.items[in.FileID.String()]
	if item == nil {
		return nil, filesvc.AccessDecision{Allowed: false}, fmt.Errorf("file not found")
	}
	copy := *item
	return &copy, filesvc.AccessDecision{Allowed: true}, nil
}

func TestContractArchiveRelatesPartiesFilesAndWorkflow(t *testing.T) {
	ctx := context.Background()
	service, _, _ := newContractFixture(t)
	supplierContract := createFixtureContract(t, service, domainpharma.ContractPartySupplier, "pharma-supplier-1", "CTR-SUP-001", "owner-1", "approver-1", time.Now().UTC().AddDate(0, 0, 10))
	if supplierContract.PartyName != "Acme Supplier" || supplierContract.WorkflowInstanceID == "" || len(supplierContract.Attachments) != 1 || supplierContract.Attachments[0].FileName != "contract.pdf" {
		t.Fatalf("supplier contract relation incomplete: %+v", supplierContract)
	}
	approved, err := service.Approve(ctx, supplierContract.ID.String(), ContractActionInput{ActorID: "approver-1", Comment: "approved"})
	if err != nil || approved.Status != domainpharma.ContractActive {
		t.Fatalf("approve contract: %+v %v", approved, err)
	}
	again, err := service.Approve(ctx, supplierContract.ID.String(), ContractActionInput{ActorID: "approver-1"})
	if err != nil || again.ApprovedAt == nil || !again.ApprovedAt.Equal(*approved.ApprovedAt) {
		t.Fatalf("approval is not idempotent: %+v %v", again, err)
	}
	customerContract := createFixtureContract(t, service, domainpharma.ContractPartyCustomer, "pharma-customer-1", "CTR-CUS-001", "owner-2", "approver-2", time.Now().UTC().AddDate(0, 0, 20))
	if customerContract.PartyName != "North Hospital" {
		t.Fatalf("customer relation missing: %+v", customerContract)
	}
	if _, err = service.Reject(ctx, customerContract.ID.String(), ContractActionInput{ActorID: "other-user"}); err == nil {
		t.Fatal("non-assignee rejected contract")
	}
	rejected, err := service.Reject(ctx, customerContract.ID.String(), ContractActionInput{ActorID: "approver-2", Comment: "declined"})
	if err != nil || rejected.Status != domainpharma.ContractRejected {
		t.Fatalf("reject contract: %+v %v", rejected, err)
	}
}

func TestContractArchiveRejectsInaccessibleFileBeforeWorkflow(t *testing.T) {
	ctx := context.Background()
	suppliers, customers := contractParties(t)
	workflow := workflowsvc.NewService(workflowsvc.NewMemoryRepository())
	service := NewContractService(suppliers, customers, workflow, contractFileFixture{deny: true}, notificationsvc.NewService(notificationsvc.NewMemoryRepository(), nil, nil), nil)
	_, err := service.Create(ctx, ContractCreateInput{Number: "CTR-FAIL", Title: "Denied file", PartyType: domainpharma.ContractPartySupplier, PartyID: "pharma-supplier-1", OwnerID: "owner-1", ApproverID: "approver-1", Currency: "CNY", EffectiveAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().AddDate(0, 1, 0), AttachmentIDs: []string{"file-1"}})
	if err == nil {
		t.Fatal("inaccessible attachment was accepted")
	}
	if _, workflowErr := workflow.GetInstance(ctx, shared.ID("contract-workflow-1")); workflowErr == nil {
		t.Fatal("workflow remained after attachment validation failure")
	}
	items, _ := service.List(ctx, ContractListInput{})
	if len(items) != 0 {
		t.Fatalf("failed contract was stored: %+v", items)
	}
}

func TestContractExpiryReminderIsQueryableAndIdempotent(t *testing.T) {
	ctx := context.Background()
	service, notifications, audit := newContractFixture(t)
	contract := createFixtureContract(t, service, domainpharma.ContractPartySupplier, "pharma-supplier-1", "CTR-EXP-001", "owner-1", "approver-1", time.Now().UTC().AddDate(0, 0, 5))
	if _, err := service.Approve(ctx, contract.ID.String(), ContractActionInput{ActorID: "approver-1"}); err != nil {
		t.Fatal(err)
	}
	first, err := service.ScanExpiry(ctx, ContractExpiryScanInput{Days: 30, ActorID: "scheduler"})
	if err != nil || first.MatchedCount != 1 || first.CreatedCount != 1 || len(first.Reminders) != 1 {
		t.Fatalf("first expiry scan: %+v %v", first, err)
	}
	second, err := service.ScanExpiry(ctx, ContractExpiryScanInput{Days: 30, ActorID: "scheduler"})
	if err != nil || second.MatchedCount != 1 || second.CreatedCount != 0 || second.Reminders[0].NotificationID != first.Reminders[0].NotificationID {
		t.Fatalf("duplicate expiry scan: %+v %v", second, err)
	}
	items, _ := notifications.List(ctx, notificationsvc.Filter{ActorID: "owner-1", Category: notificationsvc.CategoryReminder})
	if len(items) != 1 || items[0].Target.ID != contract.ID.String() {
		t.Fatalf("unexpected contract reminder notification: %+v", items)
	}
	records, _ := audit.ListByActor(ctx, "scheduler", 10)
	if len(records) != 3 {
		t.Fatalf("expiry scan audit trail incomplete: %+v", records)
	}
}

func newContractFixture(t *testing.T) (ContractService, *notificationsvc.Service, auditsvc.Service) {
	t.Helper()
	suppliers, customers := contractParties(t)
	notifications := notificationsvc.NewService(notificationsvc.NewMemoryRepository(), nil, nil)
	audit := auditsvc.NewService(clickhouse.NewAuditStore())
	file := &domainfile.FileObject{ID: "file-1", Name: "contract.pdf", MIME: "application/pdf", Size: 1024, Status: domainfile.StatusAvailable}
	service := NewContractService(suppliers, customers, workflowsvc.NewService(workflowsvc.NewMemoryRepository()), contractFileFixture{items: map[string]*domainfile.FileObject{"file-1": file}}, notifications, audit)
	return service, notifications, audit
}

func contractParties(t *testing.T) (SupplierService, CustomerService) {
	t.Helper()
	ctx := context.Background()
	suppliers := NewSupplierService(nil)
	if _, err := suppliers.Create(ctx, SupplierWriteInput{Code: "SUP-001", Name: "Acme Supplier", Rating: 5, Contacts: []domainpharma.SupplierContact{{Name: "Alice"}}}); err != nil {
		t.Fatal(err)
	}
	customers := NewCustomerService(nil)
	if _, err := customers.Create(ctx, CustomerWriteInput{Code: "CUS-001", Name: "North Hospital", Region: "North", OrganizationID: "org-1", OwnerID: "owner-2", Rating: 5, Contacts: []domainpharma.CustomerContact{{Name: "Bob"}}, Scope: CustomerAccessScope{IncludeAll: true}}); err != nil {
		t.Fatal(err)
	}
	return suppliers, customers
}

func createFixtureContract(t *testing.T, service ContractService, partyType domainpharma.ContractPartyType, partyID, number, owner, approver string, expiresAt time.Time) *domainpharma.Contract {
	t.Helper()
	item, err := service.Create(context.Background(), ContractCreateInput{Number: number, Title: "Distribution agreement", PartyType: partyType, PartyID: partyID, OwnerID: owner, ApproverID: approver, Amount: 12000, Currency: "CNY", EffectiveAt: time.Now().UTC().Add(-time.Hour), ExpiresAt: expiresAt, AttachmentIDs: []string{"file-1"}})
	if err != nil {
		t.Fatal(err)
	}
	return item
}
