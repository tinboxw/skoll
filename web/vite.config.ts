import { defineConfig, loadEnv } from "vite";
import vue from "@vitejs/plugin-vue";
import AutoImport from "unplugin-auto-import/vite";
import Components from "unplugin-vue-components/vite";
import { ElementPlusResolver } from "unplugin-vue-components/resolvers";

function normalizeAPIPrefix(raw: string): string {
	const trimmed = raw.trim();
	if (trimmed === "") {
		return "/skoll";
	}
	const withSlash = trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
	const normalized = withSlash.replace(/\/+$/, "");
	return normalized === "" ? "/skoll" : normalized;
}

function normalizeWebBasePath(raw: string): string {
	const trimmed = raw.trim();
	if (trimmed === "") {
		return "/skoll";
	}
	if (trimmed === "/") {
		return "/";
	}
	const withSlash = trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
	const normalized = withSlash.replace(/\/+$/, "");
	return normalized === "" ? "/" : normalized;
}

export default defineConfig(({ mode, command }) => {
	const env = loadEnv(mode, process.cwd(), "SKOLL_");
	const proxyTarget = (env.SKOLL_API_PROXY_TARGET || "http://127.0.0.1:8080").trim();
	const proxyTimeout = Number.parseInt(env.SKOLL_API_PROXY_TIMEOUT_MS || "10000", 10);
	const apiBasePrefix = normalizeAPIPrefix(env.SKOLL_API_BASE_PREFIX || "/skoll");
	const webBasePath = normalizeWebBasePath(env.SKOLL_WEB_BASE_PATH || "/skoll");
	const viteBase = webBasePath === "/" ? "/" : `${webBasePath}/`;
	const effectiveBase = command === "build" ? viteBase : "/";

	return {
		base: effectiveBase,
		define: {
			__SKOLL_API_BASE_PREFIX__: JSON.stringify(apiBasePrefix),
			__SKOLL_WEB_BASE_PATH__: JSON.stringify(webBasePath)
		},
		plugins: [
			vue(),
			AutoImport({
				dts: false,
				resolvers: [ElementPlusResolver()]
			}),
			Components({
				dts: false,
				resolvers: [ElementPlusResolver()]
			})
		],
		server: {
			host: "0.0.0.0",
			port: 5173,
			proxy: {
				[`${apiBasePrefix}/v1`]: {
					target: proxyTarget,
					changeOrigin: true,
					timeout: Number.isFinite(proxyTimeout) ? proxyTimeout : 10000
				},
				[`${apiBasePrefix}/health`]: {
					target: proxyTarget,
					changeOrigin: true,
					timeout: Number.isFinite(proxyTimeout) ? proxyTimeout : 10000
				},
				[`${apiBasePrefix}/ready`]: {
					target: proxyTarget,
					changeOrigin: true,
					timeout: Number.isFinite(proxyTimeout) ? proxyTimeout : 10000
				},
				[`${apiBasePrefix}/docs`]: {
					target: proxyTarget,
					changeOrigin: true,
					timeout: Number.isFinite(proxyTimeout) ? proxyTimeout : 10000
				}
			}
		},
		build: {
			rollupOptions: {
				output: {
					manualChunks: {
						vue: ["vue", "vue-router", "pinia"],
						xlsx: ["xlsx"]
					}
				}
			}
		}
	};
});


