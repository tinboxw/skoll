<script setup lang="ts">
import { onMounted, ref } from "vue";

import { PageShell } from "../../components/Common";
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
	<PageShell :title="t('page.profile')">
		<div class="profile-grid">
			<section class="panel">
				<h3>{{ t("profile.basicInfo") }}</h3>
				<el-form label-position="top" :disabled="savingProfile" @submit.prevent="saveProfile">
					<el-form-item :label="t('table.name')"><el-input v-model="name" autocomplete="name" /></el-form-item>
					<el-form-item :label="t('table.email')"><el-input v-model="email" type="email" autocomplete="email" /></el-form-item>
					<el-alert v-if="profileError" type="error" :title="profileError" show-icon :closable="false" />
					<el-alert v-if="profileSuccess" type="success" :title="profileSuccess" show-icon :closable="false" />
					<el-button native-type="submit" type="primary" :loading="savingProfile">{{ t("common.save") }}</el-button>
				</el-form>
			</section>

			<section class="panel">
				<h3>{{ t("profile.updatePassword") }}</h3>
				<el-form label-position="top" :disabled="savingPassword" @submit.prevent="savePassword">
					<el-form-item :label="t('profile.currentPassword')"><el-input v-model="currentPassword" type="password" show-password autocomplete="current-password" /></el-form-item>
					<el-form-item :label="t('profile.newPassword')"><el-input v-model="newPassword" type="password" show-password autocomplete="new-password" /></el-form-item>
					<el-form-item :label="t('profile.confirmPassword')"><el-input v-model="confirmPassword" type="password" show-password autocomplete="new-password" /></el-form-item>
					<el-alert v-if="passwordError" type="error" :title="passwordError" show-icon :closable="false" />
					<el-alert v-if="passwordSuccess" type="success" :title="passwordSuccess" show-icon :closable="false" />
					<el-button native-type="submit" type="primary" :loading="savingPassword">{{ t("profile.savePassword") }}</el-button>
				</el-form>
			</section>
		</div>
	</PageShell>
</template>

<style scoped>
.profile-grid {
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: var(--layout-gap);
}

.panel {
	padding: 12px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.panel h3 {
	margin: 0 0 12px;
}

.panel :deep(.el-alert) {
	margin-bottom: 12px;
}

@media (max-width: 760px) {
	.profile-grid {
		grid-template-columns: 1fr;
	}
}
</style>
