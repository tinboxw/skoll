import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import Components from "unplugin-vue-components/vite";
import { ElementPlusResolver } from "unplugin-vue-components/resolvers";

const documentUIPath = decodeURIComponent(new URL("../skoll-document-ui/src/index.ts", import.meta.url).pathname)
	.replace(/^\/([A-Za-z]:\/)/, "$1");

export default defineConfig({
	plugins: [vue(), Components({ resolvers: [ElementPlusResolver({ importStyle: false })], dts: false })],
	resolve: {
		alias: {
			"@skoll/document-ui": documentUIPath
		}
	},
	build: {
		lib: {
			entry: {
				index: "src/index.ts",
				core: "src/core.ts"
			},
			formats: ["es"],
			fileName: (_format, entryName) => `${entryName}.js`
		},
		rollupOptions: {
			external: (id) => ["@skoll/document-ui", "vue", "element-plus", "lucide-vue-next"]
				.some((dependency) => id === dependency || id.startsWith(`${dependency}/`))
		}
	},
	test: {
		environment: "jsdom",
		setupFiles: ["./tests/setup.ts"]
	}
});
