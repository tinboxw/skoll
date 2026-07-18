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

			<label>
				<span>{{ t("login.account") }}</span>
				<input v-model="account" type="text" :placeholder="t('login.accountPlaceholder')" :disabled="loading" />
			</label>

			<label>
				<span>{{ t("login.password") }}</span>
				<input v-model="password" type="password" :placeholder="t('login.passwordPlaceholder')" :disabled="loading" @keydown.enter="login" />
			</label>

			<p v-if="error" class="error">{{ error }}</p>

			<button type="button" :disabled="loading" @click="login">{{ loading ? t("login.loading") : t("login.submit") }}</button>
		</div>
	</section>
</template>

<style scoped>
.login-page {
	min-height: calc(100vh - 48px);
	display: grid;
	place-items: center;
}

.card {
	width: min(420px, 92vw);
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

label {
	display: grid;
	gap: 6px;
}

input {
	height: 38px;
	padding: 0 10px;
	border-radius: var(--radius-md);
	border: 1px solid var(--color-border);
	background: var(--color-surface-soft);
}

button {
	height: 38px;
	border: none;
	border-radius: var(--radius-md);
	background: var(--color-primary);
	color: var(--color-on-primary);
	cursor: pointer;
}

button:disabled {
	opacity: 0.65;
	cursor: default;
}

.error {
	color: var(--color-danger);
}
</style>
