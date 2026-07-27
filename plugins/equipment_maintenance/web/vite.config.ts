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
  plugins: [vue(), Components({ resolvers: [ElementPlusResolver()], dts: false })],
  build: {
    outDir: "dist",
    emptyOutDir: true,
    manifest: true
  },
  test: {
    environment: "jsdom"
  }
});
