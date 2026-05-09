export type ApiResponse<T> = {
	code: string;
	message: string;
	data: T;
};

export class ApiError extends Error {
	readonly status: number;

	constructor(message: string, status: number) {
		super(message);
		this.name = "ApiError";
		this.status = status;
	}
}

async function request<T>(url: string, init?: RequestInit): Promise<T> {
	const resp = await fetch(url, {
		headers: {
			"Content-Type": "application/json"
		},
		...init
	});

	if (!resp.ok) {
		throw new ApiError(`request failed: ${resp.status}`, resp.status);
	}

	return (await resp.json()) as T;
}

export function apiGet<T>(url: string): Promise<T> {
	return request<T>(url, { method: "GET" });
}

export function apiPost<T>(url: string, body?: unknown): Promise<T> {
	return request<T>(url, {
		method: "POST",
		body: body === undefined ? undefined : JSON.stringify(body)
	});
}

export function apiDelete<T>(url: string): Promise<T> {
	return request<T>(url, { method: "DELETE" });
}

