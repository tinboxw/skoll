<script setup lang="ts">
import { ref } from "vue";
import { useRoute, useRouter } from "vue-router";

import { useI18n } from "../../i18n";
import { type ApiResponse, apiPost } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";

type UserRecord = {
	id: string;
	account: string;
	name?: string;
	email: string;
};

const router = useRouter();
const route = useRoute();
const { t } = useI18n();

const account = ref("");
const name = ref("");
const email = ref("");
const password = ref("");
const loading = ref(false);
const error = ref("");
const success = ref("");

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

function validateForm(): string {
	if (account.value.trim() === "") {
		return t("user.accountRequired");
	}
	if (email.value.trim() === "" || !email.value.includes("@")) {
		return t("user.emailInvalid");
	}
	if (password.value.trim().length < 8) {
		return t("user.passwordInvalid");
	}
	return "";
}

async function submit(): Promise<void> {
	error.value = "";
	success.value = "";
	const validationMessage = validateForm();
	if (validationMessage !== "") {
		error.value = validationMessage;
		return;
	}
	loading.value = true;
	try {
		const payload = await apiPost<ApiResponse<UserRecord>>("/v1/users", {
			account: account.value.trim(),
			name: name.value.trim() || account.value.trim(),
			email: email.value.trim(),
			passwordHash: password.value.trim()
		});
		const created = payload.data as unknown as Record<string, unknown> | undefined;
		const createdName = String(created?.account ?? created?.Account ?? account.value).trim();
		success.value = `${t("user.createDone")}: ${createdName}`;
		setTimeout(() => {
			void router.push(resolveReturnTo());
		}, 300);
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		loading.value = false;
	}
}
</script>

<template>
	<section>
		<h2>{{ t("page.userAdd") }}</h2>
		<section class="create-panel">
			<h3>{{ t("user.create") }}</h3>
			<p class="hint">{{ t("user.addHint") }}</p>
			<form class="create-form" @submit.prevent="submit">
				<input v-model="account" type="text" :placeholder="t('user.account')" required :disabled="loading" @blur="error = ''" />
				<input v-model="name" type="text" :placeholder="t('table.name')" :disabled="loading" />
				<input v-model="email" type="email" :placeholder="t('table.email')" required :disabled="loading" />
				<input v-model="password" type="password" minlength="8" :placeholder="t('profile.newPassword')" required :disabled="loading" />
				<button type="submit" :disabled="loading">{{ loading ? t("common.loading") : t("user.create") }}</button>
			</form>
			<div class="toolbar">
				<button type="button" :disabled="loading" @click="backToList">{{ t("common.backToList") }}</button>
			</div>
			<p v-if="error" class="error">{{ error }}</p>
			<p v-if="success" class="success">{{ success }}</p>
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

.create-form {
	display: grid;
	gap: 10px;
	grid-template-columns: repeat(5, minmax(0, 1fr));
}

input,
button {
	height: 32px;
	padding: 0 8px;
	border-radius: var(--radius-md);
	border: 1px solid var(--color-border);
	background: var(--color-surface);
}

button {
	background: var(--color-surface-soft);
	color: var(--color-text);
	cursor: pointer;
}

.hint {
	margin: 4px 0 12px;
	color: var(--color-text-muted);
}

.toolbar {
	margin-top: 10px;
}

.error {
	margin: 0;
	color: var(--color-danger);
}

.success {
	margin: 0;
	color: var(--color-success);
}
</style>

