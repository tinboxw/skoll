package gormrepo

import (
	"log"
	"os"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestDB provides a fresh SQLite in-memory database for each test
func TestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		if isSQLiteCGODisabledError(err) {
			t.Skipf("sqlite test requires cgo: %v", err)
		}
		t.Fatalf("failed to open test database: %v", err)
	}

	// Auto migrate all models
	err = db.AutoMigrate(
		UserModel{},
		RoleModel{},
		BindingModel{},
		PolicyRuleModel{},
		AuditRecordModel{},
		SystemSettingModel{},
		PluginModel{},
		PluginMigrationModel{},
		PluginReleaseModel{},
		PluginRouteModel{},
		PermissionResourceModel{},
		MenuNodeModel{},
		FileObjectModel{},
		DictionaryTypeModel{},
		DictionaryItemModel{},
		DepartmentModel{},
		PositionModel{},
		UserAssignmentModel{},
		AuditEventModel{},
		PharmaEmployeeModel{},
		PharmaProductModel{},
		PharmaSupplierModel{},
		PharmaCustomerModel{},
		PharmaWarehouseModel{},
		PharmaStockBatchModel{}, PharmaStockBalanceModel{}, PharmaStockLedgerModel{}, PharmaStockLockModel{},
		PharmaPurchaseRequestModel{}, PharmaPurchaseOrderModel{}, PharmaPurchaseInboundModel{}, PharmaSalesOrderModel{}, PharmaSalesOutboundModel{}, PharmaStocktakeModel{}, PharmaTransferModel{},
		PharmaAnnouncementModel{}, PharmaColdChainRecordModel{},
		PharmaContractModel{}, PharmaQualityComplaintModel{}, PharmaDrugRecallModel{}, PharmaCustomerFollowUpModel{}, PharmaSalesOpportunityModel{},
		PharmaPaymentPlanModel{}, PharmaInvoiceRecordModel{}, PharmaPaymentReminderJobModel{}, PharmaInventoryAlertModel{}, PharmaInventoryAlertJobModel{}, PharmaReportExportJobModel{},
	)
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

// SetupTestDBWithLogger creates a test database with SQL logging (useful for debugging)
func SetupTestDBWithLogger(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	// Enable GORM SQL logging for debugging
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		if isSQLiteCGODisabledError(err) {
			t.Skipf("sqlite test requires cgo: %v", err)
		}
		t.Fatalf("failed to open test database: %v", err)
	}

	// Redirect GORM logs to test log
	db.Logger = logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// Auto migrate all models
	err = db.AutoMigrate(
		UserModel{},
		RoleModel{},
		BindingModel{},
		PolicyRuleModel{},
		AuditRecordModel{},
		SystemSettingModel{},
		PluginModel{},
		PluginMigrationModel{},
		PluginReleaseModel{},
		PluginRouteModel{},
		PermissionResourceModel{},
		MenuNodeModel{},
		FileObjectModel{},
		DictionaryTypeModel{},
		DictionaryItemModel{},
		DepartmentModel{},
		PositionModel{},
		UserAssignmentModel{},
		AuditEventModel{},
		PharmaEmployeeModel{},
		PharmaProductModel{},
		PharmaSupplierModel{},
		PharmaCustomerModel{},
		PharmaWarehouseModel{},
		PharmaStockBatchModel{}, PharmaStockBalanceModel{}, PharmaStockLedgerModel{}, PharmaStockLockModel{},
		PharmaPurchaseRequestModel{}, PharmaPurchaseOrderModel{}, PharmaPurchaseInboundModel{}, PharmaSalesOrderModel{}, PharmaSalesOutboundModel{}, PharmaStocktakeModel{}, PharmaTransferModel{},
		PharmaAnnouncementModel{}, PharmaColdChainRecordModel{},
		PharmaContractModel{}, PharmaQualityComplaintModel{}, PharmaDrugRecallModel{}, PharmaCustomerFollowUpModel{}, PharmaSalesOpportunityModel{},
		PharmaPaymentPlanModel{}, PharmaInvoiceRecordModel{}, PharmaPaymentReminderJobModel{}, PharmaInventoryAlertModel{}, PharmaInventoryAlertJobModel{}, PharmaReportExportJobModel{},
	)
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	cleanup := func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}

	return db, cleanup
}

func isSQLiteCGODisabledError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "requires cgo") || strings.Contains(msg, "cgo_enabled=0")
}
