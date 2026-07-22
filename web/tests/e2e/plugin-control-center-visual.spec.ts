import { expect, test } from "@playwright/test";

import { PLUGIN_ID, preparePluginControlCenter, waitForPluginSurface } from "./plugin-control-center-fixtures";

test.describe.configure({ mode: "serial" });
const visualOptions = { fullPage: true, maxDiffPixelRatio: 0.001 } as const;

test("plugin fleet keeps its light-theme visual baseline", async ({ page }) => {
	await preparePluginControlCenter(page, { locale: "zh-CN", colorScheme: "light", density: "comfortable" });
	await page.goto("/skoll/plugin-center");
	await waitForPluginSurface(page, "[data-testid='plugin-fleet-table']");
	await expect(page).toHaveScreenshot("plugin-fleet-light.png", visualOptions);
});

test("degraded plugin overview keeps its dark-theme visual baseline", async ({ page }) => {
	await preparePluginControlCenter(page, { locale: "zh-CN", colorScheme: "dark", density: "comfortable" });
	await page.goto(`/skoll/plugin-center/${PLUGIN_ID}/overview`);
	await waitForPluginSurface(page, "[data-testid='plugin-overview']");
	await expect(page).toHaveScreenshot("plugin-overview-dark.png", visualOptions);
});

test("plugin diagnostics keeps its compact English visual baseline", async ({ page }) => {
	await preparePluginControlCenter(page, { locale: "en-US", colorScheme: "light", density: "compact" });
	await page.goto(`/skoll/plugin-center/${PLUGIN_ID}/diagnostics`);
	await waitForPluginSurface(page, "[data-testid='plugin-diagnostics']");
	await expect(page.getByText("Qualification expiry scan exceeded its retry budget", { exact: true })).toBeVisible();
	await expect(page).toHaveScreenshot("plugin-diagnostics-compact.png", visualOptions);
});
