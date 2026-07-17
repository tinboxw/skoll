package gormrepo

import "testing"

func TestAllModelsIncludesPermissionAndMenuModels(t *testing.T) {
	models := AllModels()

	if !containsModel[PermissionResourceModel](models) {
		t.Fatal("AllModels() should include PermissionResourceModel")
	}
	if !containsModel[MenuNodeModel](models) {
		t.Fatal("AllModels() should include MenuNodeModel")
	}
	if !containsModel[AuditEventModel](models) {
		t.Fatal("AllModels() should include AuditEventModel")
	}
	if !containsModel[FileObjectModel](models) {
		t.Fatal("AllModels() should include FileObjectModel")
	}
	if !containsModel[DictionaryTypeModel](models) {
		t.Fatal("AllModels() should include DictionaryTypeModel")
	}
	if !containsModel[DictionaryItemModel](models) {
		t.Fatal("AllModels() should include DictionaryItemModel")
	}
	for name, present := range map[string]bool{
		"PharmaEmployeeModel":  containsModel[PharmaEmployeeModel](models),
		"PharmaProductModel":   containsModel[PharmaProductModel](models),
		"PharmaSupplierModel":  containsModel[PharmaSupplierModel](models),
		"PharmaCustomerModel":  containsModel[PharmaCustomerModel](models),
		"PharmaWarehouseModel": containsModel[PharmaWarehouseModel](models),
	} {
		if !present {
			t.Fatalf("AllModels() should include %s", name)
		}
	}
}

func containsModel[T any](models []any) bool {
	for _, model := range models {
		if _, ok := model.(*T); ok {
			return true
		}
	}
	return false
}
