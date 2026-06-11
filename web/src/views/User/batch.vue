<script setup lang="ts">
import { computed, ref } from "vue";
import * as XLSX from "xlsx";
import { useRoute, useRouter } from "vue-router";

import { useI18n } from "../../i18n";
import { useAccess } from "../../permissions/access";
import { apiPost, type ApiResponse } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";

type BatchUserRow = {
	id: string;
	account: string;
	name: string;
	email: string;
	password: string;
};

type ImportResult = {
	account: string;
	success: boolean;
	message: string;
};

type BatchCreatePayload = {
	results?: ImportResult[];
};

const { t } = useI18n();
const access = useAccess();
const router = useRouter();
const route = useRoute();
const rows = ref<BatchUserRow[]>([{ id: crypto.randomUUID(), account: "", name: "", email: "", password: "" }]);
const importing = ref(false);
const error = ref("");
const results = ref<ImportResult[]>([]);
const atomic = ref(false);
const canCreateUser = computed(() => access.can("user.create"));

const importHeaderAliases = {
	account: new Set(["account", "\u7528\u6237\u540d", "\u8d26\u53f7"]),
	name: new Set(["name", "\u59d3\u540d"]),
	email: new Set(["email", "\u90ae\u7bb1"]),
	password: new Set(["password", "\u5bc6\u7801"])
};

function resolveReturnTo(): string {
	const value = route.query.returnTo;
	if (typeof value === "string" && value.trim() !== "") {
		return value;
	}
	return "/skoll/user";
}

async function backToList(): Promise<void> {
	await router.push(resolveReturnTo());
}

function addRow(): void {
	if (!canCreateUser.value) {
		error.value = t("error.forbidden");
		return;
	}
	rows.value.push({ id: crypto.randomUUID(), account: "", name: "", email: "", password: "" });
}

function removeRow(id: string): void {
	if (!canCreateUser.value) {
		error.value = t("error.forbidden");
		return;
	}
	if (rows.value.length <= 1) {
		return;
	}
	rows.value = rows.value.filter((item) => item.id !== id);
}

function validateRow(row: BatchUserRow): string {
	if (row.account.trim() === "") {
		return t("batchUser.accountRequired");
	}
	if (row.name.trim() === "") {
		return t("batchUser.nameRequired");
	}
	if (row.email.trim() === "" || !row.email.includes("@")) {
		return t("batchUser.emailInvalid");
	}
	if (row.password.trim().length < 8) {
		return t("batchUser.passwordInvalid");
	}
	return "";
}

async function submitBatch(): Promise<void> {
	error.value = "";
	results.value = [];
	if (!canCreateUser.value) {
		error.value = t("error.forbidden");
		return;
	}
	importing.value = true;
	try {
		const nextResults: ImportResult[] = [];
		const validItems: Array<{ account: string; name: string; email: string; passwordHash: string }> = [];
		for (const item of rows.value) {
			const errMsg = validateRow(item);
			if (errMsg !== "") {
				nextResults.push({ account: item.account || "-", success: false, message: errMsg });
				continue;
			}
			validItems.push({
				account: item.account.trim(),
				name: item.name.trim(),
				email: item.email.trim(),
				passwordHash: item.password.trim()
			});
		}

		if (validItems.length > 0) {
			try {
				const payload = await apiPost<ApiResponse<BatchCreatePayload>>("/v1/users/batch", { items: validItems, atomic: atomic.value });
				const remoteResults = Array.isArray(payload.data?.results) ? payload.data.results : [];
				nextResults.push(...remoteResults.map((item) => ({
					account: String(item.account ?? "-"),
					success: Boolean(item.success),
					message: String(item.message ?? (item.success ? t("batchUser.createOk") : t("error.requestFailed")))
				})));
			} catch (e) {
				nextResults.push(...validItems.map((item) => ({
					account: item.account,
					success: false,
					message: toErrorMessage(e)
				})));
			}
		}
		results.value = nextResults;
	} finally {
		importing.value = false;
	}
}

function normalizeKey(raw: string): string {
	return raw.toLowerCase().replace(/\s+/g, "").trim();
}

function pickImportValue(kv: Map<string, unknown>, aliases: Set<string>): string {
	for (const alias of aliases) {
		const value = kv.get(normalizeKey(alias));
		if (value !== undefined && value !== null) {
			return String(value).trim();
		}
	}
	return "";
}

async function importExcel(event: Event): Promise<void> {
	error.value = "";
	if (!canCreateUser.value) {
		error.value = t("error.forbidden");
		return;
	}
	const input = event.target as HTMLInputElement;
	const file = input.files?.[0];
	if (!file) {
		return;
	}
	try {
		const buffer = await file.arrayBuffer();
		const workbook = XLSX.read(buffer, { type: "array" });
		const sheetName = workbook.SheetNames[0];
		if (!sheetName) {
			throw new Error(t("batchUser.emptySheet"));
		}
		const sheet = workbook.Sheets[sheetName];
		const rawRows = XLSX.utils.sheet_to_json<Record<string, unknown>>(sheet, { defval: "" });
		const parsed: BatchUserRow[] = rawRows.map((item) => {
			const kv = new Map<string, unknown>();
			for (const [k, v] of Object.entries(item)) {
				kv.set(normalizeKey(k), v);
			}
			return {
				id: crypto.randomUUID(),
				account: pickImportValue(kv, importHeaderAliases.account),
				name: pickImportValue(kv, importHeaderAliases.name),
				email: pickImportValue(kv, importHeaderAliases.email),
				password: pickImportValue(kv, importHeaderAliases.password)
			};
		});
		rows.value = parsed.length > 0 ? parsed : [{ id: crypto.randomUUID(), account: "", name: "", email: "", password: "" }];
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		input.value = "";
	}
}
</script>

<template>
	<section class="batch-user-page">
		<header class="page-header">
			<div>
				<h2>{{ t("page.userBatch") }}</h2>
				<p>{{ t("batchUser.desc") }}</p>
			</div>
			<el-button :disabled="importing" @click="backToList">{{ t("common.backToList") }}</el-button>
		</header>

		<el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />

		<section class="panel">
			<div class="toolbar">
				<el-button type="primary" :disabled="importing || !canCreateUser" @click="addRow">{{ t("batchUser.addRow") }}</el-button>
				<label class="upload-btn" :class="{ disabled: importing || !canCreateUser }">
					<el-button :disabled="importing || !canCreateUser">{{ t("batchUser.importExcel") }}</el-button>
					<input type="file" accept=".xlsx,.xls" :disabled="importing || !canCreateUser" @change="importExcel" />
				</label>
				<el-checkbox v-model="atomic" :disabled="importing || !canCreateUser">{{ t("batchUser.atomic") }}</el-checkbox>
				<el-button type="primary" :loading="importing" :disabled="!canCreateUser" @click="submitBatch">{{ t("batchUser.submit") }}</el-button>
			</div>

			<el-table :data="rows" border row-key="id" :empty-text="t('common.empty')">
				<el-table-column :label="t('user.account')" min-width="170">
					<template #default="{ row }">
						<el-input v-model="row.account" :disabled="importing || !canCreateUser" />
					</template>
				</el-table-column>
				<el-table-column :label="t('table.name')" min-width="160">
					<template #default="{ row }">
						<el-input v-model="row.name" :disabled="importing || !canCreateUser" />
					</template>
				</el-table-column>
				<el-table-column :label="t('table.email')" min-width="210">
					<template #default="{ row }">
						<el-input v-model="row.email" type="email" :disabled="importing || !canCreateUser" />
					</template>
				</el-table-column>
				<el-table-column :label="t('profile.newPassword')" min-width="180">
					<template #default="{ row }">
						<el-input v-model="row.password" show-password :disabled="importing || !canCreateUser" />
					</template>
				</el-table-column>
				<el-table-column :label="t('table.actions')" width="120" fixed="right">
					<template #default="{ row }">
						<el-button link type="danger" :disabled="importing || !canCreateUser || rows.length <= 1" @click="removeRow(row.id)">
							{{ t("common.delete") }}
						</el-button>
					</template>
				</el-table-column>
			</el-table>
		</section>

		<section v-if="results.length > 0" class="panel">
			<h3>{{ t("batchUser.result") }}</h3>
			<el-table :data="results" border :empty-text="t('common.empty')">
				<el-table-column prop="account" :label="t('user.account')" min-width="170" />
				<el-table-column :label="t('table.status')" width="120">
					<template #default="{ row }">
						<el-tag :type="row.success ? 'success' : 'danger'" effect="plain">
							{{ row.success ? t("batchUser.createOk") : t("error.requestFailed") }}
						</el-tag>
					</template>
				</el-table-column>
				<el-table-column prop="message" :label="t('batchUser.result')" min-width="260" show-overflow-tooltip />
			</el-table>
		</section>
	</section>
</template>

<style scoped>
.batch-user-page {
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

.panel {
	display: grid;
	gap: 12px;
	padding: 16px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.panel h3 {
	margin: 0;
	font-size: 1rem;
}

.toolbar {
	display: flex;
	flex-wrap: wrap;
	gap: 8px;
	align-items: center;
}

.upload-btn {
	position: relative;
	display: inline-flex;
	overflow: hidden;
}

.upload-btn.disabled {
	opacity: 0.7;
}

.upload-btn input {
	position: absolute;
	inset: 0;
	width: 100%;
	height: 100%;
	opacity: 0;
	cursor: pointer;
}

.upload-btn.disabled input {
	cursor: default;
}

@media (max-width: 760px) {
	.page-header,
	.toolbar {
		display: grid;
		justify-content: stretch;
	}
}
</style>
