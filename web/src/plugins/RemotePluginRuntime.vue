<script setup lang="ts">
import { computed, onErrorCaptured, onMounted, onUnmounted, ref, watch } from "vue";
import { RefreshCw, ShieldAlert, TriangleAlert } from "lucide-vue-next";
import { useRouter } from "vue-router";

import { useI18n } from "../i18n";
import { canAccess } from "../permissions/access";
import { useThemeStore } from "../stores/theme";
import { useUserStore } from "../stores/user";
import { getToken } from "../utils/auth";
import { API_BASE_PREFIX } from "../utils/api-base-prefix";
import {
	handlePluginHostMessage,
	injectHostBridgeScript,
	normalizePluginLocales,
	pluginHostIdentity,
	pushLocaleToIframe,
	pushThemeToIframe,
	resolveActiveLocale
} from "./runtime-bridge";
import {
	loadPluginPageHTML,
	PLUGIN_PAGE_RENDER_TIMEOUT_MS,
	type PluginPageLoadError
} from "./runtime-loader";
import type { FrontendPluginManifest } from "./types";

const props = defineProps<{
	plugin: FrontendPluginManifest | null;
	title?: string;
}>();

const router = useRouter();
const { locale, t } = useI18n();
const themeStore = useThemeStore();
const userStore = useUserStore();
const iframeRef = ref<HTMLIFrameElement | null>(null);
const frameURL = ref("");
const loading = ref(false);
const errorCode = ref("");
const errorDetail = ref("");
let activeLoad: AbortController | null = null;
let loadGeneration = 0;
let renderTimeout = 0;

const pluginLocales = computed(() => normalizePluginLocales(props.plugin?.i18nLocales));
const activeLocale = computed(() => resolveActiveLocale(locale.value, pluginLocales.value));
const runtimeTitle = computed(() => {
	const explicit = props.title?.trim() ?? "";
	if (explicit !== "") {
		return explicit;
	}
	const plugin = props.plugin;
	if (!plugin) {
		return t("plugin.runtime.missing");
	}
	const localized = activeLocale.value === "zh-CN"
		? plugin.nameZhCN?.trim()
		: plugin.nameEnUS?.trim();
	return `${localized || plugin.name || plugin.id} (${plugin.id})`;
});
const allowed = computed(() => {
	const menu = props.plugin?.uiMenu;
	return canAccess(
		{ roles: menu?.requiredRoles ?? [], permissions: menu?.requiredPermissions ?? [] },
		userStore.profile?.role ?? "",
		userStore.permissions
	);
});
const broken = computed(() => errorCode.value !== "");
const errorTitle = computed(() => errorCode.value === "forbidden"
	? t("plugin.runtime.forbiddenTitle")
	: t("plugin.runtime.errorTitle"));
const errorIcon = computed(() => errorCode.value === "forbidden" ? ShieldAlert : TriangleAlert);

function revokeFrameURL(): void {
	if (frameURL.value !== "") {
		URL.revokeObjectURL(frameURL.value);
		frameURL.value = "";
	}
}

function clearRenderTimeout(): void {
	if (renderTimeout !== 0) {
		window.clearTimeout(renderTimeout);
		renderTimeout = 0;
	}
}

function fail(code: string, detail: string): void {
	activeLoad?.abort();
	activeLoad = null;
	clearRenderTimeout();
	revokeFrameURL();
	loading.value = false;
	errorCode.value = code;
	errorDetail.value = detail;
}

function patchPluginHTML(content: string, plugin: FrontendPluginManifest): string {
	const basePath = `${API_BASE_PREFIX}/v1/plugins/${plugin.id}/assets/`;
	const absoluteBasePath = `${window.location.origin}${basePath}`;
	const absoluteAssetsBase = `${absoluteBasePath}assets/`;
	let patched = injectHostBridgeScript(
		content,
		plugin,
		activeLocale.value,
		pluginLocales.value,
		getToken(),
		themeStore.pluginBridgeTheme,
		pluginHostIdentity(userStore.profile, userStore.permissions)
	);
	if (!/<base\s+/i.test(patched)) {
		patched = patched.replace(/<head>/i, `<head>\n<base href="${absoluteBasePath}">`);
	}
	return patched.replace(/(["'])\/assets\//g, `$1${absoluteAssetsBase}`);
}

async function load(): Promise<void> {
	const plugin = props.plugin;
	const generation = ++loadGeneration;
	activeLoad?.abort();
	activeLoad = new AbortController();
	clearRenderTimeout();
	revokeFrameURL();
	errorCode.value = "";
	errorDetail.value = "";

	if (!plugin) {
		fail("missing", t("plugin.runtime.missing"));
		return;
	}
	if (!allowed.value) {
		fail("forbidden", t("plugin.runtime.forbidden"));
		return;
	}

	loading.value = true;
	try {
		const url = `${API_BASE_PREFIX}/v1/plugins/${plugin.id}/page?locale=${encodeURIComponent(activeLocale.value)}`;
		const content = await loadPluginPageHTML(url, {
			signal: activeLoad.signal,
			token: getToken()
		});
		if (generation !== loadGeneration || activeLoad.signal.aborted) {
			return;
		}
		const blob = new Blob([patchPluginHTML(content, plugin)], { type: "text/html" });
		frameURL.value = URL.createObjectURL(blob);
		renderTimeout = window.setTimeout(() => {
			fail("render_timeout", t("plugin.runtime.renderTimeout"));
		}, PLUGIN_PAGE_RENDER_TIMEOUT_MS);
	} catch (error) {
		if (generation !== loadGeneration) {
			return;
		}
		const runtimeError = error as Partial<PluginPageLoadError>;
		if (runtimeError.code === "aborted") {
			return;
		}
		fail(runtimeError.code || "load_failed", error instanceof Error ? error.message : t("plugin.runtime.loadFailed"));
	}
}

function onFrameLoad(): void {
	const plugin = props.plugin;
	if (!plugin) {
		return;
	}
	clearRenderTimeout();
	loading.value = false;
	pushLocaleToIframe(iframeRef.value, plugin.id, activeLocale.value, pluginLocales.value);
	pushThemeToIframe(iframeRef.value, plugin.id, themeStore.pluginBridgeTheme);
}

function onHostMessage(event: MessageEvent): void {
	const plugin = props.plugin;
	if (!plugin) {
		return;
	}
	handlePluginHostMessage(event, {
		pluginId: plugin.id,
		source: iframeRef.value?.contentWindow ?? null,
		router,
		reload: () => void load(),
		fail: (message) => fail("plugin_error", message)
	});
}

onErrorCaptured((error) => {
	fail("host_render_error", error instanceof Error ? error.message : t("plugin.runtime.loadFailed"));
	return false;
});

watch(
	() => [props.plugin?.id ?? "", props.plugin?.version ?? "", activeLocale.value, allowed.value] as const,
	() => void load(),
	{ immediate: true }
);

watch(
	() => themeStore.pluginBridgeTheme,
	(theme) => {
		const plugin = props.plugin;
		if (plugin && allowed.value && !broken.value) {
			pushThemeToIframe(iframeRef.value, plugin.id, theme);
		}
	}
);

onMounted(() => window.addEventListener("message", onHostMessage));
onUnmounted(() => {
	window.removeEventListener("message", onHostMessage);
	loadGeneration += 1;
	activeLoad?.abort();
	clearRenderTimeout();
	revokeFrameURL();
});
</script>

<template>
	<section
		class="remote-plugin-fullpage"
		:data-loading="loading ? 'true' : 'false'"
		:aria-busy="loading"
		:aria-label="runtimeTitle"
	>
		<div v-if="loading && !frameURL" class="remote-plugin-state" role="status" data-testid="plugin-runtime-loading">
			<span class="remote-plugin-state__pulse" aria-hidden="true" />
			<p>{{ t("plugin.runtime.loading") }}</p>
		</div>
		<div v-else-if="broken" class="remote-plugin-state remote-plugin-state--error" role="alert" data-testid="plugin-runtime-error">
			<component :is="errorIcon" :size="28" aria-hidden="true" />
			<h2>{{ errorTitle }}</h2>
			<p>{{ errorDetail }}</p>
			<el-button v-if="errorCode !== 'forbidden'" type="primary" :icon="RefreshCw" @click="load">
				{{ t("common.retry") }}
			</el-button>
		</div>
		<iframe
			v-else-if="frameURL"
			:key="frameURL"
			ref="iframeRef"
			:title="runtimeTitle"
			:src="frameURL"
			class="plugin-page-frame"
			loading="lazy"
			sandbox="allow-downloads allow-forms allow-same-origin allow-scripts"
			@load="onFrameLoad"
			@error="fail('iframe_error', t('plugin.runtime.renderFailed'))"
		/>
	</section>
</template>
