import { createApp } from "vue";
import { createPinia } from "pinia";

import App from "./App.vue";
import { router } from "./router";
import { bootstrapPlugins } from "./plugins";
import { usePluginStore } from "./stores/plugins";

async function start(): Promise<void> {
	const app = createApp(App);
	const pinia = createPinia();

	app.use(pinia);
	app.use(router);

	const pluginStore = usePluginStore(pinia);
	await bootstrapPlugins(router, pluginStore);

	app.mount("#app");
}

start().catch((error) => {
	// eslint-disable-next-line no-console
	console.error("web bootstrap failed", error);
});

