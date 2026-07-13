package pharmaoa

import (
	"context"
	"testing"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
)

func TestAnnouncementPublishingTargetsOrganizationAndRole(t *testing.T) {
	ctx := context.Background()
	service := NewAnnouncementService(nil)
	orgNotice, err := service.Create(ctx, AnnouncementCreateInput{Kind: domainpharma.AnnouncementKindNotice, Title: "Warehouse notice", Content: "Cycle count", Audience: domainpharma.AnnouncementAudience{OrganizationIDs: []string{"org-warehouse"}}, ActorID: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	policy, err := service.Create(ctx, AnnouncementCreateInput{Kind: domainpharma.AnnouncementKindPolicy, Title: "Quality policy", Content: "Effective now", Audience: domainpharma.AnnouncementAudience{RoleIDs: []string{"quality-manager"}}, Documents: []domainpharma.AnnouncementDocument{{FileID: "file-policy", FileName: "policy.pdf"}}, ActorID: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Publish(ctx, orgNotice.ID.String(), "publisher"); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Publish(ctx, policy.ID.String(), "publisher"); err != nil {
		t.Fatal(err)
	}
	warehouseItems, _ := service.List(ctx, AnnouncementListInput{ActorID: "user-1", OrganizationIDs: []string{"org-warehouse"}, RoleIDs: []string{"employee"}})
	qualityItems, _ := service.List(ctx, AnnouncementListInput{ActorID: "user-2", OrganizationIDs: []string{"org-other"}, RoleIDs: []string{"quality-manager"}})
	outsiderItems, _ := service.List(ctx, AnnouncementListInput{ActorID: "user-3", OrganizationIDs: []string{"org-other"}, RoleIDs: []string{"employee"}})
	if len(warehouseItems) != 1 || warehouseItems[0].ID != orgNotice.ID || len(qualityItems) != 1 || qualityItems[0].ID != policy.ID || len(outsiderItems) != 0 {
		t.Fatalf("audience filtering failed: warehouse=%+v quality=%+v outsider=%+v", warehouseItems, qualityItems, outsiderItems)
	}
}

func TestAnnouncementLifecycleWritesAudit(t *testing.T) {
	ctx := context.Background()
	audit := auditsvc.NewService(clickhouse.NewAuditStore())
	service := NewAnnouncementService(audit)
	item, err := service.Create(ctx, AnnouncementCreateInput{Kind: domainpharma.AnnouncementKindNotice, Title: "Audited notice", Content: "Read it", Audience: domainpharma.AnnouncementAudience{RoleIDs: []string{"employee"}}, ActorID: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Publish(ctx, item.ID.String(), "admin"); err != nil {
		t.Fatal(err)
	}
	if _, err = service.ConfirmRead(ctx, item.ID.String(), "employee-1", nil, []string{"employee"}); err != nil {
		t.Fatal(err)
	}
	adminRecords, _ := audit.ListByActor(ctx, "admin", 10)
	readerRecords, _ := audit.ListByActor(ctx, "employee-1", 10)
	if len(adminRecords) != 2 || len(readerRecords) != 1 || readerRecords[0].Action != "pharma_oa.announcement.read" {
		t.Fatalf("announcement audit trail incomplete: admin=%+v reader=%+v", adminRecords, readerRecords)
	}
}

func TestAnnouncementReadConfirmationIsQueryableAndIdempotent(t *testing.T) {
	ctx := context.Background()
	service := NewAnnouncementService(nil)
	item, _ := service.Create(ctx, AnnouncementCreateInput{Kind: domainpharma.AnnouncementKindNotice, Title: "Read me", Content: "Required", Audience: domainpharma.AnnouncementAudience{RoleIDs: []string{"employee"}}, ActorID: "admin"})
	_, _ = service.Publish(ctx, item.ID.String(), "admin")
	first, err := service.ConfirmRead(ctx, item.ID.String(), "user-1", nil, []string{"employee"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.ConfirmRead(ctx, item.ID.String(), "user-1", nil, []string{"employee"})
	if err != nil || second.ReadAt != first.ReadAt {
		t.Fatalf("duplicate read confirmation changed: %+v %+v %v", first, second, err)
	}
	if _, err = service.ConfirmRead(ctx, item.ID.String(), "outsider", nil, []string{"guest"}); err == nil {
		t.Fatal("out-of-audience user confirmed read")
	}
	receipts, err := service.ListReadConfirmations(ctx, item.ID.String())
	if err != nil || len(receipts) != 1 || receipts[0].UserID != "user-1" {
		t.Fatalf("unexpected receipts: %+v %v", receipts, err)
	}
}
