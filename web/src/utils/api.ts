import { clearToken, getToken } from "./auth";
import { API_BASE_PREFIX } from "./api-base-prefix";
import { stripWebBasePath, withWebBasePath } from "./web-base-path";

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
function withAPIPrefix(url: string): string {
	const normalized = url.trim();
	if (normalized === "") {
		return API_BASE_PREFIX;
	}
	if (normalized.startsWith("http://") || normalized.startsWith("https://")) {
		return normalized;
	}
	if (normalized.startsWith(`${API_BASE_PREFIX}/`)) {
		return normalized;
	}
	if (normalized === API_BASE_PREFIX) {
		return normalized;
	}
	if (normalized.startsWith("/")) {
		return `${API_BASE_PREFIX}${normalized}`;
	}
	return `${API_BASE_PREFIX}/${normalized}`;
}

function handleUnauthorized(url: string): void {
	if (typeof window === "undefined") {
		return;
	}
	const normalizedURL = withAPIPrefix(url).toLowerCase();
	if (normalizedURL.startsWith(`${API_BASE_PREFIX.toLowerCase()}/v1/auth/login`)) {
		return;
	}

	clearToken();
	window.localStorage.removeItem(USER_SESSION_KEY);

	const loginPath = withWebBasePath("/login");
	if (window.location.pathname === loginPath) {
		return;
	}
	const appPath = stripWebBasePath(window.location.pathname);
	const path = `${appPath}${window.location.search}${window.location.hash}`;
	const redirect = encodeURIComponent(path || "/");
	window.location.assign(`${loginPath}?redirect=${redirect}`);
}

async function request<T>(url: string, init?: RequestInit): Promise<T> {
	const requestURL = withAPIPrefix(url);
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
		resp = await fetch(requestURL, {
			headers,
			...init
		});
	} catch (error) {
		if (error instanceof DOMException && error.name === "AbortError") {
			throw error;
		}
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
			handleUnauthorized(requestURL);
		}
		throw new ApiError(message, resp.status, code);
	}

	return (await resp.json()) as T;
}

export function apiGet<T>(url: string, init?: Omit<RequestInit, "method">): Promise<T> {
	return request<T>(url, { ...init, method: "GET" });
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

