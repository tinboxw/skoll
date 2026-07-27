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
	if !containsModel[PluginMigrationModel](models) {
		t.Fatal("AllModels() should include PluginMigrationModel")
	}
	if !containsModel[PluginDataMutationModel](models) {
		t.Fatal("AllModels() should include PluginDataMutationModel")
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
		"WorkflowDefinitionModel":   containsModel[WorkflowDefinitionModel](models),
		"WorkflowNodeModel":         containsModel[WorkflowNodeModel](models),
		"WorkflowNodeAssigneeModel": containsModel[WorkflowNodeAssigneeModel](models),
		"WorkflowTransitionModel":   containsModel[WorkflowTransitionModel](models),
		"WorkflowInstanceModel":     containsModel[WorkflowInstanceModel](models),
		"WorkflowTaskModel":         containsModel[WorkflowTaskModel](models),
		"WorkflowActionModel":       containsModel[WorkflowActionModel](models),
	} {
		if !present {
			t.Fatalf("AllModels() should include %s", name)
		}
	}
	if !containsModel[JobModel](models) {
		t.Fatal("AllModels() should include JobModel")
	}
	for name, present := range map[string]bool{
		"NotificationItemModel":            containsModel[NotificationItemModel](models),
		"NotificationReminderRuleModel":    containsModel[NotificationReminderRuleModel](models),
		"NotificationDeliveryAttemptModel": containsModel[NotificationDeliveryAttemptModel](models),
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
