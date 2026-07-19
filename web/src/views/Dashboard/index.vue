<script setup lang="ts">
import { computed } from "vue";
import { RouterLink } from "vue-router";

import PageShell from "../../components/Common/PageShell.vue";
import StateBlock from "../../components/Common/StateBlock.vue";
import { useI18n } from "../../i18n";
import { canAccess, type AccessDirectiveValue } from "../../permissions/access";
import { useNavigationStore } from "../../stores/navigation";
import { usePluginStore } from "../../stores/plugins";
import { useUserStore } from "../../stores/user";

type DashboardCard = {
	label: string;
	value: string | number;
	meta: string;
	tone: "neutral" | "success" | "warning" | "danger";
};

type QuickLink = {
	label: string;
	to: string;
	description: string;
	permission?: AccessDirectiveValue;
};

const pluginStore = usePluginStore();
const navigationStore = useNavigationStore();
const userStore = useUserStore();
const { t } = useI18n();

const disabledPluginCount = computed(() => pluginStore.items.length - pluginStore.enabledItems.length);
const systemPluginCount = computed(() => pluginStore.systemPlugins.length);
const appPluginCount = computed(() => pluginStore.appPlugins.length);
const menuSyncFailed = computed(() => navigationStore.syncStatus === "error");

const cards = computed<DashboardCard[]>(() => [
	{
		label: t("dashboard.card.loaded"),
		value: pluginStore.items.length,
		meta: t("dashboard.card.loadedMeta"),
		tone: pluginStore.hasSyncError ? "danger" : "neutral"
	},
	{
		label: t("dashboard.card.enabled"),
		value: pluginStore.enabledItems.length,
		meta: `${t("dashboard.card.disabled")}: ${disabledPluginCount.value}`,
		tone: disabledPluginCount.value > 0 ? "warning" : "success"
	},
	{
		label: t("dashboard.card.menus"),
		value: navigationStore.systemMenus.length,
		meta: navigationStore.customized ? t("dashboard.card.customMenus") : t("dashboard.card.defaultMenus"),
		tone: menuSyncFailed.value ? "danger" : "neutral"
	},
	{
		label: t("dashboard.card.auth"),
		value: userStore.isAuthenticated ? t("dashboard.auth.loggedIn") : t("dashboard.auth.guest"),
		meta: userStore.profile?.role || t("dashboard.auth.noRole"),
		tone: userStore.isAuthenticated ? "success" : "warning"
	}
]);

const quickLinks = computed<QuickLink[]>(() => [
	{
		label: t("page.users"),
		to: "/skoll/user",
		description: t("dashboard.quick.users"),
		permission: "user.read"
	},
	{
		label: t("page.permissions"),
		to: "/skoll/permission",
		description: t("dashboard.quick.permissions"),
		permission: "permission.manage"
	},
	{
		label: t("menu.plugins"),
		to: "/skoll/plugin",
		description: t("dashboard.quick.plugins"),
		permission: "plugin.read"
	},
	{
		label: t("page.audit"),
		to: "/skoll/audit",
		description: t("dashboard.quick.audit"),
		permission: "audit.read"
	}
].filter((item) => canAccess(item.permission, userStore.profile?.role ?? "", userStore.permissions)));

const riskItems = computed(() => {
	const items: Array<{ title: string; detail: string; tone: "warning" | "danger" | "neutral" }> = [];
	if (pluginStore.hasSyncError) {
		items.push({
			title: t("dashboard.risk.pluginSync"),
			detail: pluginStore.lastSyncError || t("plugin.sync.error"),
			tone: "danger"
		});
	}
	if (pluginStore.degradedMode) {
		items.push({
			title: t("dashboard.risk.degraded"),
			detail: t("plugin.sync.degraded"),
			tone: "warning"
		});
	}
	if (menuSyncFailed.value) {
		items.push({
			title: t("dashboard.risk.menuSync"),
			detail: navigationStore.lastError || t("dashboard.risk.menuSyncDetail"),
			tone: "danger"
		});
	}
	if (disabledPluginCount.value > 0) {
		items.push({
			title: t("dashboard.risk.disabledPlugins"),
			detail: `${disabledPluginCount.value}`,
			tone: "warning"
		});
	}
	if (items.length === 0) {
		items.push({
			title: t("dashboard.risk.clear"),
			detail: t("dashboard.risk.clearDetail"),
			tone: "neutral"
		});
	}
	return items;
});
</script>

<template>
	<PageShell :title="t('page.dashboard')" :description="t('dashboard.desc')">
		<template #meta>
			<div class="status-chips">
				<span class="chip">{{ t("dashboard.systemPlugins") }} {{ systemPluginCount }}</span>
				<span class="chip">{{ t("dashboard.appPlugins") }} {{ appPluginCount }}</span>
			</div>
		</template>

		<div class="cards">
			<article v-for="card in cards" :key="card.label" class="metric-card" :class="`metric-card--${card.tone}`">
				<p class="label">{{ card.label }}</p>
				<p class="value">{{ card.value }}</p>
				<p class="meta">{{ card.meta }}</p>
			</article>
		</div>

		<div class="dashboard-grid">
			<section class="panel">
				<div class="section-header">
					<div>
						<h3>{{ t("dashboard.quick.title") }}</h3>
						<p>{{ t("dashboard.quick.desc") }}</p>
					</div>
				</div>
				<div v-if="quickLinks.length > 0" class="quick-list">
					<RouterLink v-for="item in quickLinks" :key="item.to" :to="item.to" class="quick-link">
						<span>{{ item.label }}</span>
						<small>{{ item.description }}</small>
					</RouterLink>
				</div>
				<StateBlock v-else type="forbidden" :title="t('dashboard.quick.emptyTitle')" :description="t('dashboard.quick.emptyDesc')" />
			</section>

			<section class="panel">
				<div class="section-header">
					<div>
						<h3>{{ t("dashboard.risk.title") }}</h3>
						<p>{{ t("dashboard.risk.desc") }}</p>
					</div>
				</div>
				<div class="risk-list">
					<article v-for="item in riskItems" :key="item.title" class="risk-item" :class="`risk-item--${item.tone}`">
						<strong>{{ item.title }}</strong>
						<span>{{ item.detail }}</span>
					</article>
				</div>
			</section>
		</div>
	</PageShell>
</template>

<style scoped>
.section-header p {
	color: var(--color-text-muted);
	margin: 6px 0 0;
}

.status-chips {
	display: flex;
	align-items: center;
	justify-content: flex-end;
	gap: 8px;
	flex-wrap: wrap;
}

.chip {
	padding: 5px 9px;
	border-radius: 999px;
	background: var(--color-tag-bg);
	color: var(--color-tag-text);
	font-size: 0.78rem;
	font-weight: 700;
}

.cards {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
	gap: 12px;
}

.metric-card,
.panel {
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface-soft);
}

.metric-card {
	padding: 14px;
	border-left: 4px solid var(--color-border-strong);
}

.metric-card--success {
	border-left-color: var(--color-success);
}

.metric-card--warning {
	border-left-color: var(--color-warning-text);
}

.metric-card--danger {
	border-left-color: var(--color-danger);
}

.panel {
	display: grid;
	align-content: start;
	gap: 12px;
	padding: 16px;
	background: var(--color-surface);
}

.dashboard-grid {
	display: grid;
	grid-template-columns: minmax(0, 1fr) minmax(280px, 0.72fr);
	gap: 14px;
}

.section-header h3 {
	margin: 0;
	font-size: 1rem;
}

.label {
	margin: 0;
	font-size: 0.85rem;
	color: var(--color-text-muted);
}

.value {
	margin: 8px 0 0;
	font-size: 1.3rem;
	font-weight: 700;
}

.meta {
	margin: 6px 0 0;
	color: var(--color-text-muted);
	font-size: 0.82rem;
}

.quick-list {
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: 10px;
}

.quick-link {
	display: grid;
	gap: 4px;
	padding: 12px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface-soft);
	color: var(--color-text);
	text-decoration: none;
}

.quick-link:hover {
	border-color: var(--color-primary);
}

.quick-link span {
	font-weight: 700;
}

.quick-link small {
	color: var(--color-text-muted);
	line-height: 1.45;
}

.risk-list {
	display: grid;
	gap: 10px;
}

.risk-item {
	display: grid;
	gap: 4px;
	padding: 11px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface-soft);
}

.risk-item--warning {
	border-color: var(--color-warning-text);
	background: var(--color-warning-soft);
}

.risk-item--danger {
	border-color: var(--color-danger);
}

.risk-item span {
	color: var(--color-text-muted);
	font-size: 0.84rem;
	line-height: 1.45;
}

@media (max-width: 900px) {
	.dashboard-grid,
	.quick-list {
		display: grid;
	}

	.status-chips {
		justify-content: flex-start;
	}
}
</style>
