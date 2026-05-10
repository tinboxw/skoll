<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute } from "vue-router";
import * as XLSX from "xlsx";

import { useI18n } from "../../i18n";
import { type ApiResponse, apiPost } from "../../utils/api";
import { apiDelete, apiGet } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";

type UserRecord = {
	id: string;
	account: string;
	name?: string;
	email: string;
	status?: string;
};

function normalizeUserRecord(item: unknown): UserRecord | null {
	if (!item || typeof item !== "object") {
		return null;
	}
	const row = item as Record<string, unknown>;
	const id = String(row.id ?? row.ID ?? "").trim();
	if (!id) {
		return null;
	}
	return {
		id,
		account: String(row.account ?? row.Account ?? "").trim(),
		name: String(row.name ?? row.Name ?? "").trim(),
		email: String(row.email ?? row.Email ?? "").trim(),
		status: String(row.status ?? row.Status ?? "").trim()
	};
}

const { t } = useI18n();
const route = useRoute();

const loading = ref(false);
const error = ref("");
const operating = ref(false);
const rows = ref<UserRecord[]>([]);
const page = ref(1);
const pageSize = 10;
const canGoNext = ref(false);
const lastCreatedAccount = ref("");

const createOpen = ref(false);
const createLoading = ref(false);
const createError = ref("");
const createAccount = ref("");
const createName = ref("");
const createEmail = ref("");
const createPassword = ref("");

type BatchUserRow = {
	id: string;
	account: string;
	name: string;
	email: string;
	password: string;
};

type BatchResult = {
	account: string;
	success: boolean;
	message: string;
};

const batchOpen = ref(false);
const batchLoading = ref(false);
const batchError = ref("");
const batchRows = ref<BatchUserRow[]>([]);
const batchResults = ref<BatchResult[]>([]);

const hasRows = computed(() => rows.value.length > 0);
const returnTo = computed(() => route.fullPath || "/user");

async function loadUsers(targetPage = page.value): Promise<void> {
	loading.value = true;
	error.value = "";
	try {
		const safePage = Math.max(1, targetPage);
		const offset = (safePage - 1) * pageSize;
		const payload = await apiGet<ApiResponse<UserRecord[]>>(`/v1/users?offset=${offset}&limit=${pageSize}`);
		const list = Array.isArray(payload.data)
			? payload.data.map((item) => normalizeUserRecord(item)).filter((item): item is UserRecord => item !== null)
			: [];
		rows.value = list;
		page.value = safePage;
		canGoNext.value = list.length >= pageSize;
	} catch (e) {
		error.value = toErrorMessage(e);
		rows.value = [];
		canGoNext.value = false;
	}
	finally {
		loading.value = false;
	}
}

async function deleteUser(userID: string): Promise<void> {
	if (operating.value || loading.value) {
		return;
	}
	operating.value = true;
	error.value = "";
	try {
		await apiDelete<ApiResponse<unknown>>(`/v1/users/${userID}`);
		await loadUsers(page.value);
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		operating.value = false;
	}
}

function openCreateDrawer(): void {
	createOpen.value = true;
	createError.value = "";
}

function closeCreateDrawer(): void {
	createOpen.value = false;
	createError.value = "";
	createAccount.value = "";
	createName.value = "";
	createEmail.value = "";
	createPassword.value = "";
}

function validateCreateForm(): string {
	if (createAccount.value.trim() === "") {
		return t("user.accountRequired");
	}
	if (createEmail.value.trim() === "" || !createEmail.value.includes("@")) {
		return t("user.emailInvalid");
	}
	if (createPassword.value.trim().length < 8) {
		return t("user.passwordInvalid");
	}
	return "";
}

async function submitCreate(): Promise<void> {
	createError.value = "";
	const msg = validateCreateForm();
	if (msg !== "") {
		createError.value = msg;
		return;
	}
	createLoading.value = true;
	try {
		await apiPost<ApiResponse<unknown>>("/v1/users", {
			account: createAccount.value.trim(),
			name: createName.value.trim() || createAccount.value.trim(),
			email: createEmail.value.trim(),
			passwordHash: createPassword.value.trim()
		});
		lastCreatedAccount.value = createAccount.value.trim();
		await loadUsers(page.value);
		closeCreateDrawer();
	} catch (e) {
		createError.value = toErrorMessage(e);
	} finally {
		createLoading.value = false;
	}
}

function openBatchModal(): void {
	batchOpen.value = true;
	batchError.value = "";
	batchResults.value = [];
	batchRows.value = [{ id: crypto.randomUUID(), account: "", name: "", email: "", password: "" }];
}

function closeBatchModal(): void {
	batchOpen.value = false;
	batchError.value = "";
	batchResults.value = [];
	batchRows.value = [];
}

function addBatchRow(): void {
	batchRows.value.push({ id: crypto.randomUUID(), account: "", name: "", email: "", password: "" });
}

function removeBatchRow(id: string): void {
	if (batchRows.value.length <= 1) {
		return;
	}
	batchRows.value = batchRows.value.filter((item) => item.id !== id);
}

function validateBatchRow(row: BatchUserRow): string {
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
	batchError.value = "";
	batchResults.value = [];
	batchLoading.value = true;
	let firstSuccessAccount = "";
	try {
		const nextResults: BatchResult[] = [];
		for (const item of batchRows.value) {
			const msg = validateBatchRow(item);
			if (msg !== "") {
				nextResults.push({ account: item.account || "-", success: false, message: msg });
				continue;
			}
			try {
				await apiPost<ApiResponse<unknown>>("/v1/users", {
					account: item.account.trim(),
					name: item.name.trim(),
					email: item.email.trim(),
					passwordHash: item.password.trim()
				});
				nextResults.push({ account: item.account.trim(), success: true, message: t("batchUser.createOk") });
				if (firstSuccessAccount === "") {
					firstSuccessAccount = item.account.trim();
				}
			} catch (e) {
				nextResults.push({ account: item.account.trim(), success: false, message: toErrorMessage(e) });
			}
		}
		batchResults.value = nextResults;
		if (firstSuccessAccount !== "") {
			lastCreatedAccount.value = firstSuccessAccount;
			await loadUsers(page.value);
		}
	} finally {
		batchLoading.value = false;
	}
}

function normalizeExcelKey(raw: string): string {
	return raw.toLowerCase().replace(/\s+/g, "").trim();
}

async function importExcel(event: Event): Promise<void> {
	batchError.value = "";
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
				kv.set(normalizeExcelKey(k), v);
			}
			return {
				id: crypto.randomUUID(),
				account: String(kv.get("account") ?? kv.get("用户名") ?? kv.get("账号") ?? "").trim(),
				name: String(kv.get("name") ?? kv.get("姓名") ?? "").trim(),
				email: String(kv.get("email") ?? kv.get("邮箱") ?? "").trim(),
				password: String(kv.get("password") ?? kv.get("密码") ?? "").trim()
			};
		});
		batchRows.value = parsed.length > 0 ? parsed : [{ id: crypto.randomUUID(), account: "", name: "", email: "", password: "" }];
	} catch (e) {
		batchError.value = toErrorMessage(e);
	} finally {
		input.value = "";
	}
}

function isNewRow(item: UserRecord): boolean {
	return lastCreatedAccount.value !== "" && item.account === lastCreatedAccount.value;
}

function prevPage(): void {
	if (loading.value || page.value <= 1) {
		return;
	}
	void loadUsers(page.value - 1);
}

function nextPage(): void {
	if (loading.value || !canGoNext.value) {
		return;
	}
	void loadUsers(page.value + 1);
}

void loadUsers(1);
</script>

<template>
	<section>
		<h2>{{ t("page.users") }}</h2>
		<p v-if="error" class="error">{{ error }}</p>
		<div class="toolbar">
			<div class="toolbar-actions">
				<button type="button" class="button-link" :disabled="loading || operating" @click="openCreateDrawer">{{ t("user.create") }}</button>
				<button type="button" class="button-link" :disabled="loading || operating" @click="openBatchModal">{{ t("user.batchCreate") }}</button>
				<button type="button" :disabled="loading || operating" @click="loadUsers(page)">{{ loading ? t("common.loading") : t("common.refresh") }}</button>
			</div>
			<div class="pager">
				<button type="button" :disabled="loading || operating || page <= 1" @click="prevPage">{{ t("common.prev") }}</button>
				<span>{{ t("common.page") }} {{ page }}</span>
				<button type="button" :disabled="loading || operating || !canGoNext" @click="nextPage">{{ t("common.next") }}</button>
			</div>
		</div>
		<table>
			<thead>
				<tr>
					<th>{{ t("table.id") }}</th>
					<th>{{ t("table.name") }}</th>
					<th>{{ t("table.email") }}</th>
					<th>{{ t("table.status") }}</th>
					<th>{{ t("table.actions") }}</th>
				</tr>
			</thead>
			<tbody v-if="hasRows">
				<tr v-for="item in rows" :key="item.id" :class="{ highlighted: isNewRow(item) }">
					<td>{{ item.id }}</td>
					<td>{{ item.name || item.account }}</td>
					<td>{{ item.email }}</td>
					<td>{{ item.status || "-" }}</td>
					<td class="actions">
						<router-link :to="{ path: `/user/${item.id}/edit`, query: { returnTo } }">{{ t("common.edit") }}</router-link>
						<button type="button" :disabled="operating || loading" @click="deleteUser(item.id)">{{ t("common.delete") }}</button>
					</td>
				</tr>
			</tbody>
			<tbody v-else>
				<tr>
					<td colspan="5">{{ loading ? t("common.loading") : t("common.empty") }}</td>
				</tr>
			</tbody>
		</table>

		<div v-if="createOpen" class="overlay" @click.self="closeCreateDrawer">
			<aside class="drawer">
				<div class="drawer-header">
					<h3>{{ t("page.userAdd") }}</h3>
					<button type="button" :disabled="createLoading" @click="closeCreateDrawer">X</button>
				</div>
				<p class="hint">{{ t("user.addHint") }}</p>
				<form class="drawer-form" @submit.prevent="submitCreate">
					<label>
						<span>{{ t("user.account") }}</span>
						<input v-model="createAccount" type="text" required :disabled="createLoading" />
					</label>
					<label>
						<span>{{ t("table.name") }}</span>
						<input v-model="createName" type="text" :disabled="createLoading" />
					</label>
					<label>
						<span>{{ t("table.email") }}</span>
						<input v-model="createEmail" type="email" required :disabled="createLoading" />
					</label>
					<label>
						<span>{{ t("profile.newPassword") }}</span>
						<input v-model="createPassword" type="password" minlength="8" required :disabled="createLoading" />
					</label>
					<p v-if="createError" class="error">{{ createError }}</p>
					<div class="drawer-actions">
						<button type="button" :disabled="createLoading" @click="closeCreateDrawer">{{ t("common.backToList") }}</button>
						<button type="submit" :disabled="createLoading">{{ createLoading ? t("common.loading") : t("user.create") }}</button>
					</div>
				</form>
			</aside>
		</div>

		<div v-if="batchOpen" class="overlay" @click.self="closeBatchModal">
			<section class="modal">
				<div class="drawer-header">
					<h3>{{ t("page.userBatch") }}</h3>
					<button type="button" :disabled="batchLoading" @click="closeBatchModal">X</button>
				</div>
				<p>{{ t("batchUser.desc") }}</p>
				<p v-if="batchError" class="error">{{ batchError }}</p>
				<div class="toolbar-actions">
					<button type="button" :disabled="batchLoading" @click="addBatchRow">{{ t("batchUser.addRow") }}</button>
					<label class="upload-btn">
						<span>{{ t("batchUser.importExcel") }}</span>
						<input type="file" accept=".xlsx,.xls" @change="importExcel" />
					</label>
					<button type="button" :disabled="batchLoading" @click="submitBatch">{{ batchLoading ? t("common.loading") : t("batchUser.submit") }}</button>
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
						<tr v-for="item in batchRows" :key="item.id">
							<td><input v-model="item.account" type="text" :disabled="batchLoading" /></td>
							<td><input v-model="item.name" type="text" :disabled="batchLoading" /></td>
							<td><input v-model="item.email" type="email" :disabled="batchLoading" /></td>
							<td><input v-model="item.password" type="text" :disabled="batchLoading" /></td>
							<td><button type="button" :disabled="batchLoading || batchRows.length <= 1" @click="removeBatchRow(item.id)">{{ t("common.delete") }}</button></td>
						</tr>
					</tbody>
				</table>
				<section v-if="batchResults.length > 0" class="result-panel">
					<h4>{{ t("batchUser.result") }}</h4>
					<ul>
						<li v-for="item in batchResults" :key="`${item.account}-${item.message}`" :class="item.success ? 'ok' : 'fail'">
							{{ item.account }} - {{ item.message }}
						</li>
					</ul>
				</section>
			</section>
		</div>
	</section>
</template>

<style scoped>
.toolbar {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin: 8px 0 12px;
	gap: 8px;
}

.toolbar-actions {
	display: flex;
	gap: 8px;
	align-items: center;
}

.pager {
	display: flex;
	align-items: center;
	gap: 8px;
}

table {
	width: 100%;
	border-collapse: collapse;
}

th,
td {
	text-align: left;
	padding: 8px;
	border-bottom: 1px solid var(--color-border);
}

button {
	height: 32px;
	padding: 0 10px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface-soft);
	cursor: pointer;
}

.button-link {
	display: inline-flex;
	height: 32px;
	padding: 0 10px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface-soft);
	align-items: center;
	text-decoration: none;
	color: inherit;
}

.highlighted {
	background: rgba(46, 204, 113, 0.12);
}

.actions {
	display: flex;
	gap: 8px;
	align-items: center;
}

.overlay {
	position: fixed;
	inset: 0;
	background: rgba(17, 24, 39, 0.4);
	display: flex;
	justify-content: flex-end;
	align-items: stretch;
	z-index: 30;
}

.drawer {
	width: min(460px, 100%);
	height: 100%;
	background: var(--color-surface);
	padding: 16px;
	overflow-y: auto;
	box-shadow: -18px 0 36px -18px var(--color-shadow);
}

.modal {
	width: min(980px, 96%);
	max-height: 92%;
	margin: auto;
	background: var(--color-surface);
	border-radius: var(--radius-lg);
	padding: 16px;
	overflow: auto;
	box-shadow: 0 24px 48px -24px var(--color-shadow);
}

.drawer-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	gap: 8px;
	margin-bottom: 8px;
}

.drawer-header h3 {
	margin: 0;
}

.hint {
	color: var(--color-text-muted);
	margin: 0 0 12px;
}

.drawer-form {
	display: grid;
	gap: 10px;
}

.drawer-form label {
	display: grid;
	gap: 6px;
}

.drawer-actions {
	display: flex;
	gap: 8px;
	justify-content: flex-end;
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

.upload-btn input {
	position: absolute;
	top: 0;
	left: 0;
	opacity: 0;
	width: 100%;
	height: 100%;
	cursor: pointer;
}

.result-panel {
	margin-top: 12px;
	padding: 10px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
}

.result-panel h4 {
	margin: 0 0 8px;
}

.result-panel ul {
	margin: 0;
	padding-left: 18px;
}

.ok {
	color: var(--color-success);
}

.fail {
	color: var(--color-danger);
}

button:disabled {
	opacity: 0.6;
	cursor: default;
}

.error {
	color: var(--color-danger);
	margin: 4px 0;
}
</style>

