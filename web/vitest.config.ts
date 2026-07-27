import { defineConfig } from "vitest/config";
import vue from "@vitejs/plugin-vue";

const pluginSDKPath = decodeURIComponent(new URL("../packages/skoll-plugin-sdk/src/index.ts", import.meta.url).pathname)
	.replace(/^\/([A-Za-z]:\/)/, "$1");

export default defineConfig({
	plugins: [vue()],
	resolve: {
		alias: {
			"@skoll/plugin-sdk": pluginSDKPath
		}
	},
	test: {
		environment: "jsdom",
		setupFiles: ["./tests/setup.ts"],
		include: ["src/**/*.spec.ts"]
	}
});
