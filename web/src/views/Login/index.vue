<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute, useRouter } from "vue-router";

import { useI18n } from "../../i18n";
import { syncBackendPlugins, waitForPluginBootstrap } from "../../plugins";
import { clearDefaultHomePath, getDefaultHomePath, getSystemDefaultHomePath, resolveValidatedDefaultHomePath, usePluginStore } from "../../stores/plugins";
import { useUserStore } from "../../stores/user";
import { apiPost, type ApiResponse } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";

const route = useRoute();
const router = useRouter();
const userStore = useUserStore();
const pluginStore = usePluginStore();
const { t } = useI18n();

const account = ref("admin");
const password = ref("Admin@123456");
const loading = ref(false);
const error = ref("");

const redirectTo = computed(() => {
	const q = route.query.redirect;
	if (typeof q === "string" && q.trim() !== "") {
		return q;
	}
	return getDefaultHomePath(getSystemDefaultHomePath());
});

type LoginPayload = {
	token: string;
	permissions?: string[];
	user?: {
		id?: string;
		account?: string;
		name?: string;
		email?: string;
		role?: string;
		roles?: string[];
		organizationId?: string;
		organizationPath?: string[];
	};
};

async function resolvePostLoginTarget(path: string): Promise<string> {
	const raw = path.trim();
	if (raw === "") {
		return getSystemDefaultHomePath();
	}
	const target = raw.startsWith("/") ? raw : `/${raw}`;
	if (!target.startsWith("/skoll/plugins/") && !/^\/(?!skoll(?:\/|$))[^/]+\/?$/.test(target)) {
		return target;
	}
	await waitForPluginBootstrap();
	const fallback = getSystemDefaultHomePath();
	if (getDefaultHomePath(fallback) === target) {
		const validated = resolveValidatedDefaultHomePath(pluginStore.items, fallback);
		if (validated !== target) {
			clearDefaultHomePath();
			return validated;
		}
	}
	if (router.resolve(target).matched.length > 0) {
		return target;
	}
	return fallback;
}

async function login(): Promise<void> {
	error.value = "";
	loading.value = true;
	try {
		const payload = await apiPost<ApiResponse<LoginPayload>>("/v1/auth/login", {
			account: account.value,
			password: password.value
		});
		const token = payload.data?.token?.trim() ?? "";
		if (token === "") {
			throw new Error("missing token in login response");
		}
		const profileName = payload.data?.user?.name?.trim() || payload.data?.user?.account?.trim() || account.value.trim() || "admin";
		userStore.setSession(token, {
			id: payload.data?.user?.id?.trim() || profileName,
			name: profileName,
			role: payload.data?.user?.role?.trim() || "user",
			roles: Array.isArray(payload.data?.user?.roles) ? payload.data.user.roles : [payload.data?.user?.role?.trim() || "user"],
			organizationId: payload.data?.user?.organizationId?.trim() || "",
			organizationPath: Array.isArray(payload.data?.user?.organizationPath) ? payload.data.user.organizationPath : [],
			email: payload.data?.user?.email?.trim() || ""
		}, Array.isArray(payload.data?.permissions) ? payload.data.permissions : []);
		await userStore.hydrateProfile();
		await syncBackendPlugins(router, pluginStore);
		const target = await resolvePostLoginTarget(redirectTo.value);
		await router.replace(target);
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		loading.value = false;
	}
}
</script>

<template>
	<section class="login-page">
		<div class="card">
			<h1>{{ t("login.title") }}</h1>
			<p>{{ t("login.subtitle") }}</p>

			<el-form label-position="top" :disabled="loading" @submit.prevent="login">
				<el-form-item :label="t('login.account')">
					<el-input v-model="account" autocomplete="username" :placeholder="t('login.accountPlaceholder')" />
				</el-form-item>
				<el-form-item :label="t('login.password')">
					<el-input v-model="password" type="password" show-password autocomplete="current-password" :placeholder="t('login.passwordPlaceholder')" @keydown.enter="login" />
				</el-form-item>
				<el-alert v-if="error" type="error" :title="error" show-icon :closable="false" />
				<el-button native-type="submit" type="primary" :loading="loading" class="submit-button">
					{{ loading ? t("login.loading") : t("login.submit") }}
				</el-button>
			</el-form>
		</div>
	</section>
</template>

<style scoped>
.login-page {
	width: 100%;
	min-width: 0;
	min-height: calc(100vh - 48px);
	display: grid;
	place-items: center;
}

.card {
	width: 100%;
	max-width: 420px;
	box-sizing: border-box;
	padding: 20px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-lg);
	background: var(--color-surface);
	box-shadow: 0 20px 36px -30px var(--color-shadow);
	display: grid;
	gap: 12px;
}

h1 {
	margin: 0;
	font-size: 1.3rem;
}

p {
	margin: 0;
	color: var(--color-text-muted);
}

.submit-button {
	width: 100%;
}
</style>
