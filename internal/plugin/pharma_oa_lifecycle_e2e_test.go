package plugin

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"github.com/tinboxw/skoll/pkg/security"
)

func TestPharmaOAPackagedBusinessLifecycleE2E(t *testing.T) {
	if testing.Short() || os.Getenv("SKOLL_PHARMA_OA_E2E") != "1" {
		t.Skip("set SKOLL_PHARMA_OA_E2E=1 to run the packaged Pharma OA business lifecycle E2E")
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	webModules := filepath.Join(repoRoot, "web", "node_modules")
	if info, statErr := os.Stat(webModules); statErr != nil || !info.IsDir() {
		t.Skip("web/node_modules is required for pharma-OA lifecycle E2E")
	}

	hostBackendBefore := equipmentSourceHashes(t, filepath.Join(repoRoot, "internal", "bootstrap"))
	hostFrontendBefore := equipmentSourceHashes(t, filepath.Join(repoRoot, "web", "src"))
	t.Cleanup(func() {
		if after := equipmentSourceHashes(t, filepath.Join(repoRoot, "internal", "bootstrap")); strings.Join(hostBackendBefore, "\n") != strings.Join(after, "\n") {
			t.Errorf("pharma-OA lifecycle changed host backend production files")
		}
		if after := equipmentSourceHashes(t, filepath.Join(repoRoot, "web", "src")); strings.Join(hostFrontendBefore, "\n") != strings.Join(after, "\n") {
			t.Errorf("pharma-OA lifecycle changed host frontend production files")
		}
	})

	address := reserveEquipmentAddress(t)
	source := filepath.Join(t.TempDir(), "pharma_oa")
	copyEquipmentSource(t, filepath.Join(repoRoot, "plugins", "pharma_oa"), source)
	rewritePharmaAddress(t, filepath.Join(source, "plugin.yaml"), address)
	linkEquipmentNodeModules(t, webModules, filepath.Join(source, "frontend", "node_modules"))
	before := equipmentSourceHashes(t, source)
	buildPharmaPackageSurface(t, repoRoot, source)
	after := equipmentSourceHashes(t, source)
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("build changed plugin source\nbefore=%v\nafter=%v", before, after)
	}

	result, err := BuildPackage(source, filepath.Join(t.TempDir(), "dist"), NewFileLoader())
	if err != nil {
		t.Fatalf("build package: %v", err)
	}
	if verified, verifyErr := VerifyPackage(result.ArtifactPath, result.ChecksumPath); verifyErr != nil || verified != result.SHA256 {
		t.Fatalf("verify package digest=%q want=%q err=%v", verified, result.SHA256, verifyErr)
	}
	installedDir, packageInfo, err := InstallPackage(result.ArtifactPath, result.ChecksumPath, filepath.Join(t.TempDir(), "plugins"), NewFileLoader())
	if err != nil {
		t.Fatalf("install package: %v", err)
	}
	required := []string{"plugin.yaml", "frontend/dist/index.html", managedBackendRelativePath("pharma_oa")}
	for version, name := range map[string]string{
		"001": "foundation", "002": "employees", "003": "parties", "004": "catalogs",
		"005": "qualifications", "006": "oa_requests", "007": "purchases", "008": "purchase_inbounds",
	} {
		required = append(required, "migrations/"+version+"_"+name+".up.sql", "migrations/"+version+"_"+name+".down.sql")
	}
	for _, path := range required {
		if _, statErr := os.Stat(filepath.Join(installedDir, filepath.FromSlash(path))); statErr != nil {
			t.Fatalf("installed package missing %s: %v", path, statErr)
		}
	}

	const jwtSecret = "pharma-oa-lifecycle-secret"
	transactions := &gatewayTransactions{}
	store := newPharmaLifecycleStore()
	audit := &pharmaLifecycleAudit{}
	files := newPharmaLifecycleFiles()
	jobs := newPharmaLifecycleJobs()
	workflows := newPharmaLifecycleWorkflows()
	host := pluginsdk.HostServices{
		PluginID: "pharma_oa", Transactions: transactions, DataScopes: pharmaLifecycleScopes{}, DataStore: store,
		Files: files, DocumentNumbers: gatewayDocumentNumbers{}, Documents: gatewayDocuments{}, Audit: audit,
		Config: gatewayConfig{}, Secrets: &gatewaySecrets{}, Workflows: workflows, Jobs: jobs,
	}
	gateway, err := NewHostGateway(func(pluginID string) (pluginsdk.HostServices, error) {
		if pluginID != host.PluginID {
			return pluginsdk.HostServices{}, errors.New("unexpected plugin identity")
		}
		return host, nil
	}, jwtSecret, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = gateway.Close() })

	dataDirectories, err := NewPluginDataDirectories(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}
	manager := NewRuntimeManager(NewFileLoader(), NewTopologicalResolver())
	installed, err := manager.Install(installedDir)
	if err != nil {
		t.Fatalf("runtime install: %v", err)
	}
	if installed.ID != packageInfo.ID || installed.State != StateInstalled {
		t.Fatalf("unexpected installed state: %+v package=%+v", installed, packageInfo)
	}
	migrationStore := newTestMigrationStore()
	migrationHook := NewPluginMigrationHook(migrationStore, nil)
	if _, err := migrationHook.Run(context.Background(), PluginMigrationHookInput{
		PluginID: installed.ID, PluginDir: installedDir, MigrationDirectory: installed.DataManifest.MigrationDirectory,
		Action: PluginMigrationInstall, ToVersion: installed.DataManifest.MigrationVersion,
	}); err != nil {
		t.Fatalf("install migration: %v", err)
	}
	if len(migrationStore.records) != 8 {
		t.Fatalf("install migration ledger=%+v", migrationStore.records)
	}
	for index, record := range migrationStore.records {
		if record.PluginID != installed.ID || record.Version != index+1 {
			t.Fatalf("migration ledger[%d]=%+v", index, record)
		}
	}

	launcher := NewManagedProcessLauncher(NewHTTPHealthChecker(time.Second), 50*time.Millisecond, gateway, dataDirectories)
	supervisor := NewServiceSupervisor(launcher, nil, 5*time.Second, 2*time.Second)
	t.Cleanup(func() { _ = supervisor.Shutdown(context.Background()) })
	if err := manager.Enable(installed.ID); err != nil {
		t.Fatalf("enable plugin: %v", err)
	}
	if err := supervisor.Start(context.Background(), mustPharmaInfo(t, manager)); err != nil {
		t.Fatalf("start packaged backend: %v", err)
	}

	actorAToken := signPharmaLifecycleToken(t, jwtSecret, "actor-a", "org-a")
	actorBToken := signPharmaLifecycleToken(t, jwtSecret, "actor-b", "org-b")
	baseURL := "http://" + address + "/v1/plugins/pharma_oa/api"
	employee := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/employees", actorAToken, "e2e-employee-create", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "EMP-E2E-001", "name": "李华", "departmentId": "quality", "positionId": "director", "phone": "13800000000", "email": "lihua@example.com", "hireDate": "2026-01-02",
	}, http.StatusCreated)
	if pharmaLifecycleString(t, pharmaLifecycleMap(t, employee, "item"), "employmentStatus") != "active" {
		t.Fatalf("unexpected employee response: %v", employee)
	}

	customer := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/customers", actorAToken, "e2e-customer-create", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "CUS-E2E-001", "name": "华东合规药房", "unifiedSocialCreditCode": "91330000MAE2E001", "region": "浙江省杭州市", "rating": 5,
		"contacts":        []map[string]any{{"name": "质量负责人", "phone": "13800000001", "email": "qa@example.com", "primary": true}},
		"addresses":       []map[string]any{{"label": "总部", "province": "浙江省", "city": "杭州市", "district": "拱墅区", "detail": "康桥路 8 号", "default": true}},
		"settlementTerms": map[string]any{"currency": "CNY", "paymentDays": 30, "creditLimit": 100000},
	}, http.StatusCreated)
	customerItem := pharmaLifecycleMap(t, customer, "item")
	customerID := pharmaLifecycleString(t, customerItem, "id")

	typeResponse := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/qualification-types", actorAToken, "e2e-qualification-type-create", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "DRUG-BUSINESS-E2E", "name": "药品经营许可证", "subjectType": "customer", "businessGate": "sales", "description": "客户销售业务准入证照", "validityDays": 365, "alertDays": 30, "evidenceRequired": true, "businessRequired": true,
	}, http.StatusCreated)
	typeID := pharmaLifecycleString(t, pharmaLifecycleMap(t, typeResponse, "item"), "id")
	now := time.Now().UTC()
	qualification := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/qualifications", actorAToken, "e2e-qualification-create", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "typeId": typeID, "subjectType": "customer", "subjectId": customerID, "certificateNumber": "浙药经许-E2E-001", "issuer": "浙江省药品监督管理局",
		"validFrom": now.AddDate(0, 0, -30).Format("2006-01-02"), "validTo": now.AddDate(0, 0, 10).Format("2006-01-02"),
		"evidence": map[string]any{"name": "license.pdf", "contentBase64": base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\n%%EOF"))},
	}, http.StatusCreated)
	qualificationItem := pharmaLifecycleMap(t, qualification, "item")
	qualificationID := pharmaLifecycleString(t, qualificationItem, "id")

	eligibility := pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/customers/"+customerID+"/sales-eligibility", actorAToken, "", nil, http.StatusOK)
	if eligible, ok := eligibility["eligible"].(bool); !ok || eligible {
		t.Fatalf("draft qualification unexpectedly passed sales gate: %v", eligibility)
	}
	pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/qualifications/"+qualificationID+"/submit", actorAToken, "e2e-qualification-submit", map[string]any{"version": 1}, http.StatusOK)
	approved := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/qualifications/"+qualificationID+"/approve", actorAToken, "e2e-qualification-approve", map[string]any{"comment": "证照真实有效", "version": 2}, http.StatusOK)
	if pharmaLifecycleString(t, pharmaLifecycleMap(t, approved, "item"), "status") != "approved" {
		t.Fatalf("qualification was not approved: %v", approved)
	}
	eligibility = pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/customers/"+customerID+"/sales-eligibility", actorAToken, "", nil, http.StatusOK)
	if eligible, ok := eligibility["eligible"].(bool); !ok || !eligible {
		t.Fatalf("approved qualification did not pass sales gate: %v", eligibility)
	}
	scan := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/qualifications/expiry-scan", actorAToken, "e2e-expiry-scan", map[string]any{}, http.StatusOK)
	if pharmaLifecycleInt(t, scan, "scheduled") != 1 {
		t.Fatalf("expiry scan did not schedule alert: %v", scan)
	}

	disabled := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/customers/"+customerID+"/disable", actorAToken, "e2e-customer-disable", map[string]any{"reason": "暂停合作", "version": 1}, http.StatusOK)
	if pharmaLifecycleString(t, pharmaLifecycleMap(t, disabled, "item"), "status") != "disabled" {
		t.Fatalf("customer was not disabled: %v", disabled)
	}
	pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/customers/"+customerID+"/sales-eligibility", actorAToken, "", nil, http.StatusUnprocessableEntity)
	linked := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/customers/"+customerID+"/enable", actorAToken, "e2e-customer-enable", map[string]any{"version": 2}, http.StatusOK)
	if pharmaLifecycleString(t, pharmaLifecycleMap(t, linked, "item"), "status") != "active" {
		t.Fatalf("customer was not re-enabled: %v", linked)
	}

	isolatedCustomers := pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/customers?limit=20", actorBToken, "", nil, http.StatusOK)
	isolatedEmployees := pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/employees?limit=20", actorBToken, "", nil, http.StatusOK)
	if pharmaLifecycleInt(t, isolatedCustomers, "total") != 0 || pharmaLifecycleInt(t, isolatedEmployees, "total") != 0 {
		t.Fatalf("cross-scope records leaked: customers=%v employees=%v", isolatedCustomers, isolatedEmployees)
	}

	leave := pharmaLifecycleCreateOARequest(t, baseURL, actorAToken, "leave", "Annual leave request", map[string]any{
		"leaveType": "annual", "startDate": "2026-08-03", "endDate": "2026-08-05",
	})
	leaveID := pharmaLifecycleString(t, leave, "id")
	leaveSubmit := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/oa-requests/"+leaveID+"/submit", actorAToken, "e2e-oa-leave-submit", map[string]any{"version": 1}, http.StatusOK)
	leaveTaskID := pharmaLifecyclePendingTaskID(t, leaveSubmit)
	leaveAttachment := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/oa-requests/"+leaveID+"/attachments", actorAToken, "e2e-oa-leave-attachment", map[string]any{
		"name": "leave-note.txt", "contentBase64": base64.StdEncoding.EncodeToString([]byte("medical OA lifecycle attachment")), "version": 2,
	}, http.StatusOK)
	leaveComment := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/oa-requests/"+leaveID+"/comments", actorAToken, "e2e-oa-leave-comment", map[string]any{
		"content": "Please review before the weekly roster closes.", "version": pharmaLifecycleInt(t, pharmaLifecycleMap(t, leaveAttachment, "item"), "version"),
	}, http.StatusOK)
	leaveReminder := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/oa-requests/"+leaveID+"/reminders", actorAToken, "e2e-oa-leave-reminder", map[string]any{
		"runAt": time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339), "version": pharmaLifecycleInt(t, pharmaLifecycleMap(t, leaveComment, "item"), "version"),
	}, http.StatusOK)
	leaveDelegated := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/oa-requests/"+leaveID+"/delegate", actorAToken, "e2e-oa-leave-delegate", map[string]any{
		"taskId": leaveTaskID, "targetId": "actor-a-delegate", "targetName": "Backup approver", "comment": "Delegated during business travel",
		"version": pharmaLifecycleInt(t, pharmaLifecycleMap(t, leaveReminder, "item"), "version"),
	}, http.StatusOK)
	delegatedTaskID := pharmaLifecyclePendingTaskID(t, leaveDelegated)
	leaveApproved := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/oa-requests/"+leaveID+"/approve", actorAToken, "e2e-oa-leave-approve", map[string]any{
		"taskId": delegatedTaskID, "comment": "Roster coverage confirmed", "version": pharmaLifecycleInt(t, pharmaLifecycleMap(t, leaveDelegated, "item"), "version"),
	}, http.StatusOK)
	pharmaLifecycleAssertOAStatus(t, leaveApproved, "approved")

	expense := pharmaLifecycleCreateOARequest(t, baseURL, actorAToken, "expense", "Conference travel expense", map[string]any{
		"amount": 1860.50, "category": "travel",
	})
	expenseSubmit := pharmaLifecycleSubmitOARequest(t, baseURL, actorAToken, expense, "expense")
	expenseRejected := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/oa-requests/"+pharmaLifecycleString(t, expense, "id")+"/reject", actorAToken, "e2e-oa-expense-reject", map[string]any{
		"taskId": pharmaLifecyclePendingTaskID(t, expenseSubmit), "comment": "Missing original invoice", "version": 2,
	}, http.StatusOK)
	pharmaLifecycleAssertOAStatus(t, expenseRejected, "rejected")

	procurement := pharmaLifecycleCreateOARequest(t, baseURL, actorAToken, "procurement", "Cold-chain data logger purchase", map[string]any{
		"amount": 9600, "purpose": "Replace calibration-expired warehouse loggers",
	})
	pharmaLifecycleSubmitOARequest(t, baseURL, actorAToken, procurement, "procurement")
	procurementWithdrawn := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/oa-requests/"+pharmaLifecycleString(t, procurement, "id")+"/withdraw", actorAToken, "e2e-oa-procurement-withdraw", map[string]any{
		"comment": "Specification requires revision", "version": 2,
	}, http.StatusOK)
	pharmaLifecycleAssertOAStatus(t, procurementWithdrawn, "withdrawn")

	contract := pharmaLifecycleCreateOARequest(t, baseURL, actorAToken, "contract", "Regional distribution agreement", map[string]any{
		"amount": 250000, "counterparty": "East Region Pharmacy Group", "effectiveDate": "2026-09-01",
	})
	pharmaLifecycleSubmitOARequest(t, baseURL, actorAToken, contract, "contract")
	contractCanceled := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/oa-requests/"+pharmaLifecycleString(t, contract, "id")+"/cancel", actorAToken, "e2e-oa-contract-cancel", map[string]any{
		"comment": "Counterparty withdrew before approval", "version": 2,
	}, http.StatusOK)
	pharmaLifecycleAssertOAStatus(t, contractCanceled, "canceled")

	custom := pharmaLifecycleCreateOARequest(t, baseURL, actorAToken, "custom", "GSP training room booking", map[string]any{
		"formKey": "training_room_booking",
	})
	customSubmit := pharmaLifecycleSubmitOARequest(t, baseURL, actorAToken, custom, "custom")
	customApproved := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/oa-requests/"+pharmaLifecycleString(t, custom, "id")+"/approve", actorAToken, "e2e-oa-custom-approve", map[string]any{
		"taskId": pharmaLifecyclePendingTaskID(t, customSubmit), "comment": "Room and compliance trainer are available", "version": 2,
	}, http.StatusOK)
	pharmaLifecycleAssertOAStatus(t, customApproved, "approved")

	supplierID, productID, manufacturerID := pharmaLifecycleCreatePurchaseMasterData(t, baseURL, actorAToken)
	pharmaLifecycleApprovePurchaseQualification(t, baseURL, actorAToken, "supplier", supplierID, "purchase", "supplier")
	pharmaLifecycleApprovePurchaseQualification(t, baseURL, actorAToken, "product", productID, "purchase", "product")
	pharmaLifecycleApprovePurchaseQualification(t, baseURL, actorAToken, "manufacturer", manufacturerID, "supply", "manufacturer")
	purchaseBody := map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "supplierId": supplierID,
		"reason": "Packaged purchase and receiving acceptance", "currency": "CNY", "approverId": "actor-a-approver", "approverName": "Purchase approver",
		"lines": []map[string]any{{"productId": productID, "quantity": "2.5", "unitPrice": "10.20"}},
	}
	purchaseCreated := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/purchase-requests", actorAToken, "e2e-purchase-create", purchaseBody, http.StatusCreated)
	purchaseItem := pharmaLifecycleMap(t, purchaseCreated, "item")
	purchaseID := pharmaLifecycleString(t, purchaseItem, "id")
	purchaseDuplicate := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/purchase-requests", actorAToken, "e2e-purchase-create", purchaseBody, http.StatusOK)
	if duplicate, ok := purchaseDuplicate["duplicate"].(bool); !ok || !duplicate ||
		pharmaLifecycleString(t, pharmaLifecycleMap(t, purchaseDuplicate, "item"), "id") != purchaseID {
		t.Fatalf("purchase create was not idempotent: %v", purchaseDuplicate)
	}
	purchaseApproved := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/purchase-requests/"+purchaseID+"/approve", actorAToken, "e2e-purchase-approve", map[string]any{
		"taskId": pharmaLifecyclePendingTaskID(t, purchaseCreated), "comment": "Qualifications and budget confirmed", "version": 1,
	}, http.StatusOK)
	purchaseOrder := pharmaLifecycleMap(t, purchaseApproved, "order")
	purchaseOrderID := pharmaLifecycleString(t, purchaseOrder, "id")
	purchaseOrderLine := pharmaLifecycleArrayMap(t, purchaseOrder, "lines", 0)
	purchaseOrderLineID := pharmaLifecycleString(t, purchaseOrderLine, "id")
	if pharmaLifecycleString(t, purchaseOrder, "status") != "open" || pharmaLifecycleString(t, purchaseOrder, "totalAmount") != "25.50" {
		t.Fatalf("approved purchase did not create exact open order: %v", purchaseApproved)
	}

	rejectedPurchase := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/purchase-requests", actorAToken, "e2e-purchase-reject-create", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "supplierId": supplierID,
		"reason": "Duplicate replenishment request", "currency": "CNY", "approverId": "actor-a-approver", "approverName": "Purchase approver",
		"lines": []map[string]any{{"productId": productID, "quantity": "1", "unitPrice": "10.20"}},
	}, http.StatusCreated)
	rejectedPurchaseItem := pharmaLifecycleMap(t, rejectedPurchase, "item")
	rejectedPurchaseID := pharmaLifecycleString(t, rejectedPurchaseItem, "id")
	rejectedDecision := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/purchase-requests/"+rejectedPurchaseID+"/reject", actorAToken, "e2e-purchase-reject", map[string]any{
		"taskId": pharmaLifecyclePendingTaskID(t, rejectedPurchase), "comment": "Duplicate request", "version": 1,
	}, http.StatusOK)
	if pharmaLifecycleString(t, pharmaLifecycleMap(t, rejectedDecision, "item"), "status") != "rejected" {
		t.Fatalf("purchase rejection did not close request: %v", rejectedDecision)
	}

	partialInboundBody := map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "purchaseOrderId": purchaseOrderID,
		"warehouseId": "WH-E2E", "areaId": "AREA-A", "locationId": "LOC-001", "orderVersion": 1,
		"lines": []map[string]any{{
			"orderLineId": purchaseOrderLineID, "quantity": "1.25", "batchNo": "LOT-E2E-001",
			"productionDate": now.AddDate(0, -1, 0).Format("2006-01-02"), "expiresAt": now.AddDate(2, 0, 0).Format("2006-01-02"),
		}},
		"attachments": []map[string]any{{
			"name": "receipt-evidence.txt", "contentBase64": base64.StdEncoding.EncodeToString([]byte("packaged inbound evidence")),
		}},
	}
	partialInbound := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/purchase-inbounds", actorAToken, "e2e-inbound-partial", partialInboundBody, http.StatusCreated)
	partialInboundItem := pharmaLifecycleMap(t, partialInbound, "item")
	partialInboundID := pharmaLifecycleString(t, partialInboundItem, "id")
	partialOrder := pharmaLifecycleMap(t, partialInbound, "order")
	if pharmaLifecycleString(t, partialOrder, "status") != "partial" ||
		pharmaLifecycleString(t, pharmaLifecycleArrayMap(t, partialOrder, "lines", 0), "receivedQuantity") != "1.25" {
		t.Fatalf("partial inbound did not retain exact order progress: %v", partialInbound)
	}

	isolatedOARequests := pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/oa-requests?limit=20", actorBToken, "", nil, http.StatusOK)
	if pharmaLifecycleInt(t, isolatedOARequests, "total") != 0 {
		t.Fatalf("cross-scope OA requests leaked: %v", isolatedOARequests)
	}
	pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/oa-requests/"+leaveID, actorBToken, "", nil, http.StatusNotFound)
	isolatedPurchaseRequests := pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/purchase-requests", actorBToken, "", nil, http.StatusOK)
	isolatedPurchaseOrders := pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/purchase-orders", actorBToken, "", nil, http.StatusOK)
	isolatedInbounds := pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/purchase-inbounds", actorBToken, "", nil, http.StatusOK)
	if pharmaLifecycleInt(t, isolatedPurchaseRequests, "total") != 0 || pharmaLifecycleInt(t, isolatedPurchaseOrders, "total") != 0 ||
		pharmaLifecycleInt(t, isolatedInbounds, "total") != 0 {
		t.Fatalf("cross-scope purchase records leaked: requests=%v orders=%v inbounds=%v", isolatedPurchaseRequests, isolatedPurchaseOrders, isolatedInbounds)
	}
	pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/purchase-requests/"+purchaseID, actorBToken, "", nil, http.StatusNotFound)
	pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/purchase-orders/"+purchaseOrderID, actorBToken, "", nil, http.StatusNotFound)
	pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/purchase-inbounds/"+partialInboundID, actorBToken, "", nil, http.StatusNotFound)
	if files.count() != 6 || jobs.count() != 2 || workflows.instanceCount() != 7 {
		t.Fatalf("unexpected OA host resources: files=%d jobs=%d workflows=%d", files.count(), jobs.count(), workflows.instanceCount())
	}
	for _, action := range []string{
		"pharma_oa.employee.create", "pharma_oa.customer.create", "pharma_oa.qualification_type.create", "pharma_oa.qualification.create",
		"pharma_oa.qualification.submit", "pharma_oa.qualification.approve", "pharma_oa.qualification.expiry_scan", "pharma_oa.customer.disable", "pharma_oa.customer.enable",
		"pharma_oa.oa_request.create", "pharma_oa.oa_request.submit", "pharma_oa.oa_request.attach", "pharma_oa.oa_request.comment",
		"pharma_oa.oa_request.remind", "pharma_oa.oa_request.delegate", "pharma_oa.oa_request.approve", "pharma_oa.oa_request.reject",
		"pharma_oa.oa_request.withdraw", "pharma_oa.oa_request.cancel",
		"pharma_oa.purchase.create", "pharma_oa.purchase.approve", "pharma_oa.purchase.reject", "pharma_oa.inbound.create",
	} {
		if !audit.hasAction(action) {
			t.Fatalf("missing audit action %q; actions=%v", action, audit.actions())
		}
	}
	if transactions.commits < 40 || transactions.rollbacks != 0 {
		t.Fatalf("unexpected transaction results: %+v", transactions)
	}

	if err := manager.Disable(installed.ID); err != nil {
		t.Fatalf("disable plugin: %v", err)
	}
	if err := supervisor.Stop(context.Background(), installed.ID); err != nil {
		t.Fatalf("stop plugin: %v", err)
	}
	if snapshot, ok := supervisor.Snapshot(installed.ID); !ok || snapshot.State != ServiceStateStopped {
		t.Fatalf("disabled service snapshot=%+v exists=%v", snapshot, ok)
	}
	if err := equipmentEndpointClosed(baseURL + "/customers"); err != nil {
		t.Fatal(err)
	}
	if err := equipmentEndpointClosed(baseURL + "/oa-requests"); err != nil {
		t.Fatal(err)
	}
	if err := equipmentEndpointClosed(baseURL + "/purchase-requests"); err != nil {
		t.Fatal(err)
	}
	if err := manager.Enable(installed.ID); err != nil {
		t.Fatalf("re-enable plugin: %v", err)
	}
	if err := supervisor.Start(context.Background(), mustPharmaInfo(t, manager)); err != nil {
		t.Fatalf("restart packaged backend: %v", err)
	}
	restarted := pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/customers?keyword=CUS-E2E-001&limit=20", actorAToken, "", nil, http.StatusOK)
	if pharmaLifecycleInt(t, restarted, "total") != 1 {
		t.Fatalf("restart did not retain customer data: %v", restarted)
	}
	eligibility = pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/customers/"+customerID+"/sales-eligibility", actorAToken, "", nil, http.StatusOK)
	if eligible, ok := eligibility["eligible"].(bool); !ok || !eligible {
		t.Fatalf("restart did not retain qualification gate: %v", eligibility)
	}
	restartScan := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/qualifications/expiry-scan", actorAToken, "e2e-expiry-after-restart", map[string]any{}, http.StatusOK)
	if pharmaLifecycleInt(t, restartScan, "scheduled") != 0 || pharmaLifecycleInt(t, restartScan, "skipped") != 1 {
		t.Fatalf("restart lost expiry-alert idempotency: %v", restartScan)
	}
	restartedOA := pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/oa-requests?limit=20", actorAToken, "", nil, http.StatusOK)
	if pharmaLifecycleInt(t, restartedOA, "total") != 5 {
		t.Fatalf("restart did not retain all OA requests: %v", restartedOA)
	}
	restartedLeave := pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/oa-requests/"+leaveID, actorAToken, "", nil, http.StatusOK)
	restartedLeaveItem := pharmaLifecycleMap(t, restartedLeave, "item")
	if pharmaLifecycleString(t, restartedLeaveItem, "status") != "approved" ||
		len(pharmaLifecycleArray(t, restartedLeaveItem, "attachments")) != 1 ||
		len(pharmaLifecycleArray(t, restartedLeaveItem, "comments")) != 1 {
		t.Fatalf("restart lost OA collaboration state: %v", restartedLeave)
	}
	restartedWorkflow := pharmaLifecycleMap(t, restartedLeave, "workflow")
	if pharmaLifecycleString(t, restartedWorkflow, "status") != string(pluginsdk.WorkflowInstanceApproved) ||
		len(pharmaLifecycleArray(t, restartedWorkflow, "timeline")) != 3 {
		t.Fatalf("restart lost OA workflow timeline: %v", restartedWorkflow)
	}
	if files.count() != 6 || jobs.count() != 2 || workflows.instanceCount() != 7 {
		t.Fatalf("restart changed host-owned OA resources: files=%d jobs=%d workflows=%d", files.count(), jobs.count(), workflows.instanceCount())
	}

	restartedPurchases := pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/purchase-requests", actorAToken, "", nil, http.StatusOK)
	restartedOrders := pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/purchase-orders", actorAToken, "", nil, http.StatusOK)
	restartedInbounds := pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/purchase-inbounds?purchaseOrderId="+purchaseOrderID, actorAToken, "", nil, http.StatusOK)
	if pharmaLifecycleInt(t, restartedPurchases, "total") != 2 || pharmaLifecycleInt(t, restartedOrders, "total") != 1 ||
		pharmaLifecycleInt(t, restartedInbounds, "total") != 1 {
		t.Fatalf("restart lost purchase facts: requests=%v orders=%v inbounds=%v", restartedPurchases, restartedOrders, restartedInbounds)
	}
	restartedOrderResponse := pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/purchase-orders/"+purchaseOrderID, actorAToken, "", nil, http.StatusOK)
	restartedOrder := pharmaLifecycleMap(t, restartedOrderResponse, "item")
	if pharmaLifecycleString(t, restartedOrder, "status") != "partial" || pharmaLifecycleInt(t, restartedOrder, "version") != 2 {
		t.Fatalf("restart lost partial order state: %v", restartedOrderResponse)
	}
	finalInbound := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/purchase-inbounds", actorAToken, "e2e-inbound-final", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "purchaseOrderId": purchaseOrderID,
		"warehouseId": "WH-E2E", "areaId": "AREA-A", "locationId": "LOC-002", "orderVersion": 2,
		"lines": []map[string]any{{
			"orderLineId": purchaseOrderLineID, "quantity": "1.25", "batchNo": "LOT-E2E-002",
			"productionDate": now.AddDate(0, -1, 0).Format("2006-01-02"), "expiresAt": now.AddDate(2, 0, 0).Format("2006-01-02"),
		}},
		"attachments": []map[string]any{},
	}, http.StatusCreated)
	finalOrder := pharmaLifecycleMap(t, finalInbound, "order")
	if pharmaLifecycleString(t, finalOrder, "status") != "received" ||
		pharmaLifecycleString(t, pharmaLifecycleArrayMap(t, finalOrder, "lines", 0), "receivedQuantity") != "2.5" {
		t.Fatalf("final inbound did not close exact order quantity: %v", finalInbound)
	}
	finalInbounds := pharmaLifecycleRequest(t, http.MethodGet, baseURL+"/purchase-inbounds?purchaseOrderId="+purchaseOrderID, actorAToken, "", nil, http.StatusOK)
	if pharmaLifecycleInt(t, finalInbounds, "total") != 2 || files.count() != 6 || workflows.instanceCount() != 7 {
		t.Fatalf("final receiving changed retained resources: inbounds=%v files=%d workflows=%d", finalInbounds, files.count(), workflows.instanceCount())
	}

	if err := manager.Disable(installed.ID); err != nil {
		t.Fatalf("disable before uninstall: %v", err)
	}
	if err := supervisor.Stop(context.Background(), installed.ID); err != nil {
		t.Fatalf("stop before uninstall: %v", err)
	}
	if _, err := migrationHook.Run(context.Background(), PluginMigrationHookInput{
		PluginID: installed.ID, PluginDir: installedDir, MigrationDirectory: installed.DataManifest.MigrationDirectory,
		Action: PluginMigrationUninstall, FromVersion: installed.DataManifest.MigrationVersion, UninstallPolicy: installed.DataManifest.UninstallPolicy,
	}); err != nil {
		t.Fatalf("uninstall migration: %v", err)
	}
	dataDir, err := dataDirectories.Prepare(installed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := dataDirectories.Uninstall(installed.ID, installed.DataManifest.UninstallPolicy); err != nil {
		t.Fatalf("drop plugin data: %v", err)
	}
	if err := manager.Uninstall(installed.ID); err != nil {
		t.Fatalf("uninstall plugin: %v", err)
	}
	final, err := manager.Get(installed.ID)
	if err != nil || final.State != StateUninstalled || len(migrationStore.records) != 0 {
		t.Fatalf("final state=%+v ledger=%+v err=%v", final, migrationStore.records, err)
	}
	if _, err := os.Lstat(dataDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("drop uninstall retained plugin data: %v", err)
	}
	if err := equipmentEndpointClosed(baseURL + "/customers"); err != nil {
		t.Fatal(err)
	}
	if err := equipmentEndpointClosed(baseURL + "/oa-requests"); err != nil {
		t.Fatal(err)
	}
	if err := equipmentEndpointClosed(baseURL + "/purchase-inbounds"); err != nil {
		t.Fatal(err)
	}
}

type pharmaLifecycleScope struct {
	tenant       string
	organization string
	owner        string
}

type pharmaLifecycleScopes struct{}

func (pharmaLifecycleScopes) Resolve(ctx context.Context, _ pluginsdk.Permission) (pluginsdk.ScopePredicate, error) {
	claims, ok := security.JWTClaimsFromContext(ctx)
	if !ok {
		return pluginsdk.ScopePredicate{}, errors.New("user claims required")
	}
	scope, err := pharmaScopeForSubject(claims.Subject)
	if err != nil {
		return pluginsdk.ScopePredicate{}, err
	}
	return pluginsdk.NewScopePredicate(pluginsdk.TrustedScope{
		SubjectID: scope.owner, TenantIDs: []string{scope.tenant}, OrganizationIDs: []string{scope.organization}, OwnerIDs: []string{scope.owner},
	})
}

func pharmaScopeForSubject(subject string) (pharmaLifecycleScope, error) {
	switch strings.TrimSpace(subject) {
	case "actor-a":
		return pharmaLifecycleScope{tenant: "tenant-a", organization: "org-a", owner: "actor-a"}, nil
	case "actor-b":
		return pharmaLifecycleScope{tenant: "tenant-b", organization: "org-b", owner: "actor-b"}, nil
	default:
		return pharmaLifecycleScope{}, fmt.Errorf("unsupported lifecycle actor %q", subject)
	}
}

type pharmaLifecycleStore struct {
	mu          sync.Mutex
	records     map[string]map[string]pluginsdk.DataRecord
	idempotency map[string]pharmaLifecycleMutationRef
}

type pharmaLifecycleMutationRef struct {
	table string
	id    string
}

func newPharmaLifecycleStore() *pharmaLifecycleStore {
	return &pharmaLifecycleStore{records: make(map[string]map[string]pluginsdk.DataRecord), idempotency: make(map[string]pharmaLifecycleMutationRef)}
}

func (s *pharmaLifecycleStore) Query(ctx context.Context, query pluginsdk.DataQuery) (pluginsdk.DataPage, error) {
	if err := query.Validate(); err != nil {
		return pluginsdk.DataPage{}, err
	}
	actor, err := pharmaLifecycleScopeFromContext(ctx)
	if err != nil {
		return pluginsdk.DataPage{}, err
	}
	if !pharmaLifecycleScopeIntentWithin(query.Scope.Filter, actor) {
		return pluginsdk.DataPage{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorForbidden, "scope", "query scope exceeds actor scope", false)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	records := make([]pluginsdk.DataRecord, 0)
	for _, record := range s.records[query.Table] {
		if !pharmaLifecycleRecordInScope(record, actor) || query.Filter != nil && !pharmaLifecycleFilterMatches(record, *query.Filter) {
			continue
		}
		records = append(records, record)
	}
	sort.SliceStable(records, func(left, right int) bool {
		for _, order := range query.Sort {
			leftValue := records[left].Values[order.Field].Value
			rightValue := records[right].Values[order.Field].Value
			if leftValue == rightValue {
				continue
			}
			if order.Direction == pluginsdk.DataSortDescending {
				return leftValue > rightValue
			}
			return leftValue < rightValue
		}
		return records[left].Values["id"].Value < records[right].Values["id"].Value
	})
	start := 0
	if query.Page.Cursor != "" {
		start, err = strconv.Atoi(query.Page.Cursor)
		if err != nil || start < 0 || start > len(records) {
			return pluginsdk.DataPage{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "cursor", "cursor is invalid", false)
		}
	}
	end := min(len(records), start+query.Page.Limit)
	projected := make([]pluginsdk.DataRecord, 0, end-start)
	for _, record := range records[start:end] {
		projected = append(projected, pharmaLifecycleProjectRecord(record, query.Fields))
	}
	nextCursor := ""
	if end < len(records) {
		nextCursor = strconv.Itoa(end)
	}
	return pluginsdk.DataPage{Records: projected, NextCursor: nextCursor, HasMore: nextCursor != ""}, nil
}

func (s *pharmaLifecycleStore) Mutate(ctx context.Context, mutation pluginsdk.DataMutation) (pluginsdk.DataMutationResult, error) {
	if err := mutation.Validate(); err != nil {
		return pluginsdk.DataMutationResult{}, err
	}
	if ctx.Value(gatewayTxMarker{}) != true {
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "transaction", "mutation escaped transaction", false)
	}
	actor, err := pharmaLifecycleScopeFromContext(ctx)
	if err != nil {
		return pluginsdk.DataMutationResult{}, err
	}
	if !pharmaLifecycleExactScope(mutation.Scope.Filter, actor) {
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorForbidden, "scope", "write scope is not exact", false)
	}
	idValue, ok := mutation.Key["id"]
	if !ok || idValue.Type != pluginsdk.DataValueString || idValue.Value == "" {
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorInvalidRequest, "key.id", "string id is required", false)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.idempotency[mutation.IdempotencyKey]; ok {
		record := pharmaLifecycleProjectRecord(s.records[existing.table][existing.id], mutation.Returning)
		return pluginsdk.DataMutationResult{RowsAffected: 1, Record: &record}, nil
	}
	if s.records[mutation.Table] == nil {
		s.records[mutation.Table] = make(map[string]pluginsdk.DataRecord)
	}
	id := idValue.Value
	now := time.Now().UTC().Format(time.RFC3339Nano)
	switch mutation.Operation {
	case pluginsdk.DataMutationInsert:
		if _, exists := s.records[mutation.Table][id]; exists {
			return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorConflict, "id", "record exists", false)
		}
		values := pharmaLifecycleCloneValues(mutation.Values)
		values["id"] = pharmaLifecycleStringValue(id)
		values["tenant_id"] = pharmaLifecycleStringValue(actor.tenant)
		values["organization_id"] = pharmaLifecycleStringValue(actor.organization)
		values["owner_id"] = pharmaLifecycleStringValue(actor.owner)
		values["created_at"] = pharmaLifecycleTimestampValue(now)
		values["updated_at"] = pharmaLifecycleTimestampValue(now)
		s.records[mutation.Table][id] = pluginsdk.DataRecord{Values: values, Version: 1}
	case pluginsdk.DataMutationUpdate, pluginsdk.DataMutationUpsert:
		record, exists := s.records[mutation.Table][id]
		if !exists {
			return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorNotFound, "id", "record not found", false)
		}
		if !pharmaLifecycleRecordInScope(record, actor) {
			return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorNotFound, "id", "record not found", false)
		}
		if mutation.ExpectedVersion == nil || *mutation.ExpectedVersion != record.Version {
			return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorConflict, "version", "record version is stale", false)
		}
		for field, value := range mutation.Values {
			record.Values[field] = value
		}
		record.Values["updated_at"] = pharmaLifecycleTimestampValue(now)
		record.Version++
		s.records[mutation.Table][id] = record
	case pluginsdk.DataMutationDelete:
		record, exists := s.records[mutation.Table][id]
		if !exists || !pharmaLifecycleRecordInScope(record, actor) {
			return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorNotFound, "id", "record not found", false)
		}
		if mutation.ExpectedVersion != nil && *mutation.ExpectedVersion != record.Version {
			return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorConflict, "version", "record version is stale", false)
		}
		delete(s.records[mutation.Table], id)
		s.idempotency[mutation.IdempotencyKey] = pharmaLifecycleMutationRef{table: mutation.Table, id: id}
		return pluginsdk.DataMutationResult{RowsAffected: 1}, nil
	default:
		return pluginsdk.DataMutationResult{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnsupported, "operation", "unsupported mutation", false)
	}
	s.idempotency[mutation.IdempotencyKey] = pharmaLifecycleMutationRef{table: mutation.Table, id: id}
	record := pharmaLifecycleProjectRecord(s.records[mutation.Table][id], mutation.Returning)
	return pluginsdk.DataMutationResult{RowsAffected: 1, Record: &record}, nil
}

func pharmaLifecycleScopeFromContext(ctx context.Context) (pharmaLifecycleScope, error) {
	claims, ok := security.JWTClaimsFromContext(ctx)
	if !ok {
		return pharmaLifecycleScope{}, pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorForbidden, "scope", "user claims required", false)
	}
	return pharmaScopeForSubject(claims.Subject)
}

func pharmaLifecycleExactScope(filter pluginsdk.ScopeFilter, scope pharmaLifecycleScope) bool {
	return len(filter.TenantIDs) == 1 && filter.TenantIDs[0] == scope.tenant &&
		len(filter.OrganizationIDs) == 1 && filter.OrganizationIDs[0] == scope.organization &&
		len(filter.OwnerIDs) == 1 && filter.OwnerIDs[0] == scope.owner
}

func pharmaLifecycleScopeIntentWithin(filter pluginsdk.ScopeFilter, scope pharmaLifecycleScope) bool {
	return pharmaLifecycleScopeDimensionWithin(filter.TenantIDs, scope.tenant) &&
		pharmaLifecycleScopeDimensionWithin(filter.OrganizationIDs, scope.organization) &&
		pharmaLifecycleScopeDimensionWithin(filter.OwnerIDs, scope.owner)
}

func pharmaLifecycleScopeDimensionWithin(values []string, expected string) bool {
	return len(values) == 0 || len(values) == 1 && values[0] == expected
}

func pharmaLifecycleRecordInScope(record pluginsdk.DataRecord, scope pharmaLifecycleScope) bool {
	return record.Values["tenant_id"].Value == scope.tenant && record.Values["organization_id"].Value == scope.organization && record.Values["owner_id"].Value == scope.owner
}

func pharmaLifecycleFilterMatches(record pluginsdk.DataRecord, filter pluginsdk.DataFilter) bool {
	if len(filter.All) > 0 {
		for _, child := range filter.All {
			if !pharmaLifecycleFilterMatches(record, child) {
				return false
			}
		}
		return true
	}
	if len(filter.Any) > 0 {
		for _, child := range filter.Any {
			if pharmaLifecycleFilterMatches(record, child) {
				return true
			}
		}
		return false
	}
	value, exists := record.Values[filter.Field]
	switch filter.Operator {
	case pluginsdk.DataOperatorIsNull:
		return !exists || value.Type == pluginsdk.DataValueNull
	case pluginsdk.DataOperatorNotNull:
		return exists && value.Type != pluginsdk.DataValueNull
	case pluginsdk.DataOperatorIn, pluginsdk.DataOperatorNotIn:
		matched := false
		for _, candidate := range filter.Values {
			if value == candidate {
				matched = true
				break
			}
		}
		if filter.Operator == pluginsdk.DataOperatorNotIn {
			return !matched
		}
		return matched
	case pluginsdk.DataOperatorContains:
		return filter.Value != nil && strings.Contains(strings.ToLower(value.Value), strings.ToLower(filter.Value.Value))
	case pluginsdk.DataOperatorPrefix:
		return filter.Value != nil && strings.HasPrefix(strings.ToLower(value.Value), strings.ToLower(filter.Value.Value))
	case pluginsdk.DataOperatorEqual:
		return filter.Value != nil && value == *filter.Value
	case pluginsdk.DataOperatorNotEqual:
		return filter.Value != nil && value != *filter.Value
	case pluginsdk.DataOperatorLess:
		return filter.Value != nil && value.Value < filter.Value.Value
	case pluginsdk.DataOperatorLessOrEqual:
		return filter.Value != nil && value.Value <= filter.Value.Value
	case pluginsdk.DataOperatorGreater:
		return filter.Value != nil && value.Value > filter.Value.Value
	case pluginsdk.DataOperatorGreaterOrEq:
		return filter.Value != nil && value.Value >= filter.Value.Value
	default:
		return false
	}
}

func pharmaLifecycleProjectRecord(record pluginsdk.DataRecord, fields []string) pluginsdk.DataRecord {
	if len(fields) == 0 {
		return pluginsdk.DataRecord{Values: pharmaLifecycleCloneValues(record.Values), Version: record.Version}
	}
	values := make(map[string]pluginsdk.DataValue, len(fields))
	for _, field := range fields {
		value, ok := record.Values[field]
		if !ok {
			value = pluginsdk.DataValue{Type: pluginsdk.DataValueNull}
		}
		values[field] = value
	}
	return pluginsdk.DataRecord{Values: values, Version: record.Version}
}

func pharmaLifecycleCloneValues(values map[string]pluginsdk.DataValue) map[string]pluginsdk.DataValue {
	clone := make(map[string]pluginsdk.DataValue, len(values))
	for field, value := range values {
		clone[field] = value
	}
	return clone
}

func pharmaLifecycleStringValue(value string) pluginsdk.DataValue {
	return pluginsdk.DataValue{Type: pluginsdk.DataValueString, Value: value}
}

func pharmaLifecycleTimestampValue(value string) pluginsdk.DataValue {
	return pluginsdk.DataValue{Type: pluginsdk.DataValueTimestamp, Value: value}
}

type pharmaLifecycleFiles struct {
	mu    sync.Mutex
	items map[string]pluginsdk.FileObject
}

func newPharmaLifecycleFiles() *pharmaLifecycleFiles {
	return &pharmaLifecycleFiles{items: make(map[string]pluginsdk.FileObject)}
}

func (f *pharmaLifecycleFiles) Store(_ context.Context, input pluginsdk.FileWrite) (pluginsdk.FileObject, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now().UTC()
	id := fmt.Sprintf("file-%d", len(f.items)+1)
	item := pluginsdk.FileObject{
		ID: id, Key: input.Key, Name: input.Name, Size: int64(len(input.Content)), MIME: "application/octet-stream",
		Visibility: input.Visibility, Status: "ready", Metadata: input.Metadata, CreatedAt: now, UpdatedAt: now,
	}
	f.items[id] = item
	return item, nil
}

func (f *pharmaLifecycleFiles) List(_ context.Context, query pluginsdk.FileQuery) ([]pluginsdk.FileObject, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	items := make([]pluginsdk.FileObject, 0, len(f.items))
	for _, item := range f.items {
		if query.Visibility == "" || item.Visibility == query.Visibility {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(left, right int) bool { return items[left].ID < items[right].ID })
	if query.Offset >= len(items) {
		return []pluginsdk.FileObject{}, nil
	}
	items = items[query.Offset:]
	if query.Limit > 0 && len(items) > query.Limit {
		items = items[:query.Limit]
	}
	return items, nil
}

func (f *pharmaLifecycleFiles) Get(_ context.Context, id string) (pluginsdk.FileObject, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	item, ok := f.items[id]
	if !ok {
		return pluginsdk.FileObject{}, errors.New("file not found")
	}
	return item, nil
}

func (*pharmaLifecycleFiles) Download(context.Context, string) (pluginsdk.FileDownload, error) {
	return pluginsdk.FileDownload{}, errors.New("download is outside this lifecycle acceptance")
}

func (f *pharmaLifecycleFiles) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.items, id)
	return nil
}

func (f *pharmaLifecycleFiles) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.items)
}

type pharmaLifecycleJobs struct {
	mu            sync.Mutex
	items         map[string]pluginsdk.Job
	byIdempotency map[string]string
}

func newPharmaLifecycleJobs() *pharmaLifecycleJobs {
	return &pharmaLifecycleJobs{items: make(map[string]pluginsdk.Job), byIdempotency: make(map[string]string)}
}

func (j *pharmaLifecycleJobs) Schedule(_ context.Context, input pluginsdk.JobScheduleInput) (pluginsdk.Job, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if id, exists := j.byIdempotency[input.IdempotencyKey]; exists {
		return j.items[id], nil
	}
	now := time.Now().UTC()
	item := pluginsdk.Job{
		ID: input.ID, Kind: input.Kind, IdempotencyKey: input.IdempotencyKey, Payload: input.Payload,
		Status: pluginsdk.JobStatusScheduled, RunAt: input.RunAt, MaxAttempts: input.MaxAttempts, CreatedAt: now, UpdatedAt: now,
	}
	j.items[item.ID] = item
	j.byIdempotency[item.IdempotencyKey] = item.ID
	return item, nil
}

func (*pharmaLifecycleJobs) LeaseDue(context.Context, pluginsdk.JobLeaseInput) ([]pluginsdk.Job, error) {
	return nil, nil
}

func (j *pharmaLifecycleJobs) Complete(_ context.Context, input pluginsdk.JobCompleteInput) (pluginsdk.Job, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	item, ok := j.items[input.JobID]
	if !ok {
		return pluginsdk.Job{}, errors.New("job not found")
	}
	now := time.Now().UTC()
	item.Status, item.Result, item.UpdatedAt, item.CompletedAt = pluginsdk.JobStatusSucceeded, input.Result, now, &now
	j.items[item.ID] = item
	return item, nil
}

func (j *pharmaLifecycleJobs) Fail(_ context.Context, input pluginsdk.JobFailInput) (pluginsdk.Job, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	item, ok := j.items[input.JobID]
	if !ok {
		return pluginsdk.Job{}, errors.New("job not found")
	}
	item.Status, item.LastError, item.UpdatedAt = pluginsdk.JobStatusRetryWait, input.Error, time.Now().UTC()
	j.items[item.ID] = item
	return item, nil
}

func (j *pharmaLifecycleJobs) Get(_ context.Context, id string) (pluginsdk.Job, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	item, ok := j.items[id]
	if !ok {
		return pluginsdk.Job{}, errors.New("job not found")
	}
	return item, nil
}

func (j *pharmaLifecycleJobs) List(_ context.Context, query pluginsdk.JobQuery) ([]pluginsdk.Job, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	items := make([]pluginsdk.Job, 0, len(j.items))
	for _, item := range j.items {
		if query.Kind != "" && item.Kind != query.Kind || query.Status != "" && item.Status != query.Status {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(left, right int) bool { return items[left].ID < items[right].ID })
	if query.Limit > 0 && len(items) > query.Limit {
		items = items[:query.Limit]
	}
	return items, nil
}

func (j *pharmaLifecycleJobs) count() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return len(j.items)
}

type pharmaLifecycleWorkflows struct {
	mu          sync.Mutex
	definitions map[string]pluginsdk.WorkflowDefinition
	instances   map[string]pluginsdk.WorkflowInstance
}

func newPharmaLifecycleWorkflows() *pharmaLifecycleWorkflows {
	return &pharmaLifecycleWorkflows{
		definitions: make(map[string]pluginsdk.WorkflowDefinition),
		instances:   make(map[string]pluginsdk.WorkflowInstance),
	}
}

func (w *pharmaLifecycleWorkflows) CreateDefinition(ctx context.Context, input pluginsdk.WorkflowDefinitionInput) (pluginsdk.WorkflowDefinition, error) {
	if err := pharmaLifecycleRequireTransaction(ctx); err != nil {
		return pluginsdk.WorkflowDefinition{}, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if item, exists := w.definitions[input.ID]; exists {
		return item, nil
	}
	now := time.Now().UTC()
	item := pluginsdk.WorkflowDefinition{
		ID: input.ID, Key: input.Key, Name: input.Name, Version: input.Version, Status: pluginsdk.WorkflowDefinitionDraft,
		Nodes: input.Nodes, Transitions: input.Transitions, CreatedAt: now, UpdatedAt: now,
	}
	w.definitions[item.ID] = item
	return item, nil
}

func (w *pharmaLifecycleWorkflows) GetDefinition(_ context.Context, id string) (pluginsdk.WorkflowDefinition, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	item, exists := w.definitions[id]
	if !exists {
		return pluginsdk.WorkflowDefinition{}, errors.New("workflow definition not found")
	}
	return item, nil
}

func (w *pharmaLifecycleWorkflows) PublishDefinition(ctx context.Context, id string) (pluginsdk.WorkflowDefinition, error) {
	if err := pharmaLifecycleRequireTransaction(ctx); err != nil {
		return pluginsdk.WorkflowDefinition{}, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	item, exists := w.definitions[id]
	if !exists {
		return pluginsdk.WorkflowDefinition{}, errors.New("workflow definition not found")
	}
	item.Status, item.UpdatedAt = pluginsdk.WorkflowDefinitionPublished, time.Now().UTC()
	w.definitions[id] = item
	return item, nil
}

func (w *pharmaLifecycleWorkflows) Start(ctx context.Context, input pluginsdk.WorkflowStartInput) (pluginsdk.WorkflowInstance, error) {
	if err := pharmaLifecycleRequireTransaction(ctx); err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if item, exists := w.instances[input.ID]; exists {
		return item, nil
	}
	definition, exists := w.definitions[input.DefinitionID]
	if !exists || definition.Status != pluginsdk.WorkflowDefinitionPublished {
		return pluginsdk.WorkflowInstance{}, errors.New("published workflow definition not found")
	}
	var approval pluginsdk.WorkflowNode
	for _, node := range definition.Nodes {
		if node.Type == pluginsdk.WorkflowNodeApproval {
			approval = node
			break
		}
	}
	if len(approval.AssigneeIDs) != 1 {
		return pluginsdk.WorkflowInstance{}, errors.New("one workflow approver is required")
	}
	actor := pharmaLifecycleWorkflowActor(ctx)
	now := time.Now().UTC()
	item := pluginsdk.WorkflowInstance{
		ID: input.ID, DefinitionID: input.DefinitionID, DefinitionKey: definition.Key, BusinessType: input.BusinessType,
		BusinessID: input.BusinessID, Title: input.Title, Status: pluginsdk.WorkflowInstanceRunning, Starter: actor,
		CurrentNode: approval.ID, CreatedAt: now, UpdatedAt: now,
		Tasks: []pluginsdk.WorkflowTask{{
			ID: input.ID + "-task", InstanceID: input.ID, NodeID: approval.ID,
			Assignee: pluginsdk.WorkflowActor{ID: approval.AssigneeIDs[0], Name: approval.AssigneeIDs[0]}, Status: pluginsdk.WorkflowTaskPending, CreatedAt: now,
		}},
		Timeline: []pluginsdk.WorkflowAction{{
			ID: input.ID + "-start", Type: pluginsdk.WorkflowActionStart, InstanceID: input.ID, NodeID: "start", Actor: actor, CreatedAt: now,
		}},
	}
	w.instances[item.ID] = item
	return item, nil
}

func (w *pharmaLifecycleWorkflows) GetInstance(_ context.Context, id string) (pluginsdk.WorkflowInstance, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	item, exists := w.instances[id]
	if !exists {
		return pluginsdk.WorkflowInstance{}, errors.New("workflow instance not found")
	}
	return item, nil
}

func (w *pharmaLifecycleWorkflows) Approve(ctx context.Context, input pluginsdk.WorkflowTaskActionInput) (pluginsdk.WorkflowInstance, error) {
	return w.taskAction(ctx, input, true)
}

func (w *pharmaLifecycleWorkflows) Reject(ctx context.Context, input pluginsdk.WorkflowTaskActionInput) (pluginsdk.WorkflowInstance, error) {
	return w.taskAction(ctx, input, false)
}

func (w *pharmaLifecycleWorkflows) taskAction(ctx context.Context, input pluginsdk.WorkflowTaskActionInput, approve bool) (pluginsdk.WorkflowInstance, error) {
	if err := pharmaLifecycleRequireTransaction(ctx); err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	item, exists := w.instances[input.InstanceID]
	if !exists {
		return pluginsdk.WorkflowInstance{}, errors.New("workflow instance not found")
	}
	now := time.Now().UTC()
	found := false
	for index := range item.Tasks {
		if item.Tasks[index].ID == input.TaskID && item.Tasks[index].Status == pluginsdk.WorkflowTaskPending {
			found = true
			item.Tasks[index].CompletedAt = &now
			if approve {
				item.Tasks[index].Status = pluginsdk.WorkflowTaskApproved
			} else {
				item.Tasks[index].Status = pluginsdk.WorkflowTaskRejected
			}
			break
		}
	}
	if !found {
		return pluginsdk.WorkflowInstance{}, errors.New("pending workflow task not found")
	}
	action := pluginsdk.WorkflowActionReject
	item.Status = pluginsdk.WorkflowInstanceRejected
	if approve {
		action, item.Status = pluginsdk.WorkflowActionApprove, pluginsdk.WorkflowInstanceApproved
	}
	item.Timeline = append(item.Timeline, pluginsdk.WorkflowAction{
		ID: fmt.Sprintf("%s-action-%d", item.ID, len(item.Timeline)+1), Type: action, InstanceID: item.ID,
		TaskID: input.TaskID, NodeID: item.CurrentNode, Actor: pharmaLifecycleWorkflowActor(ctx), Comment: input.Comment, CreatedAt: now,
	})
	item.UpdatedAt = now
	w.instances[item.ID] = item
	return item, nil
}

func (w *pharmaLifecycleWorkflows) Withdraw(ctx context.Context, input pluginsdk.WorkflowInstanceActionInput) (pluginsdk.WorkflowInstance, error) {
	return w.instanceAction(ctx, input, pluginsdk.WorkflowInstanceWithdrawn, pluginsdk.WorkflowActionWithdraw)
}

func (w *pharmaLifecycleWorkflows) Cancel(ctx context.Context, input pluginsdk.WorkflowInstanceActionInput) (pluginsdk.WorkflowInstance, error) {
	return w.instanceAction(ctx, input, pluginsdk.WorkflowInstanceCanceled, pluginsdk.WorkflowActionCancel)
}

func (w *pharmaLifecycleWorkflows) instanceAction(ctx context.Context, input pluginsdk.WorkflowInstanceActionInput, status pluginsdk.WorkflowInstanceStatus, action pluginsdk.WorkflowActionType) (pluginsdk.WorkflowInstance, error) {
	if err := pharmaLifecycleRequireTransaction(ctx); err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	item, exists := w.instances[input.InstanceID]
	if !exists {
		return pluginsdk.WorkflowInstance{}, errors.New("workflow instance not found")
	}
	now := time.Now().UTC()
	for index := range item.Tasks {
		if item.Tasks[index].Status == pluginsdk.WorkflowTaskPending {
			item.Tasks[index].Status = pluginsdk.WorkflowTaskCanceled
			item.Tasks[index].CompletedAt = &now
		}
	}
	item.Status, item.UpdatedAt = status, now
	item.Timeline = append(item.Timeline, pluginsdk.WorkflowAction{
		ID: fmt.Sprintf("%s-action-%d", item.ID, len(item.Timeline)+1), Type: action, InstanceID: item.ID,
		Actor: pharmaLifecycleWorkflowActor(ctx), Comment: input.Comment, CreatedAt: now,
	})
	w.instances[item.ID] = item
	return item, nil
}

func (w *pharmaLifecycleWorkflows) Transfer(ctx context.Context, input pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	if err := pharmaLifecycleRequireTransaction(ctx); err != nil {
		return pluginsdk.WorkflowInstance{}, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	item, exists := w.instances[input.InstanceID]
	if !exists {
		return pluginsdk.WorkflowInstance{}, errors.New("workflow instance not found")
	}
	now := time.Now().UTC()
	found := false
	for index := range item.Tasks {
		if item.Tasks[index].ID == input.TaskID && item.Tasks[index].Status == pluginsdk.WorkflowTaskPending {
			found = true
			item.Tasks[index].Status = pluginsdk.WorkflowTaskTransferred
			item.Tasks[index].CompletedAt = &now
			break
		}
	}
	if !found {
		return pluginsdk.WorkflowInstance{}, errors.New("pending workflow task not found")
	}
	item.Tasks = append(item.Tasks, pluginsdk.WorkflowTask{
		ID: input.TaskID + "-delegated", InstanceID: item.ID, NodeID: item.CurrentNode,
		Assignee: input.Target, Status: pluginsdk.WorkflowTaskPending, CreatedAt: now,
	})
	item.Timeline = append(item.Timeline, pluginsdk.WorkflowAction{
		ID: fmt.Sprintf("%s-action-%d", item.ID, len(item.Timeline)+1), Type: pluginsdk.WorkflowActionTransfer,
		InstanceID: item.ID, TaskID: input.TaskID, NodeID: item.CurrentNode, Actor: pharmaLifecycleWorkflowActor(ctx),
		Target: input.Target, Comment: input.Comment, CreatedAt: now,
	})
	item.UpdatedAt = now
	w.instances[item.ID] = item
	return item, nil
}

func (*pharmaLifecycleWorkflows) Copy(context.Context, pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{}, errors.New("workflow copy is outside this lifecycle acceptance")
}

func (w *pharmaLifecycleWorkflows) instanceCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.instances)
}

func pharmaLifecycleRequireTransaction(ctx context.Context) error {
	if ctx.Value(gatewayTxMarker{}) != true {
		return errors.New("workflow mutation escaped transaction")
	}
	return nil
}

func pharmaLifecycleWorkflowActor(ctx context.Context) pluginsdk.WorkflowActor {
	claims, ok := security.JWTClaimsFromContext(ctx)
	if !ok {
		return pluginsdk.WorkflowActor{}
	}
	return pluginsdk.WorkflowActor{ID: claims.Subject, Name: claims.Subject}
}

type pharmaLifecycleAudit struct {
	mu      sync.Mutex
	entries []pluginsdk.AuditEntry
}

func (a *pharmaLifecycleAudit) Record(ctx context.Context, entry pluginsdk.AuditEntry) (pluginsdk.AuditReceipt, error) {
	if ctx.Value(gatewayTxMarker{}) != true {
		return pluginsdk.AuditReceipt{}, errors.New("audit escaped transaction")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, entry)
	return pluginsdk.AuditReceipt{ID: fmt.Sprintf("audit-%d", len(a.entries)), OccurredAt: time.Now().UTC()}, nil
}

func (a *pharmaLifecycleAudit) hasAction(action string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, entry := range a.entries {
		if entry.Action == action {
			return true
		}
	}
	return false
}

func (a *pharmaLifecycleAudit) actions() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	result := make([]string, 0, len(a.entries))
	for _, entry := range a.entries {
		result = append(result, entry.Action)
	}
	return result
}

func pharmaLifecycleRequest(t *testing.T, method, target, token, requestKey string, payload map[string]any, wantStatus int) map[string]any {
	t.Helper()
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.NewReader(raw)
	}
	request, err := http.NewRequest(method, target, body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	if requestKey != "" {
		request.Header.Set("Idempotency-Key", requestKey)
	}
	response, err := (&http.Client{Timeout: 3 * time.Second}).Do(request)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, target, err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != wantStatus {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, target, response.StatusCode, wantStatus, raw)
	}
	if len(raw) == 0 {
		return nil
	}
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("decode %s %s: %v body=%s", method, target, err, raw)
	}
	return envelope.Data
}

func pharmaLifecycleMap(t *testing.T, values map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := values[key].(map[string]any)
	if !ok {
		t.Fatalf("%q is not an object: %v", key, values)
	}
	return value
}

func pharmaLifecycleString(t *testing.T, values map[string]any, key string) string {
	t.Helper()
	value, ok := values[key].(string)
	if !ok || value == "" {
		t.Fatalf("%q is not a non-empty string: %v", key, values)
	}
	return value
}

func pharmaLifecycleInt(t *testing.T, values map[string]any, key string) int64 {
	t.Helper()
	value, ok := values[key].(float64)
	if !ok {
		t.Fatalf("%q is not a number: %v", key, values)
	}
	return int64(value)
}

func pharmaLifecycleArray(t *testing.T, values map[string]any, key string) []any {
	t.Helper()
	value, ok := values[key].([]any)
	if !ok {
		t.Fatalf("%q is not an array: %v", key, values)
	}
	return value
}

func pharmaLifecycleArrayMap(t *testing.T, values map[string]any, key string, index int) map[string]any {
	t.Helper()
	items := pharmaLifecycleArray(t, values, key)
	if index < 0 || index >= len(items) {
		t.Fatalf("%q index %d is outside %d items: %v", key, index, len(items), values)
	}
	item, ok := items[index].(map[string]any)
	if !ok {
		t.Fatalf("%q index %d is not an object: %v", key, index, values)
	}
	return item
}

func pharmaLifecycleCreatePurchaseMasterData(t *testing.T, baseURL, token string) (string, string, string) {
	t.Helper()
	supplier := pharmaLifecycleMap(t, pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/suppliers", token, "e2e-purchase-supplier-create", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "SUP-E2E-001", "name": "Packaged Qualified Supplier",
		"unifiedSocialCreditCode": "91330000MAE2ES01", "region": "Zhejiang", "rating": 5,
		"contacts":        []map[string]any{{"name": "Quality owner", "phone": "13800000002", "email": "quality@supplier.example", "primary": true}},
		"addresses":       []map[string]any{{"label": "Headquarters", "province": "Zhejiang", "city": "Hangzhou", "district": "Gongshu", "detail": "88 Compliance Road", "default": true}},
		"settlementTerms": map[string]any{"currency": "CNY", "paymentDays": 30, "creditLimit": 500000},
	}, http.StatusCreated), "item")
	category := pharmaLifecycleMap(t, pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/categories", token, "e2e-purchase-category-create", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "RX-E2E", "name": "Prescription medicine", "description": "Packaged purchase category",
	}, http.StatusCreated), "item")
	unit := pharmaLifecycleMap(t, pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/units", token, "e2e-purchase-unit-create", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "BOX-E2E", "name": "Box", "description": "Packaged purchase unit", "symbol": "box", "decimalPlaces": 3,
	}, http.StatusCreated), "item")
	manufacturer := pharmaLifecycleMap(t, pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/manufacturers", token, "e2e-purchase-manufacturer-create", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "MFG-E2E-001", "name": "Packaged Qualified Manufacturer",
		"description": "Packaged medicine manufacturer", "unifiedSocialCreditCode": "91330000MAE2EM01", "licenseNumber": "MFG-E2E-2026-001",
	}, http.StatusCreated), "item")
	product := pharmaLifecycleMap(t, pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/products", token, "e2e-purchase-product-create", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "MED-E2E-001", "sku": "SKU-E2E-001",
		"name": "Packaged Acceptance Capsule", "genericName": "Acceptance Medicine", "categoryId": pharmaLifecycleString(t, category, "id"),
		"unitId": pharmaLifecycleString(t, unit, "id"), "manufacturerId": pharmaLifecycleString(t, manufacturer, "id"), "dosageForm": "capsule",
		"specification": "0.25g x 24", "approvalNumber": "NMPA-E2E-2026-001", "barcode": "690000009901",
		"storageCondition": "sealed and dry", "temperatureMin": 2, "temperatureMax": 25,
	}, http.StatusCreated), "item")
	return pharmaLifecycleString(t, supplier, "id"), pharmaLifecycleString(t, product, "id"), pharmaLifecycleString(t, manufacturer, "id")
}

func pharmaLifecycleApprovePurchaseQualification(t *testing.T, baseURL, token, subjectType, subjectID, gate, suffix string) {
	t.Helper()
	qualificationType := pharmaLifecycleMap(t, pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/qualification-types", token, "e2e-purchase-"+suffix+"-type", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "code": "PURCHASE-" + strings.ToUpper(suffix),
		"name": "Packaged purchase " + suffix + " qualification", "subjectType": subjectType, "businessGate": gate,
		"description": "Required by packaged purchase acceptance", "validityDays": 365, "alertDays": 30, "evidenceRequired": true, "businessRequired": true,
	}, http.StatusCreated), "item")
	now := time.Now().UTC()
	qualification := pharmaLifecycleMap(t, pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/qualifications", token, "e2e-purchase-"+suffix+"-qualification", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "typeId": pharmaLifecycleString(t, qualificationType, "id"),
		"subjectType": subjectType, "subjectId": subjectID, "certificateNumber": "CERT-PURCHASE-" + strings.ToUpper(suffix),
		"issuer": "Packaged Acceptance Authority", "validFrom": now.AddDate(0, 0, -1).Format("2006-01-02"), "validTo": now.AddDate(1, 0, 0).Format("2006-01-02"),
		"evidence": map[string]any{"name": suffix + ".pdf", "contentBase64": base64.StdEncoding.EncodeToString([]byte("%PDF-1.4\n%%EOF"))},
	}, http.StatusCreated), "item")
	id := pharmaLifecycleString(t, qualification, "id")
	pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/qualifications/"+id+"/submit", token, "e2e-purchase-"+suffix+"-submit", map[string]any{"version": 1}, http.StatusOK)
	pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/qualifications/"+id+"/approve", token, "e2e-purchase-"+suffix+"-approve", map[string]any{"comment": "Valid for purchase", "version": 2}, http.StatusOK)
}

func pharmaLifecycleCreateOARequest(t *testing.T, baseURL, token, requestType, title string, formData map[string]any) map[string]any {
	t.Helper()
	response := pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/oa-requests", token, "e2e-oa-"+requestType+"-create", map[string]any{
		"tenantId": "tenant-a", "organizationId": "org-a", "requestType": requestType, "title": title,
		"description": "Packaged lifecycle acceptance request", "formData": formData, "approverId": "actor-a-approver", "approverName": "Primary approver",
	}, http.StatusCreated)
	item := pharmaLifecycleMap(t, response, "item")
	if pharmaLifecycleString(t, item, "requestType") != requestType || pharmaLifecycleString(t, item, "status") != "draft" || pharmaLifecycleInt(t, item, "version") != 1 {
		t.Fatalf("unexpected created %s OA request: %v", requestType, response)
	}
	return item
}

func pharmaLifecycleSubmitOARequest(t *testing.T, baseURL, token string, item map[string]any, requestType string) map[string]any {
	t.Helper()
	return pharmaLifecycleRequest(t, http.MethodPost, baseURL+"/oa-requests/"+pharmaLifecycleString(t, item, "id")+"/submit", token, "e2e-oa-"+requestType+"-submit", map[string]any{
		"version": pharmaLifecycleInt(t, item, "version"),
	}, http.StatusOK)
}

func pharmaLifecyclePendingTaskID(t *testing.T, response map[string]any) string {
	t.Helper()
	workflow := pharmaLifecycleMap(t, response, "workflow")
	for _, raw := range pharmaLifecycleArray(t, workflow, "tasks") {
		task, ok := raw.(map[string]any)
		if ok && task["status"] == string(pluginsdk.WorkflowTaskPending) {
			return pharmaLifecycleString(t, task, "id")
		}
	}
	t.Fatalf("workflow has no pending task: %v", workflow)
	return ""
}

func pharmaLifecycleAssertOAStatus(t *testing.T, response map[string]any, want string) {
	t.Helper()
	item := pharmaLifecycleMap(t, response, "item")
	workflow := pharmaLifecycleMap(t, response, "workflow")
	if pharmaLifecycleString(t, item, "status") != want || pharmaLifecycleString(t, workflow, "status") != want {
		t.Fatalf("OA request and workflow status mismatch, want=%s response=%v", want, response)
	}
}

func signPharmaLifecycleToken(t *testing.T, secret, subject, organization string) string {
	t.Helper()
	token, err := security.SignJWT(secret, security.JWTIdentity{
		Subject: subject, OrganizationID: organization, OrganizationPath: []string{organization}, Role: "operator", Roles: []string{"operator"},
	}, 10*time.Minute, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func mustPharmaInfo(t *testing.T, manager *RuntimeManager) Info {
	t.Helper()
	info, err := manager.Get("pharma_oa")
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func rewritePharmaAddress(t *testing.T, manifestPath, address string) {
	t.Helper()
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.ReplaceAll(string(raw), "127.0.0.1:18093", address)
	if updated == string(raw) {
		t.Fatal("pharma-OA manifest service address was not rewritten")
	}
	if err := os.WriteFile(manifestPath, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
}

func buildPharmaPackageSurface(t *testing.T, repoRoot, source string) {
	t.Helper()
	binary := filepath.Join(source, filepath.FromSlash(managedBackendRelativePath("pharma_oa")))
	if err := os.MkdirAll(filepath.Dir(binary), 0o755); err != nil {
		t.Fatal(err)
	}
	goBuild := exec.Command("go", "build", "-o", binary, "./plugins/pharma_oa/backend")
	goBuild.Dir = repoRoot
	if output, err := goBuild.CombinedOutput(); err != nil {
		t.Fatalf("build pharma-OA backend: %v\n%s", err, output)
	}
	npm, err := exec.LookPath("npm")
	if err != nil {
		t.Skipf("npm is required: %v", err)
	}
	frontendRoot := filepath.Join(source, "frontend")
	frontendBuild := exec.Command(npm, "run", "build")
	frontendBuild.Dir = frontendRoot
	if output, err := frontendBuild.CombinedOutput(); err != nil {
		t.Fatalf("build pharma-OA frontend: %v\n%s", err, output)
	}
}
