<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import HeaderBar from "./components/Layout/Header.vue";
import MainContent from "./components/Layout/MainContent.vue";
import Sidebar from "./components/Layout/Sidebar.vue";
import { useI18n } from "./i18n";
import { useAppStore } from "./stores/app";
import { usePluginStore } from "./stores/plugins";
import { useUserStore } from "./stores/user";
import { apiPost } from "./utils/api";

const appStore = useAppStore();
const pluginStore = usePluginStore();
const userStore = useUserStore();
const { locale, setLocale, t } = useI18n();
const route = useRoute();
const router = useRouter();

const pluginCount = computed(() => pluginStore.items.length);
const isLoginRoute = computed(() => route.path === "/login");

type SidebarItem = {
	label: string;
	to: string;
	requiredRoles?: string[];
	requiredPermissions?: string[];
};

const sidebarItems = computed(() => {
	const allItems: SidebarItem[] = [
		{ label: t("menu.dashboard"), to: "/dashboard" },
		{ label: t("menu.users"), to: "/user" },
		{ label: t("menu.roles"), to: "/role" },
		{ label: t("menu.permissions"), to: "/permission", requiredPermissions: ["permission.manage"] },
		{ label: t("menu.plugins"), to: "/plugin" },
		{ label: t("menu.settings"), to: "/setting", requiredPermissions: ["role.manage"] }
	];

	const currentRole = userStore.profile?.role ?? "";
	const permissionSet = new Set(userStore.permissions);

	return allItems.filter((item) => {
		if (currentRole === "super_admin") {
			return true;
		}
		if (Array.isArray(item.requiredRoles) && item.requiredRoles.length > 0 && !item.requiredRoles.includes(currentRole)) {
			return false;
		}
		if (Array.isArray(item.requiredPermissions) && item.requiredPermissions.length > 0) {
			return item.requiredPermissions.every((permission) => permissionSet.has(permission));
		}
		return true;
	});
});

async function handleLogout(): Promise<void> {
	try {
		await apiPost("/v1/auth/logout");
	} catch {
		// Ignore API errors and clear local session regardless.
	}
	userStore.logout();
	await router.replace("/login");
}

async function handleOpenProfile(): Promise<void> {
	await router.push("/profile");
}

onMounted(() => {
	if (userStore.isAuthenticated) {
		void userStore.hydrateProfile();
	}
});
</script>

<template>
	<main v-if="isLoginRoute" class="login-shell">
		<RouterView />
	</main>
	<main v-else class="app-shell">
		<Sidebar :collapsed="appStore.sidebarCollapsed" :items="sidebarItems" />
		<div class="content-area">
			<HeaderBar
				:title="t('app.title')"
				:subtitle="t('app.subtitle')"
				:user-name="userStore.profile?.name || 'User'"
				:user-avatar-url="userStore.profile?.avatarUrl || ''"
				:plugin-count="pluginCount"
				:synced="pluginStore.syncedFromServer"
				:error="pluginStore.lastSyncError"
				:locale="locale"
				:set-locale="setLocale"
				:on-logout="handleLogout"
				:on-open-profile="handleOpenProfile"
				@toggle-sidebar="appStore.toggleSidebar"
			/>
			<MainContent>
				<RouterView />
			</MainContent>
		</div>
	</main>
</template>

<style scoped>
.app-shell {
	min-height: 100vh;
	display: grid;
	grid-template-columns: 240px 1fr;
	background: radial-gradient(circle at 20% 10%, var(--color-bg-accent) 0%, var(--color-bg) 55%);
}

.login-shell {
	min-height: 100vh;
	padding: 24px;
	background: radial-gradient(circle at 20% 10%, var(--color-bg-accent) 0%, var(--color-bg) 55%);
}

.content-area {
	padding: 18px;
}

@media (max-width: 860px) {
	.app-shell {
		grid-template-columns: 1fr;
	}
}
</style>

