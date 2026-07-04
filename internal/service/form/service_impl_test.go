package form

import (
	"context"
	"testing"
	"time"

	domainform "github.com/tinboxw/skoll/internal/domain/form"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestFormSchemaServiceCreateAndGet(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryRepository())
	created, err := service.CreateSchema(ctx, CreateSchemaInput{
		ID:           shared.ID("form-service-leave"),
		Key:          "oa.leave",
		Name:         "Leave Request",
		Version:      1,
		BusinessType: "oa.leave",
		Fields: []domainform.Field{
			{Key: "employee", Label: "Employee", Type: domainform.FieldUser, Required: true},
			{Key: "reason", Label: "Reason", Type: domainform.FieldTextarea, Required: true},
		},
		Now: time.Date(2026, time.July, 4, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateSchema error: %v", err)
	}
	if created.Key != "oa.leave" || created.Meta.CreatedAt.IsZero() {
		t.Fatalf("unexpected created schema: %+v", created)
	}

	byID, err := service.GetSchema(ctx, shared.ID("form-service-leave"))
	if err != nil {
		t.Fatalf("GetSchema error: %v", err)
	}
	byKey, err := service.GetSchemaByKey(ctx, "OA.LEAVE", 1)
	if err != nil {
		t.Fatalf("GetSchemaByKey error: %v", err)
	}
	if byID.ID != byKey.ID {
		t.Fatalf("expected same schema by id/key: byID=%+v byKey=%+v", byID, byKey)
	}

	byID.Fields[0].Label = "mutated"
	again, err := service.GetSchema(ctx, shared.ID("form-service-leave"))
	if err != nil {
		t.Fatalf("GetSchema again error: %v", err)
	}
	if again.Fields[0].Label == "mutated" {
		t.Fatal("repository should return cloned schemas")
	}
}

func TestFormSchemaServiceRejectsInvalidSchema(t *testing.T) {
	_, err := NewService(NewMemoryRepository()).CreateSchema(context.Background(), CreateSchemaInput{
		ID:           shared.ID("invalid"),
		Key:          "invalid",
		Name:         "Invalid",
		Version:      1,
		BusinessType: "invalid",
		Fields: []domainform.Field{
			{Key: "items", Label: "Items", Type: domainform.FieldDetailTable, DetailTable: &domainform.DetailTableConfig{}},
		},
	})
	if err == nil {
		t.Fatal("expected invalid detail table schema error")
	}
}
