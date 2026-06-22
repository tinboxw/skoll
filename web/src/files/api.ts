import { ApiError, apiDelete, apiGet, apiPost, type ApiResponse } from "../utils/api";
import { API_BASE_PREFIX } from "../utils/api-base-prefix";
import { getToken } from "../utils/auth";

export type FileVisibility = "private" | "public" | "plugin_asset";
export type FileStatus = "pending" | "available" | "failed" | "deleted";

export type FileOwnerRef = {
	type: string;
	id: string;
};

export type FileSourceRef = {
	module: string;
	pluginId?: string;
};

export type FileObject = {
	id: string;
	key: string;
	name: string;
	size: number;
	mime: string;
	hash: string;
	owner: FileOwnerRef;
	visibility: FileVisibility;
	storageDriver: string;
	status: FileStatus;
	source: FileSourceRef;
	metadata?: Record<string, string>;
	createdAt: string;
	updatedAt: string;
};

export type FileListQuery = {
	ownerType?: string;
	ownerId?: string;
	visibility?: FileVisibility;
	status?: FileStatus;
	sourceModule?: string;
	sourcePluginId?: string;
	offset?: number;
	limit?: number;
};

export type FileListResult = {
	items: FileObject[];
	offset: number;
	limit: number;
	total?: number;
};

export type FileUploadInput = {
	file: File | Blob;
	key?: string;
	name?: string;
	visibility: FileVisibility;
	ownerType?: string;
	ownerId?: string;
	sourceModule?: string;
	sourcePluginId?: string;
	metadata?: Record<string, string>;
	signal?: AbortSignal;
	onProgress?: (progress: FileUploadProgress) => void;
};

export type FileUploadProgress = {
	loaded: number;
	total: number;
	percent: number;
};

export type FileDownloadResult = {
	id: string;
	url: string;
	method: "GET";
	expiresAt: string;
	headers?: Record<string, string>;
};

export type MultipartInitInput = {
	key: string;
	name: string;
	size: number;
	mime: string;
	hash: string;
	ownerType?: string;
	ownerId?: string;
	visibility: FileVisibility;
	storageDriver?: string;
	sourceModule?: string;
	sourcePluginId?: string;
	metadata?: Record<string, string>;
};

export type MultipartUpload = {
	uploadId: string;
	key: string;
	expiresAt: string;
	metadata?: Record<string, string>;
};

export type MultipartInitResult = {
	item: FileObject;
	upload: MultipartUpload;
};

export type MultipartPart = {
	partNumber: number;
	size: number;
	hash: string;
	etag?: string;
};

export type MultipartPartInput = {
	uploadId: string;
	key: string;
	partNumber: number;
	size: number;
	hash: string;
	body: Blob | ArrayBuffer | string;
	signal?: AbortSignal;
	onProgress?: (progress: FileUploadProgress) => void;
};

export type MultipartCompleteInput = {
	uploadId: string;
	fileId: string;
	key?: string;
	expectedSize: number;
	expectedHash: string;
	parts: MultipartPart[];
};

export type MultipartAbortInput = {
	uploadId: string;
	fileId?: string;
	key?: string;
};

export async function listFiles(query: FileListQuery = {}): Promise<FileListResult> {
	const payload = await apiGet<ApiResponse<FileListResult>>(`/v1/files${buildFileQuery(query)}`);
	return normalizeFileListResult(payload.data);
}

export async function getFile(id: string): Promise<FileObject> {
	const payload = await apiGet<ApiResponse<{ item: FileObject }>>(`/v1/files/${encodeURIComponent(id)}`);
	return normalizeFileObject(payload.data.item);
}

export async function uploadFile(input: FileUploadInput): Promise<FileObject> {
	const form = new FormData();
	form.set("file", input.file);
	setFormValue(form, "key", input.key);
	setFormValue(form, "name", input.name);
	setFormValue(form, "visibility", input.visibility);
	setFormValue(form, "ownerType", input.ownerType);
	setFormValue(form, "ownerId", input.ownerId);
	setFormValue(form, "sourceModule", input.sourceModule);
	setFormValue(form, "sourcePluginId", input.sourcePluginId);
	if (input.metadata && Object.keys(input.metadata).length > 0) {
		form.set("metadata", JSON.stringify(input.metadata));
	}
	const payload = await sendXHR<ApiResponse<{ item: FileObject }>>({
		method: "POST",
		url: "/v1/files",
		body: form,
		signal: input.signal,
		onProgress: input.onProgress
	});
	return normalizeFileObject(payload.data.item);
}

export async function createFileDownload(id: string): Promise<FileDownloadResult> {
	const payload = await apiGet<ApiResponse<FileDownloadResult>>(`/v1/files/${encodeURIComponent(id)}/download`);
	return normalizeDownloadResult(payload.data);
}

export async function deleteFile(id: string): Promise<{ id: string; deleted: boolean }> {
	const payload = await apiDelete<ApiResponse<{ id: string; deleted: boolean }>>(`/v1/files/${encodeURIComponent(id)}`);
	return {
		id: String(payload.data.id ?? "").trim(),
		deleted: Boolean(payload.data.deleted)
	};
}

export async function initMultipartUpload(input: MultipartInitInput): Promise<MultipartInitResult> {
	const payload = await apiPost<ApiResponse<MultipartInitResult>>("/v1/files/multipart/init", input);
	return {
		item: normalizeFileObject(payload.data.item),
		upload: normalizeMultipartUpload(payload.data.upload)
	};
}

export async function uploadMultipartPart(input: MultipartPartInput): Promise<MultipartPart> {
	const params = new URLSearchParams();
	params.set("key", input.key.trim());
	params.set("size", String(Math.max(0, Math.trunc(input.size))));
	params.set("hash", input.hash.trim());
	const payload = await sendXHR<ApiResponse<{ part: MultipartPart }>>({
		method: "PUT",
		url: `/v1/files/multipart/${encodeURIComponent(input.uploadId)}/parts/${encodeURIComponent(String(input.partNumber))}?${params.toString()}`,
		body: input.body,
		contentType: "application/octet-stream",
		signal: input.signal,
		onProgress: input.onProgress
	});
	return normalizeMultipartPart(payload.data.part);
}

export async function completeMultipartUpload(input: MultipartCompleteInput): Promise<FileObject> {
	const payload = await apiPost<ApiResponse<{ item: FileObject }>>(`/v1/files/multipart/${encodeURIComponent(input.uploadId)}/complete`, {
		fileId: input.fileId,
		key: input.key,
		expectedSize: input.expectedSize,
		expectedHash: input.expectedHash,
		parts: input.parts
	});
	return normalizeFileObject(payload.data.item);
}

export async function abortMultipartUpload(input: MultipartAbortInput): Promise<{ uploadId: string; aborted: boolean }> {
	const payload = await apiPost<ApiResponse<{ uploadId: string; aborted: boolean }>>(`/v1/files/multipart/${encodeURIComponent(input.uploadId)}/abort`, {
		fileId: input.fileId,
		key: input.key
	});
	return {
		uploadId: String(payload.data.uploadId ?? "").trim(),
		aborted: Boolean(payload.data.aborted)
	};
}

function buildFileQuery(query: FileListQuery): string {
	const params = new URLSearchParams();
	setStringParam(params, "ownerType", query.ownerType);
	setStringParam(params, "ownerId", query.ownerId);
	setStringParam(params, "visibility", query.visibility);
	setStringParam(params, "status", query.status);
	setStringParam(params, "sourceModule", query.sourceModule);
	setStringParam(params, "sourcePluginId", query.sourcePluginId);
	setNumberParam(params, "offset", query.offset, 0);
	setNumberParam(params, "limit", query.limit, 200);
	const raw = params.toString();
	return raw ? `?${raw}` : "";
}

function setStringParam(params: URLSearchParams, key: string, value: string | undefined): void {
	const normalized = value?.trim() ?? "";
	if (normalized !== "") {
		params.set(key, normalized);
	}
}

function setNumberParam(params: URLSearchParams, key: string, value: number | undefined, max: number): void {
	if (typeof value !== "number" || !Number.isFinite(value)) {
		return;
	}
	const normalized = Math.max(0, Math.trunc(value));
	params.set(key, String(max > 0 ? Math.min(normalized, max) : normalized));
}

function setFormValue(form: FormData, key: string, value: string | undefined): void {
	const normalized = value?.trim() ?? "";
	if (normalized !== "") {
		form.set(key, normalized);
	}
}

function normalizeFileListResult(raw: FileListResult): FileListResult {
	return {
		items: Array.isArray(raw.items) ? raw.items.map(normalizeFileObject) : [],
		offset: Number.isFinite(raw.offset) ? raw.offset : 0,
		limit: Number.isFinite(raw.limit) ? raw.limit : 50,
		total: typeof raw.total === "number" && Number.isFinite(raw.total) ? raw.total : undefined
	};
}

function normalizeFileObject(raw: FileObject): FileObject {
	return {
		id: String(raw.id ?? "").trim(),
		key: String(raw.key ?? "").trim(),
		name: String(raw.name ?? "").trim(),
		size: Number.isFinite(raw.size) ? raw.size : 0,
		mime: String(raw.mime ?? "").trim(),
		hash: String(raw.hash ?? "").trim(),
		owner: {
			type: String(raw.owner?.type ?? "").trim(),
			id: String(raw.owner?.id ?? "").trim()
		},
		visibility: normalizeVisibility(raw.visibility),
		storageDriver: String(raw.storageDriver ?? "").trim(),
		status: normalizeStatus(raw.status),
		source: {
			module: String(raw.source?.module ?? "").trim(),
			pluginId: String(raw.source?.pluginId ?? "").trim() || undefined
		},
		metadata: normalizeStringRecord(raw.metadata),
		createdAt: String(raw.createdAt ?? "").trim(),
		updatedAt: String(raw.updatedAt ?? "").trim()
	};
}

function normalizeDownloadResult(raw: FileDownloadResult): FileDownloadResult {
	return {
		id: String(raw.id ?? "").trim(),
		url: String(raw.url ?? "").trim(),
		method: "GET",
		expiresAt: String(raw.expiresAt ?? "").trim(),
		headers: normalizeStringRecord(raw.headers)
	};
}

function normalizeMultipartUpload(raw: MultipartUpload): MultipartUpload {
	return {
		uploadId: String(raw.uploadId ?? "").trim(),
		key: String(raw.key ?? "").trim(),
		expiresAt: String(raw.expiresAt ?? "").trim(),
		metadata: normalizeStringRecord(raw.metadata)
	};
}

function normalizeMultipartPart(raw: MultipartPart): MultipartPart {
	return {
		partNumber: Number.isFinite(raw.partNumber) ? raw.partNumber : 0,
		size: Number.isFinite(raw.size) ? raw.size : 0,
		hash: String(raw.hash ?? "").trim(),
		etag: String(raw.etag ?? "").trim() || undefined
	};
}

function normalizeVisibility(value: unknown): FileVisibility {
	return value === "public" || value === "plugin_asset" ? value : "private";
}

function normalizeStatus(value: unknown): FileStatus {
	return value === "pending" || value === "failed" || value === "deleted" ? value : "available";
}

function normalizeStringRecord(raw: unknown): Record<string, string> | undefined {
	if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
		return undefined;
	}
	const out: Record<string, string> = {};
	for (const [key, value] of Object.entries(raw)) {
		const normalizedKey = key.trim();
		if (normalizedKey !== "" && typeof value === "string") {
			out[normalizedKey] = value;
		}
	}
	return Object.keys(out).length > 0 ? out : undefined;
}

type XHRInput = {
	method: "POST" | "PUT";
	url: string;
	body: Blob | ArrayBuffer | FormData | string;
	contentType?: string;
	signal?: AbortSignal;
	onProgress?: (progress: FileUploadProgress) => void;
};

function sendXHR<T>(input: XHRInput): Promise<T> {
	return new Promise<T>((resolve, reject) => {
		const xhr = new XMLHttpRequest();
		xhr.open(input.method, withAPIPrefix(input.url), true);
		const token = getToken().trim();
		if (token !== "") {
			xhr.setRequestHeader("Authorization", token.toLowerCase().startsWith("bearer ") ? token : `Bearer ${token}`);
		}
		if (input.contentType) {
			xhr.setRequestHeader("Content-Type", input.contentType);
		}
		xhr.upload.onprogress = (event): void => {
			if (!input.onProgress || !event.lengthComputable) {
				return;
			}
			const total = Math.max(event.total, 0);
			const loaded = Math.min(event.loaded, total);
			input.onProgress({
				loaded,
				total,
				percent: total > 0 ? Math.round((loaded / total) * 100) : 0
			});
		};
		xhr.onerror = (): void => reject(new ApiError("network_error", 0, "network_error"));
		xhr.onabort = (): void => reject(new ApiError("request_cancelled", 0, "request_cancelled"));
		xhr.onload = (): void => {
			if (xhr.status < 200 || xhr.status >= 300) {
				reject(apiErrorFromXHR(xhr));
				return;
			}
			try {
				resolve(JSON.parse(xhr.responseText) as T);
			} catch {
				reject(new ApiError("invalid_response", xhr.status, "invalid_response"));
			}
		};
		if (input.signal) {
			if (input.signal.aborted) {
				xhr.abort();
				return;
			}
			input.signal.addEventListener("abort", () => xhr.abort(), { once: true });
		}
		xhr.send(input.body);
	});
}

function apiErrorFromXHR(xhr: XMLHttpRequest): ApiError {
	let payload: Partial<ApiResponse<unknown>> | null = null;
	try {
		payload = JSON.parse(xhr.responseText) as Partial<ApiResponse<unknown>>;
	} catch {
		payload = null;
	}
	const message = typeof payload?.message === "string" && payload.message.trim() !== ""
		? payload.message
		: `request failed: ${xhr.status}`;
	const code = typeof payload?.code === "string" ? payload.code : "";
	return new ApiError(message, xhr.status, code);
}

function withAPIPrefix(url: string): string {
	const normalized = url.trim();
	if (normalized.startsWith("http://") || normalized.startsWith("https://")) {
		return normalized;
	}
	if (normalized.startsWith(`${API_BASE_PREFIX}/`)) {
		return normalized;
	}
	if (normalized.startsWith("/")) {
		return `${API_BASE_PREFIX}${normalized}`;
	}
	return `${API_BASE_PREFIX}/${normalized}`;
}
