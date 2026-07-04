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
