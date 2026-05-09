<script setup lang="ts">
import type { Locale } from "../../i18n";
import { useI18n } from "../../i18n";

defineProps<{
	title: string;
	subtitle: string;
	pluginCount: number;
	synced: boolean;
	error: string | null;
	locale: Locale;
	setLocale: (value: Locale) => void;
	onLogout: () => void | Promise<void>;
}>();

const emit = defineEmits<{
	(event: "toggle-sidebar"): void;
}>();

const { t } = useI18n();
</script>

<template>
	<header class="header">
		<div>
			<h1>{{ title }}</h1>
			<p>{{ subtitle }}</p>
		</div>
		<div class="status-box">
			<div class="chips">
				<span class="chip">{{ t("header.plugins") }} {{ pluginCount }}</span>
				<span class="chip" :class="{ warn: !synced || !!error }">{{ error ? t("header.syncError") : t("header.synced") }}</span>
			</div>
			<div class="locale-switch" role="group" :aria-label="t('header.language')">
				<span class="locale-label">{{ t("header.language") }}</span>
				<button
					type="button"
					class="locale-btn"
					:class="{ active: locale === 'zh-CN' }"
					@click="setLocale('zh-CN')"
				>
					{{ t("locale.zh-CN") }}
				</button>
				<button
					type="button"
					class="locale-btn"
					:class="{ active: locale === 'en-US' }"
					@click="setLocale('en-US')"
				>
					{{ t("locale.en-US") }}
				</button>
			</div>
			<button type="button" class="secondary" @click="onLogout">{{ t("header.logout") }}</button>
			<button type="button" @click="emit('toggle-sidebar')">{{ t("header.toggleMenu") }}</button>
		</div>
	</header>
</template>

<style scoped>
.header {
	padding: 14px 16px;
	border-radius: var(--radius-lg);
	background: linear-gradient(130deg, var(--color-surface) 0%, var(--color-surface-soft) 100%);
	border: 1px solid var(--color-border);
	display: flex;
	justify-content: space-between;
	align-items: center;
	gap: 12px;
	margin-bottom: 14px;
	box-shadow: 0 14px 30px -24px var(--color-shadow);
}

h1 {
	margin: 0;
	font-size: 1.2rem;
}

p {
	margin: 4px 0 0;
	font-size: 0.85rem;
	color: var(--color-text-muted);
}

.status-box {
	display: flex;
	align-items: center;
	gap: 10px;
}

.chips {
	display: flex;
	gap: 8px;
}

.chip {
	font-size: 0.75rem;
	padding: 4px 8px;
	border-radius: 999px;
	background: var(--color-tag-bg);
	color: var(--color-tag-text);
}

.chip.warn {
	background: var(--color-warning-soft);
	color: var(--color-warning-text);
}

.locale-switch {
	display: inline-flex;
	align-items: center;
	gap: 2px;
	border: 1px solid var(--color-border-strong);
	border-radius: 999px;
	overflow: hidden;
	padding-left: 8px;
}

.locale-label {
	font-size: 0.76rem;
	font-weight: 600;
	color: var(--color-text-muted);
	padding-right: 4px;
}

.locale-btn {
	background: transparent;
	color: var(--color-text-muted);
	border: none;
	border-radius: 0;
	padding: 6px 10px;
}

.locale-btn.active {
	background: var(--color-primary);
	color: var(--color-on-primary);
}

button {
	background: var(--color-primary);
	color: var(--color-on-primary);
	border: none;
	border-radius: var(--radius-md);
	padding: 7px 10px;
	cursor: pointer;
}

button.secondary {
	background: var(--color-surface);
	color: var(--color-text);
	border: 1px solid var(--color-border-strong);
}

button:hover {
	background: var(--color-primary-strong);
}

button.secondary:hover {
	background: var(--color-surface-soft);
}
</style>

