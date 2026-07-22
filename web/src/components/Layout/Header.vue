<script setup lang="ts">
import { computed } from "vue";
import { LogOut, Menu, UserRound } from "lucide-vue-next";

import type { Locale } from "../../i18n";
import { useI18n } from "../../i18n";
import type { ThemeColorScheme, ThemeDensity } from "../../stores/theme";

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
	colorScheme: ThemeColorScheme;
	setColorScheme: (value: ThemeColorScheme) => void;
	density: ThemeDensity;
	setDensity: (value: ThemeDensity) => void;
	onOpenProfile: () => void | Promise<void>;
	onLogout: () => void | Promise<void>;
}>();

const emit = defineEmits<{
	(event: "toggle-sidebar"): void;
}>();

const { t } = useI18n();

const syncLabel = computed(() => {
	if (props.error) return t("header.syncError");
	return props.synced ? t("header.synced") : t("header.syncing");
});

const syncDetail = computed(() => {
	if (props.error) {
		const text = props.error.trim();
		return text.length > 44 ? `${text.slice(0, 44)}...` : text;
	}
	return props.synced ? "" : t("header.syncingHint");
});

const avatarFallback = computed(() => props.userName.trim().slice(0, 1).toUpperCase() || "U");
const localeModel = computed<Locale>({ get: () => props.locale, set: props.setLocale });
const colorSchemeModel = computed<ThemeColorScheme>({ get: () => props.colorScheme, set: props.setColorScheme });
const densityModel = computed<ThemeDensity>({ get: () => props.density, set: props.setDensity });
const localeOptions = computed(() => [
	{ label: t("locale.zh-CN"), value: "zh-CN" },
	{ label: t("locale.en-US"), value: "en-US" }
]);
const themeOptions = computed(() => [
	{ label: t("header.themeLightShort"), value: "light" },
	{ label: t("header.themeDarkShort"), value: "dark" }
]);
const densityOptions = computed(() => [
	{ label: t("header.densityComfortableShort"), value: "comfortable" },
	{ label: t("header.densityCompactShort"), value: "compact" }
]);

async function handleUserCommand(command: "profile" | "logout"): Promise<void> {
	if (command === "profile") {
		await props.onOpenProfile();
		return;
	}
	await props.onLogout();
}
</script>

<template>
	<header class="header">
		<div class="header__identity">
			<h1>{{ title }}</h1>
			<p>{{ subtitle }}</p>
		</div>
		<div class="header__commands">
			<div class="header__status" :aria-label="syncLabel">
				<el-tag effect="plain">{{ t("header.plugins") }} {{ pluginCount }}</el-tag>
				<el-tooltip :content="error || syncDetail" :disabled="!error && !syncDetail">
					<el-tag :type="error || !synced ? 'warning' : 'success'" effect="plain">{{ syncLabel }}</el-tag>
				</el-tooltip>
			</div>

			<el-tooltip :content="t('header.language')">
				<el-segmented v-model="localeModel" size="small" :options="localeOptions" :aria-label="t('header.language')" />
			</el-tooltip>
			<el-tooltip :content="t('header.theme')">
				<el-segmented v-model="colorSchemeModel" size="small" :options="themeOptions" :aria-label="t('header.theme')" />
			</el-tooltip>
			<el-tooltip :content="t('header.density')">
				<el-segmented v-model="densityModel" size="small" :options="densityOptions" :aria-label="t('header.density')" />
			</el-tooltip>

			<el-dropdown trigger="click" @command="handleUserCommand">
				<el-button text circle :aria-label="userName">
					<el-avatar :size="32" :src="userAvatarUrl || undefined">{{ avatarFallback }}</el-avatar>
				</el-button>
				<template #dropdown>
					<el-dropdown-menu>
						<el-dropdown-item disabled>{{ userName }}</el-dropdown-item>
						<el-dropdown-item command="profile" :icon="UserRound">{{ t("header.profile") }}</el-dropdown-item>
						<el-dropdown-item command="logout" :icon="LogOut" divided>{{ t("header.logout") }}</el-dropdown-item>
					</el-dropdown-menu>
				</template>
			</el-dropdown>

			<el-tooltip :content="t('header.toggleMenu')">
				<el-button :icon="Menu" circle :aria-label="t('header.toggleMenu')" @click="emit('toggle-sidebar')" />
			</el-tooltip>
		</div>
	</header>
</template>

<style scoped>
.header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: var(--layout-gap);
	margin-bottom: 14px;
	padding: 12px 14px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
	box-shadow: 0 10px 24px -22px var(--color-shadow);
}

.header__identity {
	min-width: 0;
}

.header__identity h1 {
	margin: 0;
	font-size: 1.15rem;
}

.header__identity p {
	margin: 3px 0 0;
	color: var(--color-text-muted);
	font-size: 0.82rem;
}

.header__commands,
.header__status {
	display: flex;
	align-items: center;
	justify-content: flex-end;
	gap: 8px;
	min-width: 0;
	flex-wrap: wrap;
}

@media (max-width: 860px) {
	.header {
		align-items: stretch;
		flex-direction: column;
	}

	.header__commands,
	.header__status {
		justify-content: flex-start;
	}
}
</style>
