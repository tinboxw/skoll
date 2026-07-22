export function isHostedPluginRoute(routeName: unknown, routePath: unknown): boolean {
	const name = String(routeName || "");
	const path = String(routePath || "");
	const isDynamicPluginPage = name.startsWith("plugin-") && !name.startsWith("plugin-center");
	return isDynamicPluginPage || name.startsWith("app-home-") || path.startsWith("/skoll/plugins/");
}
