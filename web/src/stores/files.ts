import { defineStore } from "pinia";

import { ApiError } from "../utils/api";
import {
	abortMultipartUpload,
	completeMultipartUpload,
	createFileDownload,
	deleteFile,
	getFile,
	initMultipartUpload,
	listFiles,
	uploadFile,
	uploadMultipartPart,
	type FileDownloadResult,
	type FileListQuery,
	type FileObject,
	type FileUploadInput,
	type FileUploadProgress,
	type MultipartCompleteInput,
	type MultipartInitInput,
	type MultipartInitResult,
	type MultipartPart,
	type MultipartPartInput
} from "../files/api";

const uploadControllers = new Map<string, AbortController>();

type FileOperationStatus = "idle" | "loading" | "success" | "error";
type FileUploadStatus = FileOperationStatus | "cancelled";

type FileUploadTask = {
	id: string;
	name: string;
	status: FileUploadStatus;
	progress: FileUploadProgress;
	error: string | null;
	fileId: string | null;
	startedAt: string;
	finishedAt: string | null;
};

type FileStoreState = {
	items: FileObject[];
	selected: FileObject | null;
	query: FileListQuery;
	listStatus: FileOperationStatus;
	detailStatus: FileOperationStatus;
	mutationStatus: FileOperationStatus;
	lastError: string | null;
	lastRefreshedAt: string | null;
	uploadTasks: Record<string, FileUploadTask>;
	download: FileDownloadResult | null;
	multipart: MultipartInitResult | null;
};

export const useFileStore = defineStore("files", {
	state: (): FileStoreState => ({
		items: [],
		selected: null,
		query: {
			offset: 0,
			limit: 50
		},
		listStatus: "idle",
		detailStatus: "idle",
		mutationStatus: "idle",
		lastError: null,
		lastRefreshedAt: null,
		uploadTasks: {},
		download: null,
		multipart: null
	}),
	getters: {
		isLoading: (state): boolean => state.listStatus === "loading" || state.detailStatus === "loading" || state.mutationStatus === "loading",
		hasError: (state): boolean => state.lastError !== null,
		uploadingTasks: (state): FileUploadTask[] => Object.values(state.uploadTasks).filter((task) => task.status === "loading"),
		failedUploadTasks: (state): FileUploadTask[] => Object.values(state.uploadTasks).filter((task) => task.status === "error"),
		hasActiveUploads(): boolean {
			return this.uploadingTasks.length > 0;
		}
	},
	actions: {
		async refresh(query?: FileListQuery): Promise<FileObject[]> {
			this.listStatus = "loading";
			this.lastError = null;
			this.query = normalizeQuery(query ?? this.query);
			try {
				const result = await listFiles(this.query);
				this.items = result.items;
				this.listStatus = "success";
				this.lastRefreshedAt = new Date().toISOString();
				return this.items;
			} catch (error) {
				this.listStatus = "error";
				this.lastError = errorMessage(error);
				throw error;
			}
		},
		async loadDetail(id: string): Promise<FileObject> {
			this.detailStatus = "loading";
			this.lastError = null;
			try {
				const item = await getFile(id);
				this.selected = item;
				this.detailStatus = "success";
				upsertItem(this.items, item);
				return item;
			} catch (error) {
				this.detailStatus = "error";
				this.lastError = errorMessage(error);
				throw error;
			}
		},
		async upload(input: Omit<FileUploadInput, "signal" | "onProgress">, taskId = createUploadTaskID()): Promise<FileObject> {
			const controller = new AbortController();
			uploadControllers.set(taskId, controller);
			this.uploadTasks[taskId] = createUploadTask(taskId, resolveUploadName(input.file, input.name));
			try {
				const item = await uploadFile({
					...input,
					signal: controller.signal,
					onProgress: (progress) => this.updateUploadProgress(taskId, progress)
				});
				uploadControllers.delete(taskId);
				this.finishUploadTask(taskId, "success", null, item.id);
				upsertItem(this.items, item);
				this.lastRefreshedAt = new Date().toISOString();
				return item;
			} catch (error) {
				const cancelled = error instanceof ApiError && error.code === "request_cancelled";
				uploadControllers.delete(taskId);
				this.finishUploadTask(taskId, cancelled ? "cancelled" : "error", cancelled ? null : errorMessage(error), null);
				if (!cancelled) {
					this.lastError = errorMessage(error);
				}
				throw error;
			}
		},
		cancelUpload(taskId: string): void {
			const task = this.uploadTasks[taskId];
			if (!task || task.status !== "loading") {
				return;
			}
			uploadControllers.get(taskId)?.abort();
			uploadControllers.delete(taskId);
			this.finishUploadTask(taskId, "cancelled", null, task.fileId);
		},
		async requestDownload(id: string): Promise<FileDownloadResult> {
			this.mutationStatus = "loading";
			this.lastError = null;
			try {
				const result = await createFileDownload(id);
				this.download = result;
				this.mutationStatus = "success";
				return result;
			} catch (error) {
				this.mutationStatus = "error";
				this.lastError = errorMessage(error);
				throw error;
			}
		},
		async remove(id: string): Promise<void> {
			this.mutationStatus = "loading";
			this.lastError = null;
			try {
				await deleteFile(id);
				this.items = this.items.filter((item) => item.id !== id);
				if (this.selected?.id === id) {
					this.selected = null;
				}
				this.mutationStatus = "success";
			} catch (error) {
				this.mutationStatus = "error";
				this.lastError = errorMessage(error);
				throw error;
			}
		},
		async beginMultipart(input: MultipartInitInput): Promise<MultipartInitResult> {
			this.mutationStatus = "loading";
			this.lastError = null;
			try {
				const result = await initMultipartUpload(input);
				this.multipart = result;
				upsertItem(this.items, result.item);
				this.mutationStatus = "success";
				return result;
			} catch (error) {
				this.mutationStatus = "error";
				this.lastError = errorMessage(error);
				throw error;
			}
		},
		async uploadPart(input: Omit<MultipartPartInput, "signal" | "onProgress">, taskId = createUploadTaskID()): Promise<MultipartPart> {
			const controller = new AbortController();
			uploadControllers.set(taskId, controller);
			this.uploadTasks[taskId] = createUploadTask(taskId, `${input.key}#${input.partNumber}`);
			try {
				const part = await uploadMultipartPart({
					...input,
					signal: controller.signal,
					onProgress: (progress) => this.updateUploadProgress(taskId, progress)
				});
				uploadControllers.delete(taskId);
				this.finishUploadTask(taskId, "success", null, null);
				return part;
			} catch (error) {
				const cancelled = error instanceof ApiError && error.code === "request_cancelled";
				uploadControllers.delete(taskId);
				this.finishUploadTask(taskId, cancelled ? "cancelled" : "error", cancelled ? null : errorMessage(error), null);
				if (!cancelled) {
					this.lastError = errorMessage(error);
				}
				throw error;
			}
		},
		async completeMultipart(input: MultipartCompleteInput): Promise<FileObject> {
			this.mutationStatus = "loading";
			this.lastError = null;
			try {
				const item = await completeMultipartUpload(input);
				this.multipart = null;
				upsertItem(this.items, item);
				this.mutationStatus = "success";
				return item;
			} catch (error) {
				this.mutationStatus = "error";
				this.lastError = errorMessage(error);
				throw error;
			}
		},
		async abortMultipart(uploadId: string, fileId?: string, key?: string): Promise<void> {
			this.mutationStatus = "loading";
			this.lastError = null;
			try {
				await abortMultipartUpload({ uploadId, fileId, key });
				this.multipart = null;
				this.mutationStatus = "success";
			} catch (error) {
				this.mutationStatus = "error";
				this.lastError = errorMessage(error);
				throw error;
			}
		},
		updateUploadProgress(taskId: string, progress: FileUploadProgress): void {
			const task = this.uploadTasks[taskId];
			if (!task) {
				return;
			}
			task.progress = progress;
		},
		finishUploadTask(taskId: string, status: FileUploadStatus, error: string | null, fileId: string | null): void {
			const task = this.uploadTasks[taskId];
			if (!task) {
				return;
			}
			task.status = status;
			task.error = error;
			task.fileId = fileId;
			task.finishedAt = new Date().toISOString();
			if (status === "success") {
				task.progress = { loaded: task.progress.total, total: task.progress.total, percent: 100 };
			}
		},
		clearError(): void {
			this.lastError = null;
		},
		clearUploadTask(taskId: string): void {
			uploadControllers.get(taskId)?.abort();
			uploadControllers.delete(taskId);
			delete this.uploadTasks[taskId];
		}
	}
});

function normalizeQuery(query: FileListQuery): FileListQuery {
	return {
		...query,
		offset: typeof query.offset === "number" && Number.isFinite(query.offset) ? Math.max(0, Math.trunc(query.offset)) : 0,
		limit: typeof query.limit === "number" && Number.isFinite(query.limit) ? Math.min(Math.max(1, Math.trunc(query.limit)), 200) : 50
	};
}

function upsertItem(items: FileObject[], item: FileObject): void {
	const index = items.findIndex((candidate) => candidate.id === item.id);
	if (index >= 0) {
		items[index] = item;
		return;
	}
	items.unshift(item);
}

function createUploadTaskID(): string {
	return `upload-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function createUploadTask(id: string, name: string): FileUploadTask {
	return {
		id,
		name,
		status: "loading",
		progress: { loaded: 0, total: 0, percent: 0 },
		error: null,
		fileId: null,
		startedAt: new Date().toISOString(),
		finishedAt: null
	};
}

function resolveUploadName(file: File | Blob, fallback?: string): string {
	const namedFile = file instanceof File ? file.name : "";
	return fallback?.trim() || namedFile || "upload";
}

function errorMessage(error: unknown): string {
	if (error instanceof Error && error.message.trim() !== "") {
		return error.message;
	}
	return "request failed";
}
