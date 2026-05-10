import { clearToken, getToken } from "./auth";

export type ApiResponse<T> = {
	code: string;
	message: string;
	data: T;
};

export class ApiError extends Error {
	readonly status: number;
	readonly code: string;

	constructor(message: string, status: number, code = "") {
		super(message);
		this.name = "ApiError";
		this.status = status;
		this.code = code;
	}
}

const USER_SESSION_KEY = "skoll.auth.userSession";

function handleUnauthorized(url: string): void {
	if (typeof window === "undefined") {
		return;
	}
	const normalizedURL = url.trim().toLowerCase();
	if (normalizedURL.startsWith("/v1/auth/login")) {
		return;
	}

	clearToken();
	window.localStorage.removeItem(USER_SESSION_KEY);

	const path = `${window.location.pathname}${window.location.search}${window.location.hash}`;
	if (window.location.pathname === "/login") {
		return;
	}
	const redirect = encodeURIComponent(path || "/");
	window.location.assign(`/login?redirect=${redirect}`);
}

async function request<T>(url: string, init?: RequestInit): Promise<T> {
	const token = getToken().trim();
	const headers = new Headers(init?.headers ?? {});
	if (!headers.has("Content-Type")) {
		headers.set("Content-Type", "application/json");
	}
	if (token !== "") {
		const authValue = token.toLowerCase().startsWith("bearer ") ? token : `Bearer ${token}`;
		headers.set("Authorization", authValue);
	}

	let resp: Response;
	try {
		resp = await fetch(url, {
			headers,
			...init
		});
	} catch {
		throw new ApiError("network_error", 0, "network_error");
	}

	if (!resp.ok) {
		let payload: Partial<ApiResponse<unknown>> | null = null;
		try {
			payload = (await resp.json()) as Partial<ApiResponse<unknown>>;
		} catch {
			payload = null;
		}
		const message = typeof payload?.message === "string" && payload.message.trim() !== ""
			? payload.message
			: `request failed: ${resp.status}`;
		const code = typeof payload?.code === "string" ? payload.code : "";
		if (resp.status === 401) {
			handleUnauthorized(url);
		}
		throw new ApiError(message, resp.status, code);
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

export function apiPut<T>(url: string, body?: unknown): Promise<T> {
	return request<T>(url, {
		method: "PUT",
		body: body === undefined ? undefined : JSON.stringify(body)
	});
}

export function apiPatch<T>(url: string, body?: unknown): Promise<T> {
	return request<T>(url, {
		method: "PATCH",
		body: body === undefined ? undefined : JSON.stringify(body)
	});
}

export function apiDelete<T>(url: string): Promise<T> {
	return request<T>(url, { method: "DELETE" });
}

