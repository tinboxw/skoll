import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import Components from "unplugin-vue-components/vite";
import { ElementPlusResolver } from "unplugin-vue-components/resolvers";

const pluginSDKPath = decodeURIComponent(new URL("../../../packages/skoll-plugin-sdk/src/index.ts", import.meta.url).pathname)
  .replace(/^\/([A-Za-z]:\/)/, "$1");

export default defineConfig({
  base: "./",
  resolve: {
    alias: {
      "@skoll/plugin-sdk": pluginSDKPath
    }
  },
  plugins: [vue(), Components({ resolvers: [ElementPlusResolver({ importStyle: false })], dts: false })],
  build: {
    outDir: "dist",
    emptyOutDir: true,
    manifest: true,
    target: "es2022",
    cssCodeSplit: true,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes("node_modules/lucide-vue-next")) return "icons";
          if (id.includes("node_modules/vue") || id.includes("node_modules/@vue")) return "vue";
          return undefined;
        }
      }
    }
  },
  test: {
    environment: "jsdom"
  }
});
