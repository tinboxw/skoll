import { expect, test, type Page, type TestInfo } from "@playwright/test";

const PLUGIN_ID = "pharma_oa";

test("plugin fleet and routed workspace stay operational", async ({ page }, testInfo) => {
	await login(page, "admin", "Admin@123456");
	await mockDataControl(page);

	await page.goto("/skoll/plugin-center");
	await expect(page.locator("[data-testid='plugin-fleet-table']")).toBeVisible();
	await expect(page.locator("[data-testid='plugin-fleet-table'] .el-table__row")).not.toHaveCount(0);
	await assertNoHorizontalOverflow(page, "fleet");

	const surfaces = [
		["overview", "[data-testid='plugin-overview']"],
		["runtime", "[data-testid='plugin-runtime']"],
		["capabilities", "[data-testid='plugin-capabilities']"],
		["data", "[data-testid='plugin-data']"],
		["migrations", "[data-testid='plugin-migrations']"],
		["settings", "[data-testid='plugin-settings']"]
	] as const;

	for (const [route, selector] of surfaces) {
		await test.step(route, async () => {
			await page.goto(`/skoll/plugin-center/${PLUGIN_ID}/${route}`);
			await expect(page.locator(".page-shell")).toHaveAttribute("aria-busy", "false");
			await expect(page.locator(selector)).toBeVisible();
			await expect(page.locator("[data-testid='plugin-runtime-state']")).toBeVisible();
			if (route === "data") await expect(page.getByText("plugin_pharma_oa_documents", { exact: true })).toBeVisible();
			await assertNoHorizontalOverflow(page, route);
		});
	}

	await expect(page.getByRole("button", { name: /停用|Disable/ })).toBeVisible();
	await expect(page.getByRole("button", { name: /卸载|Uninstall/ })).toBeVisible();
	await page.goto(`/skoll/plugin-center/${PLUGIN_ID}/migrations`);
	await expect(page.locator("[data-testid='plugin-migration-blocked']")).toBeVisible();
	await expect(page.locator("[data-testid='plugin-migration-rollback']")).toHaveCount(0);
	await attachScreenshot(page, testInfo, "plugin-control-center");
});

test("plugin control center exposes a distinct forbidden state", async ({ page }) => {
	await login(page, "dept_admin", "Dept@123456");
	await page.goto(`/skoll/plugin-center/${PLUGIN_ID}/overview`);
	await expect(page).toHaveURL(/\/skoll\/forbidden/);
	await expect(page.locator(".state-block--forbidden")).toBeVisible();
	await assertNoHorizontalOverflow(page, "forbidden");
});

async function login(page: Page, account: string, password: string): Promise<void> {
	await page.goto("/skoll/login");
	await page.evaluate(() => {
		localStorage.clear();
		localStorage.setItem("skoll.ui.locale", "zh-CN");
	});
	await page.reload();
	await page.locator("input[autocomplete='username']:visible").fill(account);
	await page.locator("input[autocomplete='current-password']:visible").fill(password);
	await page.getByRole("button", { name: "登录", exact: true }).click();
	await expect(page).not.toHaveURL(/\/login/);
}

async function assertNoHorizontalOverflow(page: Page, state: string): Promise<void> {
	const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
	expect(overflow, `${state}: document horizontal overflow`).toBeLessThanOrEqual(1);
}

async function attachScreenshot(page: Page, testInfo: TestInfo, name: string): Promise<void> {
	const screenshot = await page.screenshot({ fullPage: true });
	await testInfo.attach(name, { body: screenshot, contentType: "image/png" });
}

async function mockDataControl(page: Page): Promise<void> {
	await page.route(`**/skoll/v1/plugins/${PLUGIN_ID}/data-control`, async (route) => {
		await route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify({
				code: "ok",
				message: "ok",
				data: {
					pluginId: PLUGIN_ID,
					capturedAt: new Date().toISOString(),
					state: "enabled",
					schema: {
						available: true,
						registered: true,
						namespace: "plugin_pharma_oa",
						totalSizeBytes: 16384,
						sizeKnown: true,
						tables: [{
							logicalName: "documents",
							physicalName: "plugin_pharma_oa_documents",
							fields: ["id", "status", "created_at"],
							primaryKey: ["id"],
							indexCount: 2,
							exists: true,
							sizeBytes: 16384,
							sizeKnown: true
						}]
					},
					migration: {
						declaredVersion: "1.0.0",
						currentVersion: 1,
						applied: [{ version: 1, name: "create_documents", checksum: "sha256:test", appliedAt: new Date().toISOString() }],
						pending: []
					},
					policy: { uninstall: "retain", rollback: "automatic", effect: "retain_data" },
					actions: { canRollback: false, rollbackMaxSteps: 1, blockedReason: "plugin_must_be_disabled" }
				}
			})
		});
	});
}
