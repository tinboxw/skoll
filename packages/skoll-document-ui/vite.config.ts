import { resolve } from "node:path";

import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vite";
import Components from "unplugin-vue-components/vite";
import { ElementPlusResolver } from "unplugin-vue-components/resolvers";

export default defineConfig({
	plugins: [
		vue(),
		Components({ dts: false, resolvers: [ElementPlusResolver({ importStyle: false })] })
	],
	build: {
		lib: {
			entry: resolve(__dirname, "src/index.ts"),
			formats: ["es"],
			fileName: "index",
			cssFileName: "style"
		},
		rollupOptions: {
			external: (id) => ["vue", "element-plus", "lucide-vue-next"].some((dependency) => id === dependency || id.startsWith(`${dependency}/`))
		}
	},
	test: {
		environment: "jsdom",
		setupFiles: ["./tests/setup.ts"]
	}
});
