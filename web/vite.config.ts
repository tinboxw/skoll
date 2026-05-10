import { defineConfig, loadEnv } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig(({ mode }) => {
	const env = loadEnv(mode, process.cwd(), "SKOLL_");
	const proxyTarget = (env.SKOLL_API_PROXY_TARGET || "http://127.0.0.1:8080").trim();
	const proxyTimeout = Number.parseInt(env.SKOLL_API_PROXY_TIMEOUT_MS || "10000", 10);

	return {
		plugins: [vue()],
		server: {
			host: "0.0.0.0",
			port: 5173,
			proxy: {
				"/v1": {
					target: proxyTarget,
					changeOrigin: true,
					timeout: Number.isFinite(proxyTimeout) ? proxyTimeout : 10000
				},
				"/health": {
					target: proxyTarget,
					changeOrigin: true,
					timeout: Number.isFinite(proxyTimeout) ? proxyTimeout : 10000
				}
			}
		}
	};
});

