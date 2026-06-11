import { createApp } from "vue";
import { createPinia } from "pinia";

import App from "./App.vue";
import "./styles/variables.scss";
import "./styles/global.scss";
import { router } from "./router";
import { bootstrapPlugins } from "./plugins";
import { installPermissionDirective } from "./permissions/directive";
import { usePluginStore } from "./stores/plugins";

async function start(): Promise<void> {
	const app = createApp(App);
	const pinia = createPinia();

	app.use(pinia);
	app.use(router);
	installPermissionDirective(app);
	app.mount("#app");

	const pluginStore = usePluginStore(pinia);
	void bootstrapPlugins(router, pluginStore).catch((error) => {
		// eslint-disable-next-line no-console
		console.error("plugin bootstrap failed", error);
	});
}

start().catch((error) => {
	// eslint-disable-next-line no-console
	console.error("web bootstrap failed", error);
});

