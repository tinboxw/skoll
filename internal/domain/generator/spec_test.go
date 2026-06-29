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

func TestNewGeneratorSpecValidationRules(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*GeneratorSpecInput)
		wantErr string
	}{
		{
			name: "invalid module name",
			mutate: func(in *GeneratorSpecInput) {
				in.Module.Name = "bad-name"
			},
			wantErr: "module name",
		},
		{
			name: "invalid field type",
			mutate: func(in *GeneratorSpecInput) {
				in.Fields[1].Type = FieldType("money")
			},
			wantErr: "field type",
		},
		{
			name: "duplicate field name",
			mutate: func(in *GeneratorSpecInput) {
				in.Fields[2].Name = "name"
			},
			wantErr: "field name conflict",
		},
		{
			name: "duplicate column name",
			mutate: func(in *GeneratorSpecInput) {
				in.Fields[2].ColumnName = "product_name"
			},
			wantErr: "column name conflict",
		},
		{
			name: "index unknown field",
			mutate: func(in *GeneratorSpecInput) {
				in.Indexes[0].Fields = []string{"missing"}
			},
			wantErr: "unknown field",
		},
		{
			name: "invalid permission key",
			mutate: func(in *GeneratorSpecInput) {
				in.Permissions.ReadKey = "Product Read"
			},
			wantErr: "permission key",
		},
		{
			name: "menu permission unknown",
			mutate: func(in *GeneratorSpecInput) {
				in.Menu.RequiredPermissions = []string{"product.export"}
			},
			wantErr: "unknown permission",
		},
		{
			name: "page unknown field",
			mutate: func(in *GeneratorSpecInput) {
				in.Page.List.Columns = []string{"id", "missing"}
			},
			wantErr: "unknown field",
		},
		{
			name: "validation rule needs value",
			mutate: func(in *GeneratorSpecInput) {
				in.Fields[1].Validation = []ValidationRule{{Type: ValidationMax}}
			},
			wantErr: "validation value",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validGeneratorSpecInput()
			tc.mutate(&in)
			_, err := NewGeneratorSpec(in)
			if err == nil {
				t.Fatalf("expected error containing %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func validGeneratorSpecInput() GeneratorSpecInput {
	now := time.Now().UTC()
	return GeneratorSpecInput{
		ID: shared.ID("spec-product"),
		Module: ModuleSpec{
			Name:        "product",
			Package:     "product",
			DisplayName: "Product",
		},
		Table: TableSpec{
			Name:           "products",
			DomainName:     "Product",
			CollectionName: "Products",
		},
		Fields: []FieldSpec{
			{Name: "id", Label: "ID", Type: FieldTypeID, PrimaryKey: true, Required: true},
			{Name: "name", ColumnName: "product_name", Label: "Name", Type: FieldTypeString, Required: true, Validation: []ValidationRule{{Type: ValidationMax, Value: "128"}}},
			{Name: "price", Label: "Price", Type: FieldTypeDecimal, Required: true},
		},
		Indexes: []IndexSpec{
			{Name: "idx_products_name", Fields: []string{"name"}},
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
			Path:                "/skoll/products",
			Component:           "ProductList",
			RequiredPermissions: []string{"product.read"},
		},
		Page: PageSpec{
			Title:     "Products",
			RouteName: "product-list",
			List: PageListSpec{
				Columns: []string{"id", "name", "price"},
				Filters: []string{"name"},
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
	}
}
