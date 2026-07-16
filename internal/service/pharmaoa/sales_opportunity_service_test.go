package pharmaoa

import (
	"context"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
)

func TestSalesOpportunityLifecycleScopeAndStatistics(t *testing.T) {
	ctx := context.Background()
	customers := NewCustomerService(nil)
	products := NewProductService(nil)
	customerOne, _ := customers.Create(ctx, CustomerWriteInput{Code: "CUS-1", Name: "First Pharmacy", Region: "East", OrganizationID: "org-a", OwnerID: "sales-1", Contacts: []domainpharma.CustomerContact{{Name: "Amy"}}})
	customerTwo, _ := customers.Create(ctx, CustomerWriteInput{Code: "CUS-2", Name: "Second Pharmacy", Region: "East", OrganizationID: "org-a", OwnerID: "sales-2", Contacts: []domainpharma.CustomerContact{{Name: "Bob"}}})
	product, _ := products.Create(ctx, ProductWriteInput{Code: "DRUG-1", Name: "Example Drug", Spec: "10mg", DosageForm: "tablet", Manufacturer: "Skoll", ApprovalNumber: "A-1"})
	service := NewSalesOpportunityService(customers, products, nil)
	closeDate := time.Now().UTC().AddDate(0, 1, 0)
	first, err := service.Create(ctx, SalesOpportunityCreateInput{Title: "Hospital expansion", CustomerID: customerOne.ID.String(), ProductIDs: []string{product.ID.String()}, ExpectedAmountCents: 125050, EstimatedCloseDate: closeDate, ActorID: "sales-1"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Stage != domainpharma.SalesOpportunityLead || first.Products == nil || first.StageHistory == nil {
		t.Fatalf("unexpected opportunity: %+v", first)
	}
	if _, err = service.Create(ctx, SalesOpportunityCreateInput{Title: "Spoofed", CustomerID: customerTwo.ID.String(), ProductIDs: []string{product.ID.String()}, ExpectedAmountCents: 100, EstimatedCloseDate: closeDate, ActorID: "sales-1"}); err == nil {
		t.Fatal("sales user created an opportunity for an unauthorized customer")
	}
	second, err := service.Create(ctx, SalesOpportunityCreateInput{Title: "Retail rollout", CustomerID: customerTwo.ID.String(), ProductIDs: []string{product.ID.String()}, ExpectedAmountCents: 250000, EstimatedCloseDate: closeDate, ActorID: "sales-2"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Advance(ctx, first.ID.String(), SalesOpportunityAdvanceInput{Stage: domainpharma.SalesOpportunityProposal, ActorID: "sales-1"}); err == nil {
		t.Fatal("stage skipping was accepted")
	}
	for _, stage := range []domainpharma.SalesOpportunityStage{domainpharma.SalesOpportunityQualified, domainpharma.SalesOpportunityProposal, domainpharma.SalesOpportunityNegotiation, domainpharma.SalesOpportunityWon} {
		first, err = service.Advance(ctx, first.ID.String(), SalesOpportunityAdvanceInput{Stage: stage, ActorID: "sales-1", Note: "progress"})
		if err != nil {
			t.Fatalf("advance to %s: %v", stage, err)
		}
	}
	if _, err = service.Update(ctx, first.ID.String(), SalesOpportunityUpdateInput{Title: "Changed", ProductIDs: []string{product.ID.String()}, ExpectedAmountCents: 1, EstimatedCloseDate: closeDate, ActorID: "sales-1"}); err == nil {
		t.Fatal("terminal opportunity was updated")
	}
	if _, err = service.Advance(ctx, second.ID.String(), SalesOpportunityAdvanceInput{Stage: domainpharma.SalesOpportunityLost, ActorID: "sales-2"}); err == nil {
		t.Fatal("lost opportunity without reason was accepted")
	}
	if _, err = service.Advance(ctx, second.ID.String(), SalesOpportunityAdvanceInput{Stage: domainpharma.SalesOpportunityLost, Note: "Budget cancelled", ActorID: "sales-2"}); err != nil {
		t.Fatal(err)
	}
	own, _ := service.List(ctx, SalesOpportunityListInput{ActorID: "sales-1"})
	org, _ := service.List(ctx, SalesOpportunityListInput{ActorID: "manager", Scope: SalesOpportunityAccessScope{OrganizationID: "org-a"}})
	stats, _ := service.Statistics(ctx, SalesOpportunityStatisticsInput{ActorID: "manager", Scope: SalesOpportunityAccessScope{OrganizationID: "org-a"}})
	if len(own) != 1 || len(org) != 2 || stats.TotalCount != 2 || stats.WonCount != 1 || stats.LostCount != 1 || stats.ExpectedAmountCents != 375050 {
		t.Fatalf("scope/statistics mismatch own=%d org=%d stats=%+v", len(own), len(org), stats)
	}
}

func TestSalesOpportunityRejectsDisabledProductAndWritesAudit(t *testing.T) {
	ctx := context.Background()
	audit := auditsvc.NewService(clickhouse.NewAuditStore())
	customers := NewCustomerService(nil)
	products := NewProductService(nil)
	customer, _ := customers.Create(ctx, CustomerWriteInput{Code: "CUS-A", Name: "Audit Pharmacy", Region: "East", OrganizationID: "org-a", OwnerID: "sales-1", Contacts: []domainpharma.CustomerContact{{Name: "Amy"}}})
	product, _ := products.Create(ctx, ProductWriteInput{Code: "DRUG-A", Name: "Audit Drug", Spec: "10mg", DosageForm: "tablet", Manufacturer: "Skoll", ApprovalNumber: "A-2"})
	_, _ = products.Disable(ctx, product.ID.String(), ProductDisableInput{Reason: "withdrawn"})
	service := NewSalesOpportunityService(customers, products, audit)
	input := SalesOpportunityCreateInput{Title: "Audit", CustomerID: customer.ID.String(), ProductIDs: []string{product.ID.String()}, ExpectedAmountCents: 100, EstimatedCloseDate: time.Now().UTC().AddDate(0, 1, 0), ActorID: "sales-1"}
	if _, err := service.Create(ctx, input); err == nil {
		t.Fatal("disabled product was accepted")
	}
	active, _ := products.Create(ctx, ProductWriteInput{Code: "DRUG-B", Name: "Active Drug", Spec: "10mg", DosageForm: "tablet", Manufacturer: "Skoll", ApprovalNumber: "A-3"})
	input.ProductIDs = []string{active.ID.String()}
	item, err := service.Create(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = service.List(ctx, SalesOpportunityListInput{ActorID: "sales-1"})
	_, _ = service.Statistics(ctx, SalesOpportunityStatisticsInput{ActorID: "sales-1"})
	_, _ = service.Advance(ctx, item.ID.String(), SalesOpportunityAdvanceInput{Stage: domainpharma.SalesOpportunityQualified, ActorID: "sales-1"})
	records, err := audit.ListByActor(ctx, "sales-1", 10)
	if err != nil {
		t.Fatal(err)
	}
	actions := map[string]bool{}
	for _, record := range records {
		actions[record.Action] = true
	}
	for _, action := range []string{"pharma_oa.sales_opportunity.create", "pharma_oa.sales_opportunity.read_list", "pharma_oa.sales_opportunity.read_statistics", "pharma_oa.sales_opportunity.advance"} {
		if !actions[action] {
			t.Fatalf("audit action %q is missing from %+v", action, records)
		}
	}
}
