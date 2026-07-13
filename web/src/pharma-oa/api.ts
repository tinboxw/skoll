import { apiGet, apiPost, apiPut, type ApiResponse } from "../utils/api";

export type EmployeeStatus = "active" | "on_leave" | "left";

export type EmployeeCertificate = {
	id: string;
	name: string;
	number: string;
	expiresAt: string;
};

export type PharmaEmployee = {
	id: string;
	code: string;
	name: string;
	departmentId: string;
	positionId: string;
	phone: string;
	email: string;
	status: EmployeeStatus;
	leaveReason?: string;
	certificates: EmployeeCertificate[];
};

export type EmployeeRequest = {
	code: string;
	name: string;
	departmentId: string;
	positionId: string;
	phone?: string;
	email?: string;
	certificates: EmployeeCertificate[];
	actorId?: string;
};

export type QualificationReminder = {
	employeeId: string;
	employeeCode: string;
	employeeName: string;
	certificate: string;
	number: string;
	expiresAt: string;
};

type EmployeeListPayload = {
	items: PharmaEmployee[];
	offset: number;
	limit: number;
};

type EmployeeItemPayload = {
	item: PharmaEmployee;
};

type ReminderPayload = {
	items: QualificationReminder[];
};

export async function listEmployees(query: { keyword?: string; status?: string } = {}): Promise<PharmaEmployee[]> {
	const params = new URLSearchParams();
	if (query.keyword?.trim()) {
		params.set("keyword", query.keyword.trim());
	}
	if (query.status?.trim()) {
		params.set("status", query.status.trim());
	}
	const suffix = params.toString() ? `?${params.toString()}` : "";
	const payload = await apiGet<ApiResponse<EmployeeListPayload>>(`/v1/pharma-oa/employees${suffix}`);
	return payload.data.items;
}

export async function createEmployee(body: EmployeeRequest): Promise<PharmaEmployee> {
	const payload = await apiPost<ApiResponse<EmployeeItemPayload>>("/v1/pharma-oa/employees", body);
	return payload.data.item;
}

export async function updateEmployee(id: string, body: EmployeeRequest): Promise<PharmaEmployee> {
	const payload = await apiPut<ApiResponse<EmployeeItemPayload>>(`/v1/pharma-oa/employees/${encodeURIComponent(id)}`, body);
	return payload.data.item;
}

export async function markEmployeeLeft(id: string, reason: string, actorId?: string): Promise<PharmaEmployee> {
	const payload = await apiPost<ApiResponse<EmployeeItemPayload>>(`/v1/pharma-oa/employees/${encodeURIComponent(id)}/leave`, { reason, actorId });
	return payload.data.item;
}

export async function listEmployeeQualificationReminders(days = 30): Promise<QualificationReminder[]> {
	const payload = await apiGet<ApiResponse<ReminderPayload>>(`/v1/pharma-oa/employees/qualification-reminders?days=${encodeURIComponent(String(days))}`);
	return payload.data.items;
}

export type CustomerStatus = "active" | "disabled";

export type CustomerContact = {
	id: string;
	name: string;
	phone: string;
	email: string;
	title: string;
};

export type CustomerAttachment = {
	fileId: string;
	fileName: string;
	size: number;
};

export type CustomerQualification = {
	id: string;
	name: string;
	number: string;
	expiresAt: string;
	attachments: CustomerAttachment[];
};

export type PharmaCustomer = {
	id: string;
	code: string;
	name: string;
	region: string;
	organizationId: string;
	ownerId: string;
	rating: number;
	status: CustomerStatus;
	disableReason?: string;
	contacts: CustomerContact[];
	qualifications: CustomerQualification[];
};

export type CustomerScope = {
	ownerId?: string;
	organizationId?: string;
	includeAll?: boolean;
};

export type CustomerRequest = {
	code: string;
	name: string;
	region: string;
	organizationId: string;
	ownerId: string;
	rating?: number;
	contacts: CustomerContact[];
	qualifications: CustomerQualification[];
	actorId?: string;
	scope?: CustomerScope;
};

export type CustomerQualificationReminder = {
	customerId: string;
	customerCode: string;
	customerName: string;
	qualification: string;
	number: string;
	expiresAt: string;
};

export type CustomerSalesEligibility = {
	customerId: string;
	allowed: boolean;
	reason?: string;
};

type CustomerListPayload = {
	items: PharmaCustomer[];
	offset: number;
	limit: number;
};

type CustomerItemPayload = {
	item: PharmaCustomer;
};

type CustomerReminderPayload = {
	items: CustomerQualificationReminder[];
};

type CustomerEligibilityPayload = {
	item: CustomerSalesEligibility;
};

function appendCustomerScope(params: URLSearchParams, scope: CustomerScope = {}): void {
	if (scope.ownerId?.trim()) {
		params.set("ownerId", scope.ownerId.trim());
	}
	if (scope.organizationId?.trim()) {
		params.set("organizationId", scope.organizationId.trim());
	}
	if (scope.includeAll === true) {
		params.set("includeAll", "true");
	}
}

export async function listCustomers(query: { keyword?: string; status?: string; region?: string; scope?: CustomerScope } = {}): Promise<PharmaCustomer[]> {
	const params = new URLSearchParams();
	if (query.keyword?.trim()) {
		params.set("keyword", query.keyword.trim());
	}
	if (query.status?.trim()) {
		params.set("status", query.status.trim());
	}
	if (query.region?.trim()) {
		params.set("region", query.region.trim());
	}
	appendCustomerScope(params, query.scope);
	const suffix = params.toString() ? `?${params.toString()}` : "";
	const payload = await apiGet<ApiResponse<CustomerListPayload>>(`/v1/pharma-oa/customers${suffix}`);
	return payload.data.items;
}

export async function createCustomer(body: CustomerRequest): Promise<PharmaCustomer> {
	const payload = await apiPost<ApiResponse<CustomerItemPayload>>("/v1/pharma-oa/customers", body);
	return payload.data.item;
}

export async function updateCustomer(id: string, body: CustomerRequest): Promise<PharmaCustomer> {
	const payload = await apiPut<ApiResponse<CustomerItemPayload>>(`/v1/pharma-oa/customers/${encodeURIComponent(id)}`, body);
	return payload.data.item;
}

export async function disableCustomer(id: string, reason: string, actorId?: string, scope?: CustomerScope): Promise<PharmaCustomer> {
	const payload = await apiPost<ApiResponse<CustomerItemPayload>>(`/v1/pharma-oa/customers/${encodeURIComponent(id)}/disable`, { reason, actorId, scope });
	return payload.data.item;
}

export async function listCustomerQualificationReminders(days = 30, scope: CustomerScope = {}): Promise<CustomerQualificationReminder[]> {
	const params = new URLSearchParams({ days: String(days) });
	appendCustomerScope(params, scope);
	const payload = await apiGet<ApiResponse<CustomerReminderPayload>>(`/v1/pharma-oa/customers/qualification-reminders?${params.toString()}`);
	return payload.data.items;
}

export async function validateCustomerSalesEligibility(id: string, scope: CustomerScope = {}): Promise<CustomerSalesEligibility> {
	const params = new URLSearchParams();
	appendCustomerScope(params, scope);
	const suffix = params.toString() ? `?${params.toString()}` : "";
	const payload = await apiGet<ApiResponse<CustomerEligibilityPayload>>(`/v1/pharma-oa/customers/${encodeURIComponent(id)}/sales-eligibility${suffix}`);
	return payload.data.item;
}

export type PurchaseLine = { productId: string; quantity: number; unitPrice: number; amount: number };
export type PurchaseOrder = { id: string; number: string; purchaseRequestId: string; supplierId: string; lines: PurchaseLine[]; totalAmount: number; status: "open"; approvedBy: string; approvedAt: string };
export type InboundAttachment = { fileId: string; fileName: string; size: number };
export type PurchaseInboundLine = { productId: string; quantity: number; batchNo: string; productionDate: string; expiresAt: string; batchId?: string; ledgerId?: string };
export type PurchaseInbound = { id: string; number: string; purchaseOrderId: string; warehouseId: string; areaId: string; locationId: string; lines: PurchaseInboundLine[]; attachments: InboundAttachment[]; status: "completed"; receivedBy: string; receivedAt: string };
type PurchaseOrderListPayload = { items: PurchaseOrder[] };
type PurchaseInboundListPayload = { items: PurchaseInbound[] };
type PurchaseInboundItemPayload = { item: PurchaseInbound };

export async function listPurchaseOrders(): Promise<PurchaseOrder[]> {
	const payload = await apiGet<ApiResponse<PurchaseOrderListPayload>>("/v1/pharma-oa/purchase-orders");
	return payload.data.items;
}

export async function listPurchaseInbounds(): Promise<PurchaseInbound[]> {
	const payload = await apiGet<ApiResponse<PurchaseInboundListPayload>>("/v1/pharma-oa/purchase-inbounds");
	return payload.data.items;
}

export async function createPurchaseInbound(body: { number: string; purchaseOrderId: string; warehouseId: string; areaId: string; locationId: string; lines: PurchaseInboundLine[]; attachments: InboundAttachment[]; actorId: string }): Promise<PurchaseInbound> {
	const payload = await apiPost<ApiResponse<PurchaseInboundItemPayload>>("/v1/pharma-oa/purchase-inbounds", body);
	return payload.data.item;
}

export type SalesLine = { productId: string; quantity: number; unitPrice: number; amount?: number };
export type SalesOrder = { id: string; number: string; customerId: string; lines: SalesLine[]; totalAmount: number; status: "open"; createdBy: string; createdAt: string };
export type SalesOutboundLine = { productId: string; quantity: number; batchId: string; ledgerId?: string };
export type SalesOutbound = { id: string; number: string; salesOrderId: string; customerId: string; warehouseId: string; areaId: string; locationId: string; lines: SalesOutboundLine[]; status: "completed"; shippedBy: string; shippedAt: string };
type SalesOrderListPayload = { items: SalesOrder[] };
type SalesOrderItemPayload = { item: SalesOrder };
type SalesOutboundListPayload = { items: SalesOutbound[] };
type SalesOutboundItemPayload = { item: SalesOutbound };

export async function listSalesOrders(): Promise<SalesOrder[]> {
	const payload = await apiGet<ApiResponse<SalesOrderListPayload>>("/v1/pharma-oa/sales-orders");
	return payload.data.items;
}

export async function createSalesOrder(body: { number: string; customerId: string; lines: SalesLine[]; actorId: string }): Promise<SalesOrder> {
	const payload = await apiPost<ApiResponse<SalesOrderItemPayload>>("/v1/pharma-oa/sales-orders", body);
	return payload.data.item;
}

export async function listSalesOutbounds(): Promise<SalesOutbound[]> {
	const payload = await apiGet<ApiResponse<SalesOutboundListPayload>>("/v1/pharma-oa/sales-outbounds");
	return payload.data.items;
}

export async function createSalesOutbound(body: { number: string; salesOrderId: string; warehouseId: string; areaId: string; locationId: string; lines: SalesOutboundLine[]; actorId: string }): Promise<SalesOutbound> {
	const payload = await apiPost<ApiResponse<SalesOutboundItemPayload>>("/v1/pharma-oa/sales-outbounds", body);
	return payload.data.item;
}

export type AnnouncementKind = "announcement" | "policy";
export type AnnouncementStatus = "draft" | "published";
export type AnnouncementDocument = { fileId: string; fileName: string };
export type AnnouncementAudience = { organizationIds: string[]; roleIds: string[] };
export type PharmaAnnouncement = {
	id: string;
	kind: AnnouncementKind;
	title: string;
	content: string;
	audience: AnnouncementAudience;
	documents: AnnouncementDocument[];
	status: AnnouncementStatus;
	createdBy: string;
	createdAt: string;
	publishedBy?: string;
	publishedAt?: string;
};
export type AnnouncementReadConfirmation = { announcementId: string; userId: string; readAt: string };
type AnnouncementListPayload = { items: PharmaAnnouncement[] };
type AnnouncementItemPayload = { item: PharmaAnnouncement };
type AnnouncementReceiptPayload = { item: AnnouncementReadConfirmation };
type AnnouncementReceiptsPayload = { items: AnnouncementReadConfirmation[] };

function announcementAudienceQuery(query: { organizationIds?: string[]; roleIds?: string[]; includeDraft?: boolean }): string {
	const params = new URLSearchParams();
	query.organizationIds?.filter(Boolean).forEach((id) => params.append("organizationId", id));
	query.roleIds?.filter(Boolean).forEach((id) => params.append("roleId", id));
	if (query.includeDraft) params.set("includeDraft", "true");
	return params.toString() ? `?${params.toString()}` : "";
}

export async function listAnnouncements(query: { organizationIds?: string[]; roleIds?: string[]; includeDraft?: boolean } = {}): Promise<PharmaAnnouncement[]> {
	const payload = await apiGet<ApiResponse<AnnouncementListPayload>>(`/v1/pharma-oa/announcements${announcementAudienceQuery(query)}`);
	return payload.data.items;
}

export async function createAnnouncement(body: { kind: AnnouncementKind; title: string; content: string; audience: AnnouncementAudience; documents: AnnouncementDocument[]; actorId: string }): Promise<PharmaAnnouncement> {
	const payload = await apiPost<ApiResponse<AnnouncementItemPayload>>("/v1/pharma-oa/announcements", body);
	return payload.data.item;
}

export async function publishAnnouncement(id: string, actorId: string): Promise<PharmaAnnouncement> {
	const payload = await apiPost<ApiResponse<AnnouncementItemPayload>>(`/v1/pharma-oa/announcements/${encodeURIComponent(id)}/publish`, { actorId });
	return payload.data.item;
}

export async function confirmAnnouncementRead(id: string, query: { organizationIds?: string[]; roleIds?: string[] } = {}): Promise<AnnouncementReadConfirmation> {
	const payload = await apiPost<ApiResponse<AnnouncementReceiptPayload>>(`/v1/pharma-oa/announcements/${encodeURIComponent(id)}/read${announcementAudienceQuery(query)}`);
	return payload.data.item;
}

export async function listAnnouncementReadConfirmations(id: string): Promise<AnnouncementReadConfirmation[]> {
	const payload = await apiGet<ApiResponse<AnnouncementReceiptsPayload>>(`/v1/pharma-oa/announcements/${encodeURIComponent(id)}/read-confirmations`);
	return payload.data.items;
}

export type PharmaSupplier = { id: string; code: string; name: string; status: "active" | "disabled" };
export type ContractPartyType = "supplier" | "customer";
export type ContractStatus = "pending_approval" | "active" | "rejected" | "expired";
export type ContractAttachment = { fileId: string; fileName: string; mime: string; size: number };
export type PharmaContract = {
	id: string;
	number: string;
	title: string;
	partyType: ContractPartyType;
	partyId: string;
	partyName: string;
	ownerId: string;
	approverId: string;
	amount: number;
	currency: string;
	effectiveAt: string;
	expiresAt: string;
	attachments: ContractAttachment[];
	workflowInstanceId: string;
	status: ContractStatus;
	approvedBy?: string;
	approvedAt?: string;
	rejectedBy?: string;
	rejectedAt?: string;
	reminderNotificationId?: string;
};
export type ContractExpiryReminder = { contractId: string; contractNumber: string; partyType: ContractPartyType; partyId: string; partyName: string; expiresAt: string; recipientId: string; notificationId: string; targetPath: string };
export type ContractExpiryScanResult = { matchedCount: number; createdCount: number; reminders: ContractExpiryReminder[] };
type SupplierListPayload = { items: PharmaSupplier[] };
type ContractListPayload = { items: PharmaContract[] };
type ContractItemPayload = { item: PharmaContract };
type ContractExpiryPayload = { item: ContractExpiryScanResult };

export async function listSuppliers(query: { keyword?: string; status?: string } = {}): Promise<PharmaSupplier[]> {
	const params = new URLSearchParams();
	if (query.keyword?.trim()) params.set("keyword", query.keyword.trim());
	if (query.status?.trim()) params.set("status", query.status.trim());
	const suffix = params.toString() ? `?${params.toString()}` : "";
	const payload = await apiGet<ApiResponse<SupplierListPayload>>(`/v1/pharma-oa/suppliers${suffix}`);
	return payload.data.items;
}

export async function listContracts(query: { keyword?: string; partyType?: ContractPartyType | ""; status?: ContractStatus | "" } = {}): Promise<PharmaContract[]> {
	const params = new URLSearchParams();
	if (query.keyword?.trim()) params.set("keyword", query.keyword.trim());
	if (query.partyType) params.set("partyType", query.partyType);
	if (query.status) params.set("status", query.status);
	const suffix = params.toString() ? `?${params.toString()}` : "";
	const payload = await apiGet<ApiResponse<ContractListPayload>>(`/v1/pharma-oa/contracts${suffix}`);
	return payload.data.items;
}

export async function createContract(body: { number: string; title: string; partyType: ContractPartyType; partyId: string; ownerId: string; approverId: string; amount: number; currency: string; effectiveAt: string; expiresAt: string; attachmentIds: string[] }): Promise<PharmaContract> {
	const payload = await apiPost<ApiResponse<ContractItemPayload>>("/v1/pharma-oa/contracts", body);
	return payload.data.item;
}

export async function approveContract(id: string, actorId: string, comment: string): Promise<PharmaContract> {
	const payload = await apiPost<ApiResponse<ContractItemPayload>>(`/v1/pharma-oa/contracts/${encodeURIComponent(id)}/approve`, { actorId, comment });
	return payload.data.item;
}

export async function rejectContract(id: string, actorId: string, comment: string): Promise<PharmaContract> {
	const payload = await apiPost<ApiResponse<ContractItemPayload>>(`/v1/pharma-oa/contracts/${encodeURIComponent(id)}/reject`, { actorId, comment });
	return payload.data.item;
}

export async function scanContractExpiry(days: number, actorId: string): Promise<ContractExpiryScanResult> {
	const payload = await apiPost<ApiResponse<ContractExpiryPayload>>("/v1/pharma-oa/contracts/expiry-scan", { days, actorId });
	return payload.data.item;
}

export type QualificationSubjectType = "employee" | "supplier" | "customer";
export type QualificationStatus = "valid" | "expiring" | "expired" | "permanent";
export type QualificationRecord = {
	id: string;
	subjectType: QualificationSubjectType;
	subjectId: string;
	subjectCode: string;
	subjectName: string;
	subjectStatus: string;
	qualificationId: string;
	qualificationName: string;
	number: string;
	expiresAt: string;
	status: QualificationStatus;
	attachmentCount: number;
	recipientId: string;
	targetPath: string;
	reminderNotificationId?: string;
};
export type QualificationScanResult = { matchedCount: number; createdCount: number; reminders: QualificationRecord[] };
type QualificationListPayload = { items: QualificationRecord[] };
type QualificationScanPayload = { item: QualificationScanResult };

export async function listQualifications(query: { keyword?: string; subjectType?: QualificationSubjectType | ""; status?: QualificationStatus | ""; days?: number } = {}): Promise<QualificationRecord[]> {
	const params = new URLSearchParams();
	if (query.keyword?.trim()) params.set("keyword", query.keyword.trim());
	if (query.subjectType) params.set("subjectType", query.subjectType);
	if (query.status) params.set("status", query.status);
	if (query.days) params.set("days", String(query.days));
	const suffix = params.toString() ? `?${params.toString()}` : "";
	const payload = await apiGet<ApiResponse<QualificationListPayload>>(`/v1/pharma-oa/qualifications${suffix}`);
	return payload.data.items;
}

export async function scanQualificationExpiry(days: number, actorId: string): Promise<QualificationScanResult> {
	const payload = await apiPost<ApiResponse<QualificationScanPayload>>("/v1/pharma-oa/qualifications/expiry-scan", { days, actorId });
	return payload.data.item;
}

export type PharmaProduct = {
	id: string;
	code: string;
	name: string;
	spec: string;
	dosageForm: string;
	manufacturer: string;
	approvalNumber: string;
	status: "active" | "disabled";
};

export type QualityComplaintStatus = "pending" | "resolved" | "rejected";
export type QualityComplaintBatch = {
	id: string;
	productId: string;
	batchNo: string;
	productionDate: string;
	expiresAt: string;
};
export type QualityComplaint = {
	id: string;
	number: string;
	title: string;
	description: string;
	customerId: string;
	customerName: string;
	productId: string;
	productName: string;
	batchId: string;
	batchNo: string;
	reporterId: string;
	handlerId: string;
	attachments: ContractAttachment[];
	workflowInstanceId: string;
	status: QualityComplaintStatus;
	conclusion?: string;
	resolvedBy?: string;
	resolvedAt?: string;
	rejectedBy?: string;
	rejectedAt?: string;
	meta?: { createdAt: string; updatedAt: string };
};

type ProductListPayload = { items: PharmaProduct[] };
type QualityComplaintListPayload = { items: QualityComplaint[] };
type QualityComplaintItemPayload = { item: QualityComplaint };
type QualityComplaintBatchListPayload = { items: QualityComplaintBatch[] };

export async function listProducts(query: { keyword?: string; status?: string } = {}): Promise<PharmaProduct[]> {
	const params = new URLSearchParams();
	if (query.keyword?.trim()) params.set("keyword", query.keyword.trim());
	if (query.status?.trim()) params.set("status", query.status.trim());
	const suffix = params.toString() ? `?${params.toString()}` : "";
	const payload = await apiGet<ApiResponse<ProductListPayload>>(`/v1/pharma-oa/products${suffix}`);
	return payload.data.items;
}

export async function listQualityComplaints(query: { keyword?: string; status?: QualityComplaintStatus | "" } = {}): Promise<QualityComplaint[]> {
	const params = new URLSearchParams();
	if (query.keyword?.trim()) params.set("keyword", query.keyword.trim());
	if (query.status) params.set("status", query.status);
	const suffix = params.toString() ? `?${params.toString()}` : "";
	const payload = await apiGet<ApiResponse<QualityComplaintListPayload>>(`/v1/pharma-oa/quality-complaints${suffix}`);
	return payload.data.items;
}

export async function listQualityComplaintBatches(productId = ""): Promise<QualityComplaintBatch[]> {
	const params = new URLSearchParams();
	if (productId.trim()) params.set("productId", productId.trim());
	const suffix = params.toString() ? `?${params.toString()}` : "";
	const payload = await apiGet<ApiResponse<QualityComplaintBatchListPayload>>(`/v1/pharma-oa/quality-complaints/batches${suffix}`);
	return payload.data.items;
}

export async function createQualityComplaint(body: { number: string; title: string; description: string; customerId: string; productId: string; batchId: string; reporterId: string; handlerId: string; attachmentIds: string[] }): Promise<QualityComplaint> {
	const payload = await apiPost<ApiResponse<QualityComplaintItemPayload>>("/v1/pharma-oa/quality-complaints", body);
	return payload.data.item;
}

export async function resolveQualityComplaint(id: string, conclusion: string, actorId: string): Promise<QualityComplaint> {
	const payload = await apiPost<ApiResponse<QualityComplaintItemPayload>>(`/v1/pharma-oa/quality-complaints/${encodeURIComponent(id)}/resolve`, { conclusion, actorId });
	return payload.data.item;
}

export async function rejectQualityComplaint(id: string, conclusion: string, actorId: string): Promise<QualityComplaint> {
	const payload = await apiPost<ApiResponse<QualityComplaintItemPayload>>(`/v1/pharma-oa/quality-complaints/${encodeURIComponent(id)}/reject`, { conclusion, actorId });
	return payload.data.item;
}

export type DrugRecallStatus = "active" | "completed";
export type DrugRecallTaskStatus = "pending" | "completed";
export type DrugRecallBatch = QualityComplaintBatch;
export type DrugRecallScope = { customerId: string; customerName: string; outboundIds: string[]; quantity: number };
export type DrugRecallTask = DrugRecallScope & { id: string; status: DrugRecallTaskStatus; completionNote?: string; completedBy?: string; completedAt?: string };
export type DrugRecall = {
	id: string;
	number: string;
	title: string;
	reason: string;
	productId: string;
	productName: string;
	batchId: string;
	batchNo: string;
	sourceComplaintId?: string;
	tasks: DrugRecallTask[];
	status: DrugRecallStatus;
	initiatedBy: string;
	initiatedAt: string;
	completedBy?: string;
	completedAt?: string;
	meta?: { createdAt: string; updatedAt: string };
};
type DrugRecallListPayload = { items: DrugRecall[] };
type DrugRecallItemPayload = { item: DrugRecall };
type DrugRecallBatchListPayload = { items: DrugRecallBatch[] };
type DrugRecallScopeListPayload = { items: DrugRecallScope[] };

export async function listDrugRecalls(query: { keyword?: string; status?: DrugRecallStatus | "" } = {}): Promise<DrugRecall[]> {
	const params = new URLSearchParams();
	if (query.keyword?.trim()) params.set("keyword", query.keyword.trim());
	if (query.status) params.set("status", query.status);
	const suffix = params.toString() ? `?${params.toString()}` : "";
	const payload = await apiGet<ApiResponse<DrugRecallListPayload>>(`/v1/pharma-oa/drug-recalls${suffix}`);
	return payload.data.items;
}

export async function listDrugRecallBatches(): Promise<DrugRecallBatch[]> {
	const payload = await apiGet<ApiResponse<DrugRecallBatchListPayload>>("/v1/pharma-oa/drug-recalls/batches");
	return payload.data.items;
}

export async function previewDrugRecallScope(batchId: string): Promise<DrugRecallScope[]> {
	const params = new URLSearchParams({ batchId: batchId.trim() });
	const payload = await apiGet<ApiResponse<DrugRecallScopeListPayload>>(`/v1/pharma-oa/drug-recalls/scope?${params.toString()}`);
	return payload.data.items;
}

export async function createDrugRecall(body: { number: string; title: string; reason: string; batchId: string; sourceComplaintId?: string; actorId: string }): Promise<DrugRecall> {
	const payload = await apiPost<ApiResponse<DrugRecallItemPayload>>("/v1/pharma-oa/drug-recalls", body);
	return payload.data.item;
}

export async function completeDrugRecallTask(recallId: string, taskId: string, note: string, actorId: string): Promise<DrugRecall> {
	const payload = await apiPost<ApiResponse<DrugRecallItemPayload>>(`/v1/pharma-oa/drug-recalls/${encodeURIComponent(recallId)}/tasks/${encodeURIComponent(taskId)}/complete`, { note, actorId });
	return payload.data.item;
}

export type ColdChainContext = {
	balanceId: string;
	productId: string;
	batchId: string;
	batchNo: string;
	warehouseId: string;
	areaId: string;
	locationId: string;
	quantity: number;
	minCelsius: number;
	maxCelsius: number;
};

export type ColdChainRecord = ColdChainContext & {
	id: string;
	temperatureCelsius: number;
	humidityPercent: number;
	source: string;
	recordedBy: string;
	recordedAt: string;
	createdAt: string;
};

export type ColdChainAnomalyStatus = "active" | "resolved";
export type ColdChainAnomaly = Omit<ColdChainRecord, "quantity" | "source" | "recordedBy" | "recordedAt" | "createdAt"> & {
	status: ColdChainAnomalyStatus;
	risk: "medium" | "high";
	recordId: string;
	minHumidityPercent: number;
	maxHumidityPercent: number;
	reasons: string[];
	recipientId: string;
	notificationId: string;
	targetPath: string;
	firstSeenAt: string;
	lastSeenAt: string;
	resolvedAt?: string;
};

export type ColdChainJob = {
	id: string;
	status: "pending" | "running" | "succeeded" | "failed";
	policy: { minHumidityPercent: number; maxHumidityPercent: number; recipientId: string };
	initiatedBy: string;
	lastRunBy: string;
	retryCount: number;
	matchedCount: number;
	createdCount: number;
	resolvedCount: number;
	error?: string;
	logs: Array<{ level: "info" | "error"; message: string; createdAt: string }>;
	createdAt: string;
	startedAt?: string;
	completedAt?: string;
};

type ColdChainContextListPayload = { items: ColdChainContext[] };
type ColdChainRecordListPayload = { items: ColdChainRecord[] };
type ColdChainAnomalyListPayload = { items: ColdChainAnomaly[] };
type ColdChainJobListPayload = { items: ColdChainJob[] };
type ColdChainRecordItemPayload = { item: ColdChainRecord };
type ColdChainJobItemPayload = { item: ColdChainJob };

export async function listColdChainContexts(): Promise<ColdChainContext[]> {
	const payload = await apiGet<ApiResponse<ColdChainContextListPayload>>("/v1/pharma-oa/cold-chain-contexts");
	return payload.data.items;
}

export async function listColdChainRecords(query: { batchId?: string; warehouseId?: string; limit?: number } = {}): Promise<ColdChainRecord[]> {
	const params = new URLSearchParams();
	if (query.batchId?.trim()) params.set("batchId", query.batchId.trim());
	if (query.warehouseId?.trim()) params.set("warehouseId", query.warehouseId.trim());
	if (query.limit) params.set("limit", String(query.limit));
	const suffix = params.toString() ? `?${params.toString()}` : "";
	const payload = await apiGet<ApiResponse<ColdChainRecordListPayload>>(`/v1/pharma-oa/cold-chain-records${suffix}`);
	return payload.data.items;
}

export async function createColdChainRecord(body: { balanceId: string; temperatureCelsius: number; humidityPercent: number; source: string; recordedAt: string; actorId: string }): Promise<ColdChainRecord> {
	const payload = await apiPost<ApiResponse<ColdChainRecordItemPayload>>("/v1/pharma-oa/cold-chain-records", body);
	return payload.data.item;
}

export async function listColdChainAnomalies(activeOnly = false): Promise<ColdChainAnomaly[]> {
	const payload = await apiGet<ApiResponse<ColdChainAnomalyListPayload>>(`/v1/pharma-oa/cold-chain-anomalies?activeOnly=${String(activeOnly)}`);
	return payload.data.items;
}

export async function listColdChainJobs(): Promise<ColdChainJob[]> {
	const payload = await apiGet<ApiResponse<ColdChainJobListPayload>>("/v1/pharma-oa/cold-chain-jobs");
	return payload.data.items;
}

export async function runColdChainScan(body: { minHumidityPercent: number; maxHumidityPercent: number; recipientId: string; actorId: string }): Promise<ColdChainJob> {
	const payload = await apiPost<ApiResponse<ColdChainJobItemPayload>>("/v1/pharma-oa/cold-chain-jobs", body);
	return payload.data.item;
}

export async function retryColdChainScan(id: string, actorId: string): Promise<ColdChainJob> {
	const payload = await apiPost<ApiResponse<ColdChainJobItemPayload>>(`/v1/pharma-oa/cold-chain-jobs/${encodeURIComponent(id)}/retry`, { actorId });
	return payload.data.item;
}
