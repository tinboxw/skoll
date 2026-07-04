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
