package generator

import (
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestNewGeneratorSpecDefinesCRUDContract(t *testing.T) {
	now := time.Now().UTC()
	spec, err := NewGeneratorSpec(GeneratorSpecInput{
		ID: shared.ID("spec-product"),
		Module: ModuleSpec{
			Name:        " Product ",
			Package:     " Product ",
			DisplayName: "Product",
			Description: "Generated product CRUD",
		},
		Table: TableSpec{
			Name:           " products ",
			DomainName:     "Product",
			CollectionName: "Products",
			Comment:        "Product table",
		},
		Fields: []FieldSpec{
			{Name: " ID ", Label: "ID", Type: FieldTypeID, PrimaryKey: true, Required: true, ListVisible: true},
			{Name: " Name ", ColumnName: " product_name ", Label: "Name", Type: FieldTypeString, Required: true, Filterable: true, Sortable: true, ListVisible: true, FormVisible: true, Validation: []ValidationRule{{Type: ValidationMax, Value: "128"}}},
			{Name: " Price ", Label: "Price", Type: FieldTypeDecimal, Required: true, FormVisible: true},
		},
		Indexes: []IndexSpec{
			{Name: " idx_products_name ", Fields: []string{" Name "}},
		},
		Permissions: PermissionSpec{
			Resource:  "product",
			ReadKey:   "product.read",
			CreateKey: "product.create",
			UpdateKey: "product.update",
			DeleteKey: "product.delete",
			ManageKey: "product.manage",
		},
		Menu: MenuSpec{
			Key:                 "product",
			ParentKey:           "business",
			Path:                "/skoll/products",
			Component:           "ProductList",
			Icon:                "package",
			Order:               100,
			RequiredPermissions: []string{" product.read "},
		},
		Page: PageSpec{
			Title:     "Products",
			RouteName: "product-list",
			List: PageListSpec{
				Columns: []string{"id", "name", "price"},
				Filters: []string{"name"},
				Actions: []string{"create", "update", "delete"},
			},
			Form: PageFormSpec{
				Fields: []string{"name", "price"},
				Mode:   "drawer",
			},
		},
		Audit: AuditSpec{
			Resource: "product",
			Actions:  []string{"create", "update", "delete"},
		},
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("NewGeneratorSpec error: %v", err)
	}

	if spec.Module.Name != "product" || spec.Module.Package != "product" {
		t.Fatalf("expected normalized module, got %+v", spec.Module)
	}
	if spec.Table.Name != "products" {
		t.Fatalf("expected normalized table name, got %q", spec.Table.Name)
	}
	if spec.Fields[0].ColumnName != "id" {
		t.Fatalf("expected default column name from field name, got %q", spec.Fields[0].ColumnName)
	}
	field, ok := spec.FieldByName(" NAME ")
	if !ok || field.ColumnName != "product_name" {
		t.Fatalf("expected field lookup by normalized name, got field=%+v ok=%v", field, ok)
	}
	if got := strings.Join(spec.FieldNames(), ","); got != "id,name,price" {
		t.Fatalf("unexpected field names: %s", got)
	}
	if spec.Menu.RequiredPermissions[0] != "product.read" {
		t.Fatalf("expected normalized permission key, got %q", spec.Menu.RequiredPermissions[0])
	}
	if spec.Meta.CreatedAt.IsZero() || !spec.Meta.UpdatedAt.Equal(spec.Meta.CreatedAt) {
		t.Fatalf("expected meta timestamps copied from input, got %+v", spec.Meta)
	}
}

func TestNewGeneratorSpecRejectsIncompleteSections(t *testing.T) {
	now := time.Now().UTC()
	_, err := NewGeneratorSpec(GeneratorSpecInput{
		ID:        shared.ID("spec-incomplete"),
		Module:    ModuleSpec{Name: "product", Package: "product", DisplayName: "Product"},
		Table:     TableSpec{Name: "products", DomainName: "Product", CollectionName: "Products"},
		Fields:    []FieldSpec{{Name: "id", Label: "ID", Type: FieldTypeID}},
		CreatedAt: now,
	})
	if err == nil {
		t.Fatalf("expected incomplete permission/menu/page/audit error")
	}
	if !strings.Contains(err.Error(), "permission") {
		t.Fatalf("unexpected error: %v", err)
	}
}
