package datastore

import (
	"context"
	"errors"
	"testing"

	"github.com/tinboxw/skoll/internal/repository"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

func TestServiceExecutesTypedCursorQueriesAndScopedMutations(t *testing.T) {
	_, db, _, audit := newMutationFixture(t)
	registry := NewSchemaRegistry()
	if _, err := registry.Register(validPluginSchema("medical_oa")); err != nil {
		t.Fatal(err)
	}
	scopes := &recordingScopeService{predicate: mustScope(t, pluginsdk.TrustedScope{
		SubjectID: "employee-1", TenantIDs: []string{"tenant-a"}, OwnerIDs: []string{"employee-1"}, OrganizationIDs: []string{"org-a"},
	})}
	uow := storesql.NewUnitOfWorkWithDB(db)
	service, err := NewService(db, uow, registry, scopes, audit, DialectSQLite, "medical_oa")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"product-1", "product-2", "product-3"} {
		if _, err = service.Mutate(context.Background(), productMutation(pluginsdk.DataMutationInsert, id, "insert-"+id)); err != nil {
			t.Fatalf("insert %s: %v", id, err)
		}
	}

	query := basicQuery()
	query.Page.Limit = 2
	first, err := service.Query(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Records) != 2 || !first.HasMore || first.NextCursor == "" || first.Records[0].Values["id"].Value != "product-1" {
		t.Fatalf("first page=%+v", first)
	}
	query.Page.Cursor = first.NextCursor
	second, err := service.Query(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Records) != 1 || second.HasMore || second.NextCursor != "" || second.Records[0].Values["id"].Value != "product-3" {
		t.Fatalf("second page=%+v", second)
	}

	other, err := NewService(db, uow, registry, scopes, audit, DialectSQLite, "other_plugin")
	if err != nil {
		t.Fatal(err)
	}
	otherQuery := basicQuery()
	otherQuery.Scope.Permission.Resource = "other_plugin.product"
	_, err = other.Query(context.Background(), otherQuery)
	assertStoreError(t, err, pluginsdk.DataStoreErrorNotFound, "pluginId")
}

func TestServiceUsesOuterTransactionForMutationAndQuery(t *testing.T) {
	_, db, _, audit := newMutationFixture(t)
	registry := NewSchemaRegistry()
	if _, err := registry.Register(validPluginSchema("medical_oa")); err != nil {
		t.Fatal(err)
	}
	scopes := &recordingScopeService{predicate: mustScope(t, pluginsdk.TrustedScope{
		SubjectID: "employee-1", TenantIDs: []string{"tenant-a"}, OwnerIDs: []string{"employee-1"}, OrganizationIDs: []string{"org-a"},
	})}
	uow := storesql.NewUnitOfWorkWithDB(db)
	service, err := NewService(db, uow, registry, scopes, audit, DialectSQLite, "medical_oa")
	if err != nil {
		t.Fatal(err)
	}
	rollback := errors.New("rollback")
	err = uow.Do(context.Background(), func(tx repository.Tx) error {
		if _, mutateErr := service.Mutate(tx.Context(), productMutation(pluginsdk.DataMutationInsert, "product-tx", "insert-product-tx")); mutateErr != nil {
			return mutateErr
		}
		query := basicQuery()
		query.Filter = &pluginsdk.DataFilter{Field: "id", Operator: pluginsdk.DataOperatorEqual, Value: dataValue(pluginsdk.DataValueString, "product-tx")}
		page, queryErr := service.Query(tx.Context(), query)
		if queryErr != nil {
			return queryErr
		}
		if len(page.Records) != 1 {
			t.Fatalf("transaction page=%+v", page)
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("transaction error=%v", err)
	}
	page, err := service.Query(context.Background(), basicQuery())
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Records) != 0 {
		t.Fatalf("rolled back record remained: %+v", page)
	}
}
