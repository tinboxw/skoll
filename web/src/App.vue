<script setup lang="ts">
import { computed, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import HeaderBar from "./components/Layout/Header.vue";
import MainContent from "./components/Layout/MainContent.vue";
import PinnedTabs from "./components/Layout/PinnedTabs.vue";
import Sidebar from "./components/Layout/Sidebar.vue";
import { useI18n } from "./i18n";
import { buildSidebarItems } from "./navigation/menu";
import { useAppStore } from "./stores/app";
import { useNavigationStore } from "./stores/navigation";
import { usePluginStore } from "./stores/plugins";
import { useTabsStore } from "./stores/tabs";
import { useThemeStore } from "./stores/theme";
import { useUserStore } from "./stores/user";
import { apiPost } from "./utils/api";

const appStore = useAppStore();
const navigationStore = useNavigationStore();
const pluginStore = usePluginStore();
const tabsStore = useTabsStore();
const themeStore = useThemeStore();
const userStore = useUserStore();
const { locale, setLocale, t } = useI18n();
const route = useRoute();
const router = useRouter();
const ADMIN_BASE = "/skoll";

const pluginCount = computed(() => pluginStore.items.length);
const isLoginRoute = computed(() => route.path === `${ADMIN_BASE}/login`);
const isPluginHostRoute = computed(() => {
	const name = String(route.name || "");
	const path = String(route.path || "");
	return name.startsWith("plugin-") || name.startsWith("app-home-") || path.startsWith(`${ADMIN_BASE}/plugins/`);
});

function resolvePluginLabel(item: { id: string; name?: string; nameZhCN?: string; nameEnUS?: string }): string {
	const currentLocale = String(locale.value || "zh-CN").trim();
	const zh = String(item.nameZhCN || "").trim();
	const en = String(item.nameEnUS || "").trim();
	if (currentLocale === "zh-CN" && zh) {
		return zh;
	}
	if (currentLocale === "en-US" && en) {
		return en;
	}
	return String(item.name || "").trim() || item.id;
}

async function openPinnedTab(path: string): Promise<void> {
	await router.push(path);
}

async function closePinnedTab(path: string): Promise<void> {
	const active = route.path === path || route.path.startsWith(path + "/");
	tabsStore.unpinByPath(path);
	if (active) {
		await router.push(`${ADMIN_BASE}/dashboard`);
	}
}

async function closeRecentTab(path: string): Promise<void> {
	const active = route.path === path || route.path.startsWith(path + "/");
	tabsStore.removeRecentByPath(path);
	if (active) {
		await router.push(`${ADMIN_BASE}/dashboard`);
	}
}

function pinRecentTab(path: string): void {
	const item = tabsStore.recentItems.find((tab) => tab.path === path);
	if (!item) {
		return;
	}
	tabsStore.pinPluginTab(item);
}

function resolveVisitedPluginTab(path: string): { id: string; label: string; path: string } | null {
	const normalized = path.trim();
	if (normalized === "") {
		return null;
	}

	const candidates = pluginStore.items
		.filter((item) => {
			if (item.uiMode === "backend_only") {
				return false;
			}
			if (item.uiNavPosition !== "top_tab") {
				return false;
			}
			if (item.uiTabMode === "disabled") {
				return false;
			}
			return typeof item.entryPath === "string" && item.entryPath.trim() !== "";
		})
		.map((item) => ({
			id: item.id,
			label: resolvePluginLabel(item),
			path: item.entryPath as string
		}))
		.sort((a, b) => b.path.length - a.path.length);

	for (const item of candidates) {
		if (normalized === item.path || normalized.startsWith(item.path + "/")) {
			return item;
		}
	}

	const appHomeMatch = normalized.match(/^\/(?!skoll(?:\/|$))([^/]+)\/?$/);
	if (appHomeMatch) {
		const appID = appHomeMatch[1] || "";
		if (!appID) {
			return null;
		}
		const appPlugin = pluginStore.items.find((item) => item.level === "app" && item.appId === appID && item.enabled !== false);
		return {
			id: appPlugin?.id || `app:${appID}`,
			label: appPlugin ? resolvePluginLabel(appPlugin) : appID,
			path: `/${appID}`
		};
	}

	return null;
}

const topTabPluginIDs = computed(() => {
	return pluginStore.items
		.filter((item) => item.enabled !== false && item.uiMode !== "backend_only" && item.uiNavPosition === "top_tab" && item.uiTabMode !== "disabled")
		.map((item) => item.id);
});

watch(
	topTabPluginIDs,
	(pluginIDs) => {
		tabsStore.prunePluginTabs(pluginIDs);
	},
	{ immediate: true }
);

watch(
	() => route.path,
	(path) => {
		const tab = resolveVisitedPluginTab(path);
		if (!tab) {
			return;
		}
		const pluginItem = pluginStore.items.find((item) => item.id === tab.id || item.entryPath === tab.path);
		if (pluginItem?.uiTabMode === "fixed") {
			tabsStore.pinPluginTab(tab);
			return;
		}
		tabsStore.touchRecentTab(tab);
	},
	{ immediate: true }
);

const sidebarItems = computed(() => buildSidebarItems({
	t,
	systemMenus: navigationStore.systemMenus,
	plugins: pluginStore.items,
	currentRole: userStore.profile?.role ?? "",
	currentLocale: String(locale.value || "zh-CN"),
	permissions: userStore.permissions,
	resolvePluginLabel
}));

async function hydrateAuthenticatedShell(): Promise<void> {
	await userStore.hydrateProfile();
	await navigationStore.loadSystemMenus();
}

async function handleLogout(): Promise<void> {
	try {
		await apiPost("/v1/auth/logout");
	} catch {
		// Ignore API errors and clear local session regardless.
	}
	userStore.logout();
	await router.replace(`${ADMIN_BASE}/login`);
}

async function handleOpenProfile(): Promise<void> {
	await router.push(`${ADMIN_BASE}/profile`);
}

watch(
	() => userStore.isAuthenticated,
	(isAuthenticated) => {
		if (isAuthenticated) {
			void hydrateAuthenticatedShell();
			return;
		}
		navigationStore.$patch({
			systemMenus: [],
			customized: false,
			syncStatus: "idle",
			lastError: null
		});
	},
	{ immediate: true }
);
</script>

<template>
	<main v-if="isLoginRoute" class="login-shell">
		<RouterView />
	</main>
	<main v-else class="app-shell">
		<Sidebar :collapsed="appStore.sidebarCollapsed" :items="sidebarItems" />
		<div class="content-area" :class="{ 'plugin-content-area': isPluginHostRoute }">
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
				:theme="themeStore.mode"
				:set-theme="themeStore.setTheme"
				:on-logout="handleLogout"
				:on-open-profile="handleOpenProfile"
				@toggle-sidebar="appStore.toggleSidebar"
			/>
			<PinnedTabs
				:pinned-items="tabsStore.pinnedItems"
				:recent-items="tabsStore.recentItems"
				@open="openPinnedTab"
				@unpin="closePinnedTab"
				@remove-recent="closeRecentTab"
				@pin="pinRecentTab"
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
	padding: var(--content-padding);
	box-sizing: border-box;
	min-width: 0;
}

.plugin-content-area {
	height: 100vh;
	box-sizing: border-box;
	overflow: hidden;
	display: grid;
	grid-template-rows: auto auto minmax(0, 1fr);
	gap: var(--layout-gap);
}

@media (max-width: 860px) {
	.app-shell {
		grid-template-columns: 1fr;
	}
}
</style>

