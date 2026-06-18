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
}

func containsModel[T any](models []any) bool {
	for _, model := range models {
		if _, ok := model.(*T); ok {
			return true
		}
	}
	return false
}
