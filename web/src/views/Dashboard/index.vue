<script setup lang="ts">
import { computed } from "vue";

import { useI18n } from "../../i18n";
import { usePluginStore } from "../../stores/plugins";
import { useUserStore } from "../../stores/user";

const pluginStore = usePluginStore();
const userStore = useUserStore();
const { t } = useI18n();

const cards = computed(() => [
	{ label: t("dashboard.card.loaded"), value: pluginStore.items.length },
	{ label: t("dashboard.card.enabled"), value: pluginStore.enabledItems.length },
	{ label: t("dashboard.card.auth"), value: userStore.isAuthenticated ? t("dashboard.auth.loggedIn") : t("dashboard.auth.guest") }
]);
</script>

<template>
	<section>
		<h2>{{ t("page.dashboard") }}</h2>
		<p class="desc">{{ t("dashboard.desc") }}</p>
		<div class="cards">
			<article v-for="card in cards" :key="card.label" class="card">
				<p class="label">{{ card.label }}</p>
				<p class="value">{{ card.value }}</p>
			</article>
		</div>
	</section>
</template>

<style scoped>
.desc {
	color: var(--color-text-muted);
	margin-bottom: 14px;
}

.cards {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
	gap: 12px;
}

.card {
	padding: 12px;
	border-radius: var(--radius-md);
	border: 1px solid var(--color-border);
	background: var(--color-surface-soft);
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
</style>

