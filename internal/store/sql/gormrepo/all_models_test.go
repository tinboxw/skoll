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
	for name, present := range map[string]bool{
		"PharmaEmployeeModel":           containsModel[PharmaEmployeeModel](models),
		"PharmaProductModel":            containsModel[PharmaProductModel](models),
		"PharmaSupplierModel":           containsModel[PharmaSupplierModel](models),
		"PharmaCustomerModel":           containsModel[PharmaCustomerModel](models),
		"PharmaWarehouseModel":          containsModel[PharmaWarehouseModel](models),
		"PharmaStockBatchModel":         containsModel[PharmaStockBatchModel](models),
		"PharmaStockBalanceModel":       containsModel[PharmaStockBalanceModel](models),
		"PharmaStockLedgerModel":        containsModel[PharmaStockLedgerModel](models),
		"PharmaPurchaseRequestModel":    containsModel[PharmaPurchaseRequestModel](models),
		"PharmaPurchaseOrderModel":      containsModel[PharmaPurchaseOrderModel](models),
		"PharmaPurchaseInboundModel":    containsModel[PharmaPurchaseInboundModel](models),
		"PharmaSalesOrderModel":         containsModel[PharmaSalesOrderModel](models),
		"PharmaSalesOutboundModel":      containsModel[PharmaSalesOutboundModel](models),
		"PharmaStocktakeModel":          containsModel[PharmaStocktakeModel](models),
		"PharmaTransferModel":           containsModel[PharmaTransferModel](models),
		"PharmaAnnouncementModel":       containsModel[PharmaAnnouncementModel](models),
		"PharmaColdChainRecordModel":    containsModel[PharmaColdChainRecordModel](models),
		"PharmaContractModel":           containsModel[PharmaContractModel](models),
		"PharmaQualityComplaintModel":   containsModel[PharmaQualityComplaintModel](models),
		"PharmaDrugRecallModel":         containsModel[PharmaDrugRecallModel](models),
		"PharmaCustomerFollowUpModel":   containsModel[PharmaCustomerFollowUpModel](models),
		"PharmaSalesOpportunityModel":   containsModel[PharmaSalesOpportunityModel](models),
		"PharmaPaymentPlanModel":        containsModel[PharmaPaymentPlanModel](models),
		"PharmaInvoiceRecordModel":      containsModel[PharmaInvoiceRecordModel](models),
		"PharmaPaymentReminderJobModel": containsModel[PharmaPaymentReminderJobModel](models),
		"PharmaInventoryAlertModel":     containsModel[PharmaInventoryAlertModel](models),
		"PharmaInventoryAlertJobModel":  containsModel[PharmaInventoryAlertJobModel](models),
		"PharmaReportExportJobModel":    containsModel[PharmaReportExportJobModel](models),
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
