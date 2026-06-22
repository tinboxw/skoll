<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { Delete, Download, RefreshCw, Upload } from "lucide-vue-next";

import { confirmAction } from "../../composables/useConfirmAction";
import { downloadBlob } from "../../composables/useDownloadBlob";
import type { FileObject, FileStatus, FileVisibility } from "../../files/api";
import { useI18n } from "../../i18n";
import { useFileStore } from "../../stores/files";
import { toErrorMessage, formatDateTime } from "../../utils/common";

const { t } = useI18n();
const fileStore = useFileStore();

const fileInput = ref<HTMLInputElement | null>(null);
const detailVisible = ref(false);
const selected = ref<FileObject | null>(null);
const filters = reactive<{
	visibility: "" | FileVisibility;
	status: "" | FileStatus;
	keyword: string;
}>({
	visibility: "",
	status: "",
	keyword: ""
});

const filteredItems = computed(() => {
	const keyword = filters.keyword.trim().toLowerCase();
	if (!keyword) {
		return fileStore.items;
	}
	return fileStore.items.filter((item) => {
		return item.name.toLowerCase().includes(keyword) ||
			item.key.toLowerCase().includes(keyword) ||
			item.hash.toLowerCase().includes(keyword) ||
			item.owner.id.toLowerCase().includes(keyword);
	});
});

const uploadTasks = computed(() => Object.values(fileStore.uploadTasks).sort((a, b) => b.startedAt.localeCompare(a.startedAt)));

function statusTagType(status: FileStatus): "success" | "info" | "warning" | "danger" {
	if (status === "available") {
		return "success";
	}
	if (status === "pending") {
		return "warning";
	}
	if (status === "failed") {
		return "danger";
	}
	return "info";
}

function visibilityTagType(visibility: FileVisibility): "primary" | "success" | "warning" {
	if (visibility === "public") {
		return "success";
	}
	if (visibility === "plugin_asset") {
		return "warning";
	}
	return "primary";
}

function formatSize(size: number): string {
	if (!Number.isFinite(size) || size <= 0) {
		return "0 B";
	}
	const units = ["B", "KB", "MB", "GB", "TB"];
	let value = size;
	let index = 0;
	while (value >= 1024 && index < units.length - 1) {
		value /= 1024;
		index += 1;
	}
	return `${value.toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}

async function loadFiles(): Promise<void> {
	try {
		await fileStore.refresh({
			visibility: filters.visibility || undefined,
			status: filters.status || undefined,
			offset: 0,
			limit: 100
		});
	} catch {
		// Store keeps lastError for the page-level alert.
	}
}

function chooseFile(): void {
	fileInput.value?.click();
}

async function handleFileChange(event: Event): Promise<void> {
	const input = event.target as HTMLInputElement;
	const file = input.files?.[0];
	input.value = "";
	if (!file) {
		return;
	}
	try {
		await fileStore.upload({
			file,
			key: `uploads/${new Date().toISOString().slice(0, 10).split("-").join("/")}/${file.name}`,
			name: file.name,
			visibility: "private",
			sourceModule: "system"
		});
		await loadFiles();
	} catch (error) {
		if (!fileStore.lastError) {
			fileStore.lastError = toErrorMessage(error);
		}
	}
}

async function openDetail(row: FileObject): Promise<void> {
	try {
		selected.value = await fileStore.loadDetail(row.id);
		detailVisible.value = true;
	} catch {
		selected.value = row;
		detailVisible.value = true;
	}
}

async function downloadFile(row: FileObject): Promise<void> {
	try {
		const result = await fileStore.requestDownload(row.id);
		if (result.url.startsWith("local://")) {
			return;
		}
		const response = await fetch(result.url, {
			method: result.method,
			headers: result.headers
		});
		if (!response.ok) {
			throw new Error(`download failed: ${response.status}`);
		}
		downloadBlob({ blob: await response.blob(), filename: row.name || row.id });
	} catch (error) {
		fileStore.lastError = toErrorMessage(error);
	}
}

async function removeFile(row: FileObject): Promise<void> {
	const confirmed = await confirmAction({
		title: t("common.confirm"),
		message: `${t("file.deleteConfirm")}: ${row.name}`,
		confirmText: t("common.delete"),
		cancelText: t("common.cancel"),
		danger: true
	});
	if (!confirmed) {
		return;
	}
	try {
		await fileStore.remove(row.id);
	} catch {
		// Store keeps lastError for the page-level alert.
	}
}

onMounted(() => {
	void loadFiles();
});
</script>

<template>
	<section class="file-page">
		<header class="page-header">
			<div>
				<h2>{{ t("page.files") }}</h2>
				<p>{{ t("file.desc") }}</p>
			</div>
			<div class="header-actions">
				<input ref="fileInput" class="native-file-input" type="file" @change="handleFileChange" />
				<el-button :icon="RefreshCw" :loading="fileStore.listStatus === 'loading'" :disabled="fileStore.mutationStatus === 'loading'" @click="loadFiles">
					{{ t("common.refresh") }}
				</el-button>
				<el-button type="primary" :icon="Upload" :disabled="fileStore.listStatus === 'loading'" @click="chooseFile">
					{{ t("file.upload") }}
				</el-button>
			</div>
		</header>

		<el-alert v-if="fileStore.lastError" :title="fileStore.lastError" type="error" show-icon :closable="true" @close="fileStore.clearError()" />

		<section v-if="uploadTasks.length > 0" class="upload-panel">
			<div v-for="task in uploadTasks.slice(0, 4)" :key="task.id" class="upload-task">
				<div>
					<strong>{{ task.name }}</strong>
					<small>{{ task.status === "loading" ? t("file.uploading") : task.status }}</small>
				</div>
				<el-progress :percentage="task.progress.percent" :status="task.status === 'error' ? 'exception' : task.status === 'success' ? 'success' : undefined" />
				<el-button v-if="task.status === 'loading'" link type="danger" @click="fileStore.cancelUpload(task.id)">{{ t("common.cancel") }}</el-button>
				<el-button v-else link @click="fileStore.clearUploadTask(task.id)">{{ t("file.dismiss") }}</el-button>
			</div>
		</section>

		<section class="panel">
			<el-form class="filters" label-position="top" @submit.prevent="loadFiles">
				<el-form-item :label="t('file.keyword')">
					<el-input v-model="filters.keyword" clearable :placeholder="t('file.keywordPlaceholder')" />
				</el-form-item>
				<el-form-item :label="t('file.visibility')">
					<el-select v-model="filters.visibility" clearable :placeholder="t('file.allVisibility')">
						<el-option :label="t('file.visibility.private')" value="private" />
						<el-option :label="t('file.visibility.public')" value="public" />
						<el-option :label="t('file.visibility.pluginAsset')" value="plugin_asset" />
					</el-select>
				</el-form-item>
				<el-form-item :label="t('file.status')">
					<el-select v-model="filters.status" clearable :placeholder="t('file.allStatus')">
						<el-option :label="t('file.status.pending')" value="pending" />
						<el-option :label="t('file.status.available')" value="available" />
						<el-option :label="t('file.status.failed')" value="failed" />
						<el-option :label="t('file.status.deleted')" value="deleted" />
					</el-select>
				</el-form-item>
				<el-form-item>
					<el-button type="primary" native-type="submit" :loading="fileStore.listStatus === 'loading'">{{ t("common.refresh") }}</el-button>
				</el-form-item>
			</el-form>

			<el-table
				v-loading="fileStore.listStatus === 'loading'"
				:data="filteredItems"
				border
				row-key="id"
				:empty-text="fileStore.listStatus === 'error' ? t('error.requestFailed') : t('common.empty')"
				@row-dblclick="openDetail"
			>
				<el-table-column :label="t('file.name')" min-width="220" show-overflow-tooltip>
					<template #default="{ row }">
						<div class="file-cell">
							<strong>{{ row.name }}</strong>
							<small>{{ row.key }}</small>
						</div>
					</template>
				</el-table-column>
				<el-table-column prop="mime" :label="t('file.mime')" min-width="150" show-overflow-tooltip />
				<el-table-column :label="t('file.size')" width="110">
					<template #default="{ row }">{{ formatSize(row.size) }}</template>
				</el-table-column>
				<el-table-column :label="t('file.visibility')" width="130">
					<template #default="{ row }">
						<el-tag :type="visibilityTagType(row.visibility)" effect="plain">{{ t(`file.visibility.${row.visibility === "plugin_asset" ? "pluginAsset" : row.visibility}`) }}</el-tag>
					</template>
				</el-table-column>
				<el-table-column :label="t('file.status')" width="120">
					<template #default="{ row }">
						<el-tag :type="statusTagType(row.status)" effect="plain">{{ t(`file.status.${row.status}`) }}</el-tag>
					</template>
				</el-table-column>
				<el-table-column :label="t('file.owner')" min-width="130" show-overflow-tooltip>
					<template #default="{ row }">{{ row.owner.type }}:{{ row.owner.id }}</template>
				</el-table-column>
				<el-table-column :label="t('file.updatedAt')" width="180">
					<template #default="{ row }">{{ formatDateTime(row.updatedAt) }}</template>
				</el-table-column>
				<el-table-column :label="t('table.actions')" width="190" fixed="right">
					<template #default="{ row }">
						<el-button link @click="openDetail(row)">{{ t("file.detail") }}</el-button>
						<el-button link type="primary" :icon="Download" :disabled="row.status !== 'available'" @click="downloadFile(row)">{{ t("file.download") }}</el-button>
						<el-button link type="danger" :icon="Delete" :disabled="fileStore.mutationStatus === 'loading'" @click="removeFile(row)">{{ t("common.delete") }}</el-button>
					</template>
				</el-table-column>
			</el-table>
		</section>

		<el-drawer v-model="detailVisible" :title="selected?.name || t('file.detail')" size="420px">
			<el-descriptions v-if="selected" :column="1" border>
				<el-descriptions-item :label="t('file.id')">{{ selected.id }}</el-descriptions-item>
				<el-descriptions-item :label="t('file.key')">{{ selected.key }}</el-descriptions-item>
				<el-descriptions-item :label="t('file.hash')">{{ selected.hash }}</el-descriptions-item>
				<el-descriptions-item :label="t('file.mime')">{{ selected.mime }}</el-descriptions-item>
				<el-descriptions-item :label="t('file.size')">{{ formatSize(selected.size) }}</el-descriptions-item>
				<el-descriptions-item :label="t('file.source')">{{ selected.source.module }}{{ selected.source.pluginId ? ` / ${selected.source.pluginId}` : "" }}</el-descriptions-item>
				<el-descriptions-item :label="t('file.createdAt')">{{ formatDateTime(selected.createdAt) }}</el-descriptions-item>
				<el-descriptions-item :label="t('file.updatedAt')">{{ formatDateTime(selected.updatedAt) }}</el-descriptions-item>
			</el-descriptions>
			<el-empty v-else :description="t('common.empty')" />
		</el-drawer>
	</section>
</template>

<style scoped>
.file-page {
	display: grid;
	gap: 14px;
}

.page-header {
	display: flex;
	align-items: flex-start;
	justify-content: space-between;
	gap: 16px;
}

.page-header h2 {
	margin: 0;
	font-size: 1.35rem;
}

.page-header p {
	margin: 6px 0 0;
	color: var(--color-text-muted);
}

.header-actions {
	display: flex;
	align-items: center;
	gap: 8px;
}

.native-file-input {
	display: none;
}

.upload-panel,
.panel {
	display: grid;
	gap: 12px;
	padding: 16px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.upload-task {
	display: grid;
	grid-template-columns: minmax(160px, 240px) minmax(180px, 1fr) auto;
	align-items: center;
	gap: 12px;
}

.upload-task strong,
.file-cell strong {
	display: block;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.upload-task small,
.file-cell small {
	display: block;
	margin-top: 3px;
	color: var(--color-text-muted);
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.filters {
	display: grid;
	grid-template-columns: minmax(220px, 1fr) 180px 180px auto;
	gap: 12px;
	align-items: end;
}

@media (max-width: 980px) {
	.page-header,
	.header-actions,
	.filters,
	.upload-task {
		display: grid;
		grid-template-columns: 1fr;
	}
}
</style>
