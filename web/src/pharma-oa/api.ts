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
