<script setup lang="ts">
import { onMounted, ref } from "vue";

import { useI18n } from "../../i18n";
import { useUserStore } from "../../stores/user";
import { apiPatch, apiPut, type ApiResponse } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";

const { t } = useI18n();
const userStore = useUserStore();

const name = ref("");
const email = ref("");
const currentPassword = ref("");
const newPassword = ref("");
const confirmPassword = ref("");
const savingProfile = ref(false);
const savingPassword = ref(false);
const profileError = ref("");
const profileSuccess = ref("");
const passwordError = ref("");
const passwordSuccess = ref("");

function loadProfile(): void {
	name.value = userStore.profile?.name ?? "";
	email.value = userStore.profile?.email ?? "";
}

function validateProfile(): string {
	if (name.value.trim() === "") {
		return t("profile.nameRequired");
	}
	if (email.value.trim() === "") {
		return t("profile.emailRequired");
	}
	if (!email.value.includes("@")) {
		return t("profile.emailInvalid");
	}
	return "";
}

function validatePassword(): string {
	if (currentPassword.value.trim() === "") {
		return t("profile.currentPasswordRequired");
	}
	if (newPassword.value.trim().length < 8) {
		return t("profile.newPasswordInvalid");
	}
	if (newPassword.value !== confirmPassword.value) {
		return t("profile.passwordNotMatch");
	}
	return "";
}

async function saveProfile(): Promise<void> {
	profileError.value = "";
	profileSuccess.value = "";
	const message = validateProfile();
	if (message !== "") {
		profileError.value = message;
		return;
	}

	savingProfile.value = true;
	try {
		await apiPut<ApiResponse<unknown>>("/v1/auth/me/profile", {
			name: name.value.trim(),
			email: email.value.trim()
		});
		await userStore.hydrateProfile();
		profileSuccess.value = t("profile.saveSuccess");
	} catch (e) {
		profileError.value = toErrorMessage(e);
	} finally {
		savingProfile.value = false;
	}
}

async function savePassword(): Promise<void> {
	passwordError.value = "";
	passwordSuccess.value = "";
	const message = validatePassword();
	if (message !== "") {
		passwordError.value = message;
		return;
	}

	savingPassword.value = true;
	try {
		await apiPatch<ApiResponse<unknown>>("/v1/auth/me/password", {
			currentPassword: currentPassword.value,
			newPassword: newPassword.value,
			confirmPassword: confirmPassword.value
		});
		passwordSuccess.value = t("profile.passwordSaveSuccess");
		currentPassword.value = "";
		newPassword.value = "";
		confirmPassword.value = "";
	} catch (e) {
		passwordError.value = toErrorMessage(e);
	} finally {
		savingPassword.value = false;
	}
}

onMounted(() => {
	loadProfile();
});
</script>

<template>
	<section>
		<h2>{{ t("page.profile") }}</h2>
		<div class="panel">
			<h3>{{ t("profile.basicInfo") }}</h3>
			<label>
				<span>{{ t("table.name") }}</span>
				<input v-model="name" type="text" :disabled="savingProfile" />
			</label>
			<label>
				<span>{{ t("table.email") }}</span>
				<input v-model="email" type="email" :disabled="savingProfile" />
			</label>
			<p v-if="profileError" class="error">{{ profileError }}</p>
			<p v-if="profileSuccess" class="success">{{ profileSuccess }}</p>
			<button type="button" :disabled="savingProfile" @click="saveProfile">{{ savingProfile ? t("common.loading") : t("common.save") }}</button>
		</div>

		<div class="panel">
			<h3>{{ t("profile.updatePassword") }}</h3>
			<label>
				<span>{{ t("profile.currentPassword") }}</span>
				<input v-model="currentPassword" type="password" :disabled="savingPassword" />
			</label>
			<label>
				<span>{{ t("profile.newPassword") }}</span>
				<input v-model="newPassword" type="password" :disabled="savingPassword" />
			</label>
			<label>
				<span>{{ t("profile.confirmPassword") }}</span>
				<input v-model="confirmPassword" type="password" :disabled="savingPassword" />
			</label>
			<p v-if="passwordError" class="error">{{ passwordError }}</p>
			<p v-if="passwordSuccess" class="success">{{ passwordSuccess }}</p>
			<button type="button" :disabled="savingPassword" @click="savePassword">{{ savingPassword ? t("common.loading") : t("profile.savePassword") }}</button>
		</div>
	</section>
</template>

<style scoped>
.panel {
	margin-bottom: 12px;
	padding: 12px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	display: grid;
	gap: 8px;
	max-width: 520px;
}

label {
	display: grid;
	gap: 6px;
}

input,
button {
	height: 34px;
	padding: 0 10px;
	border-radius: var(--radius-md);
	border: 1px solid var(--color-border);
}

button {
	background: var(--color-surface-soft);
	cursor: pointer;
	width: fit-content;
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
