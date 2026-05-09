<script setup lang="ts">
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import HeaderBar from "./components/Layout/Header.vue";
import MainContent from "./components/Layout/MainContent.vue";
import Sidebar from "./components/Layout/Sidebar.vue";
import { useI18n } from "./i18n";
import { useAppStore } from "./stores/app";
import { usePluginStore } from "./stores/plugins";
import { useUserStore } from "./stores/user";

const appStore = useAppStore();
const pluginStore = usePluginStore();
const userStore = useUserStore();
const { locale, setLocale, t } = useI18n();
const route = useRoute();
const router = useRouter();

const pluginCount = computed(() => pluginStore.items.length);
const isLoginRoute = computed(() => route.path === "/login");
const sidebarItems = computed(() => [
	{ label: t("menu.dashboard"), to: "/dashboard" },
	{ label: t("menu.users"), to: "/user" },
	{ label: t("menu.roles"), to: "/role" },
	{ label: t("menu.permissions"), to: "/permission" },
	{ label: t("menu.plugins"), to: "/plugin" },
	{ label: t("menu.settings"), to: "/setting" }
]);

async function handleLogout(): Promise<void> {
	userStore.logout();
	await router.replace("/login");
}
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
				:plugin-count="pluginCount"
				:synced="pluginStore.syncedFromServer"
				:error="pluginStore.lastSyncError"
				:locale="locale"
				:set-locale="setLocale"
				:on-logout="handleLogout"
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

