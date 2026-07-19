<script setup lang="ts">
import { computed, ref } from "vue";

import type { Locale } from "../../i18n";
import { useI18n } from "../../i18n";
import type { ThemeMode } from "../../stores/theme";

const props = defineProps<{
	title: string;
	subtitle: string;
	userName: string;
	userAvatarUrl: string;
	pluginCount: number;
	synced: boolean;
	error: string | null;
	locale: Locale;
	setLocale: (value: Locale) => void;
	theme: ThemeMode;
	setTheme: (value: ThemeMode) => void;
	onOpenProfile: () => void | Promise<void>;
	onLogout: () => void | Promise<void>;
}>();

const emit = defineEmits<{
	(event: "toggle-sidebar"): void;
}>();

const { t } = useI18n();
const menuOpen = ref(false);

const syncLabel = computed(() => {
	if (props.error) {
		return t("header.syncError");
	}
	if (props.synced) {
		return t("header.synced");
	}
	return t("header.syncing");
});

const syncDetail = computed(() => {
	if (props.error) {
		const text = props.error.trim();
		if (!text) {
			return "";
		}
		return text.length > 44 ? `${text.slice(0, 44)}...` : text;
	}
	if (!props.synced) {
		return t("header.syncingHint");
	}
	return "";
});

const avatarFallback = computed(() => {
	const source = props.userName.trim();
	if (source === "") {
		return "U";
	}
	return source.slice(0, 1).toUpperCase();
});

function toggleMenu(): void {
	menuOpen.value = !menuOpen.value;
}

async function openProfile(): Promise<void> {
	menuOpen.value = false;
	await props.onOpenProfile();
}

async function logout(): Promise<void> {
	menuOpen.value = false;
	await props.onLogout();
}
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
				<span class="chip" :class="{ warn: !synced || !!error }" :title="error ?? undefined">{{ syncLabel }}</span>
				<span v-if="syncDetail" class="chip detail" :class="{ warn: !!error }" :title="error ?? undefined">{{ syncDetail }}</span>
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
			<div class="theme-switch" role="group" :aria-label="t('header.theme')">
				<button
					type="button"
					class="theme-btn"
					:title="t('header.themeLight')"
					:class="{ active: theme === 'light' }"
					@click="setTheme('light')"
				>
					{{ t("header.themeLightShort") }}
				</button>
				<button
					type="button"
					class="theme-btn"
					:title="t('header.themeDark')"
					:class="{ active: theme === 'dark' }"
					@click="setTheme('dark')"
				>
					{{ t("header.themeDarkShort") }}
				</button>
				<button
					type="button"
					class="theme-btn"
					:title="t('header.themeCompact')"
					:class="{ active: theme === 'compact' }"
					@click="setTheme('compact')"
				>
					{{ t("header.themeCompactShort") }}
				</button>
			</div>
			<div class="user-menu">
				<button type="button" class="avatar-btn" @click="toggleMenu">
					<img v-if="userAvatarUrl" :src="userAvatarUrl" :alt="userName" class="avatar-img" />
					<span v-else class="avatar-text">{{ avatarFallback }}</span>
				</button>
				<div v-if="menuOpen" class="menu-panel">
					<p class="menu-user">{{ userName }}</p>
					<button type="button" class="menu-item" @click="openProfile">{{ t("header.profile") }}</button>
					<button type="button" class="menu-item danger" @click="logout">{{ t("header.logout") }}</button>
				</div>
			</div>
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
	gap: var(--layout-gap);
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
	flex-wrap: wrap;
	justify-content: flex-end;
}

.user-menu {
	position: relative;
}

.avatar-btn {
	width: 36px;
	height: 36px;
	padding: 0;
	border-radius: 999px;
	border: 1px solid var(--color-border-strong);
	background: var(--color-surface);
	overflow: hidden;
	display: inline-flex;
	align-items: center;
	justify-content: center;
}

.avatar-img {
	width: 100%;
	height: 100%;
	object-fit: cover;
}

.avatar-text {
	font-size: 0.8rem;
	font-weight: 700;
	color: var(--color-text);
}

.menu-panel {
	position: absolute;
	top: 40px;
	right: 0;
	min-width: 170px;
	background: var(--color-surface);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	box-shadow: 0 12px 24px -16px var(--color-shadow);
	padding: 8px;
	display: grid;
	gap: 6px;
	z-index: 10;
}

.menu-user {
	margin: 0;
	padding: 4px 6px;
	font-size: 0.78rem;
	color: var(--color-text-muted);
}

.menu-item {
	background: var(--color-surface-soft);
	border: 1px solid var(--color-border);
	color: var(--color-text);
	padding: 7px 8px;
	border-radius: var(--radius-sm);
	text-align: left;
	font-size: 0.82rem;
}

.menu-item.danger {
	color: var(--color-danger);
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

.chip.detail {
	max-width: 280px;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
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

.theme-switch {
	display: inline-flex;
	align-items: center;
	border: 1px solid var(--color-border-strong);
	border-radius: 999px;
	overflow: hidden;
	background: var(--color-surface);
}

.theme-btn {
	min-width: 34px;
	background: transparent;
	color: var(--color-text-muted);
	border: none;
	border-radius: 0;
	padding: 6px 9px;
	font-size: 0.76rem;
	font-weight: 700;
}

.theme-btn.active {
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

@media (max-width: 860px) {
	.header {
		align-items: stretch;
		flex-direction: column;
	}

	.status-box {
		justify-content: flex-start;
	}

	.chips {
		flex-wrap: wrap;
	}
}
</style>

