<script setup lang="ts">
import { ref } from "vue";
import * as XLSX from "xlsx";
import { useRoute, useRouter } from "vue-router";

import { useI18n } from "../../i18n";
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
const router = useRouter();
const route = useRoute();
const rows = ref<BatchUserRow[]>([{ id: crypto.randomUUID(), account: "", name: "", email: "", password: "" }]);
const importing = ref(false);
const error = ref("");
const results = ref<ImportResult[]>([]);
const atomic = ref(false);

function resolveReturnTo(): string {
	const value = route.query.returnTo;
	if (typeof value === "string" && value.trim() !== "") {
		return value;
	}
	return "/user";
}

async function backToList(): Promise<void> {
	await router.push(resolveReturnTo());
}

function addRow(): void {
	rows.value.push({ id: crypto.randomUUID(), account: "", name: "", email: "", password: "" });
}

function removeRow(id: string): void {
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

async function importExcel(event: Event): Promise<void> {
	error.value = "";
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
				account: String(kv.get("account") ?? kv.get("用户名") ?? kv.get("账号") ?? "").trim(),
				name: String(kv.get("name") ?? kv.get("姓名") ?? "").trim(),
				email: String(kv.get("email") ?? kv.get("邮箱") ?? "").trim(),
				password: String(kv.get("password") ?? kv.get("密码") ?? "").trim()
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
	<section>
		<h2>{{ t("page.userBatch") }}</h2>
		<section class="create-panel">
			<h3>{{ t("batchUser.submit") }}</h3>
			<p>{{ t("batchUser.desc") }}</p>
			<p v-if="error" class="error">{{ error }}</p>
			<div class="toolbar">
				<button type="button" :disabled="importing" @click="addRow">{{ t("batchUser.addRow") }}</button>
				<label class="upload-btn">
					<span>{{ t("batchUser.importExcel") }}</span>
					<input type="file" accept=".xlsx,.xls" @change="importExcel" />
				</label>
				<label class="atomic-flag">
					<input v-model="atomic" type="checkbox" :disabled="importing" />
					<span>{{ t("batchUser.atomic") }}</span>
				</label>
				<button type="button" :disabled="importing" @click="submitBatch">{{ importing ? t("common.loading") : t("batchUser.submit") }}</button>
				<button type="button" :disabled="importing" @click="backToList">{{ t("common.backToList") }}</button>
			</div>

			<table>
				<thead>
					<tr>
						<th>{{ t("user.account") }}</th>
						<th>{{ t("table.name") }}</th>
						<th>{{ t("table.email") }}</th>
						<th>{{ t("profile.newPassword") }}</th>
						<th>{{ t("table.actions") }}</th>
					</tr>
				</thead>
				<tbody>
					<tr v-for="item in rows" :key="item.id">
						<td><input v-model="item.account" type="text" :disabled="importing" /></td>
						<td><input v-model="item.name" type="text" :disabled="importing" /></td>
						<td><input v-model="item.email" type="email" :disabled="importing" /></td>
						<td><input v-model="item.password" type="text" :disabled="importing" /></td>
						<td><button type="button" :disabled="importing || rows.length <= 1" @click="removeRow(item.id)">{{ t("common.delete") }}</button></td>
					</tr>
				</tbody>
			</table>

			<section v-if="results.length > 0" class="result-panel">
				<h3>{{ t("batchUser.result") }}</h3>
				<ul>
					<li v-for="item in results" :key="`${item.account}-${item.message}`" :class="item.success ? 'ok' : 'fail'">
						{{ item.account }} - {{ item.message }}
					</li>
				</ul>
			</section>
		</section>
	</section>
</template>

<style scoped>
.create-panel {
	margin: 12px 0;
	padding: 12px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
}

.create-panel h3 {
	margin: 0 0 8px;
}

.toolbar {
	display: flex;
	gap: 8px;
	align-items: center;
	margin-bottom: 12px;
}

.upload-btn {
	position: relative;
	overflow: hidden;
	padding: 7px 10px;
	border-radius: var(--radius-md);
	border: 1px solid var(--color-border);
	background: var(--color-surface-soft);
	cursor: pointer;
}

.atomic-flag {
	display: inline-flex;
	align-items: center;
	gap: 6px;
	height: 32px;
	padding: 0 8px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.atomic-flag input {
	width: 14px;
	height: 14px;
}

.upload-btn input {
	position: absolute;
	top: 0;
	left: 0;
	opacity: 0;
	width: 100%;
	height: 100%;
	cursor: pointer;
}

button {
	height: 32px;
	padding: 0 10px;
	border-radius: var(--radius-md);
	border: 1px solid var(--color-border);
	background: var(--color-surface-soft);
	cursor: pointer;
}

table {
	width: 100%;
	border-collapse: collapse;
}

th,
td {
	padding: 8px;
	border-bottom: 1px solid var(--color-border);
}

input {
	width: 100%;
	height: 32px;
	padding: 0 8px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
}

.result-panel {
	margin-top: 12px;
	padding: 12px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
}

.result-panel ul {
	margin: 0;
	padding-left: 18px;
}

.ok {
	color: var(--color-success);
}

.fail,
.error {
	color: var(--color-danger);
}
</style>
