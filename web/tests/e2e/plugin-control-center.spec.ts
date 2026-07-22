import { expect, test, type Page, type TestInfo } from "@playwright/test";

const PLUGIN_ID = "pharma_oa";

const APPEARANCE_MATRIX = [
	{ locale: "zh-CN", colorScheme: "light", density: "comfortable", navigation: "插件工作区导航", overview: "概览" },
	{ locale: "zh-CN", colorScheme: "dark", density: "compact", navigation: "插件工作区导航", overview: "概览" },
	{ locale: "en-US", colorScheme: "light", density: "compact", navigation: "Plugin workspace navigation", overview: "Overview" },
	{ locale: "en-US", colorScheme: "dark", density: "comfortable", navigation: "Plugin workspace navigation", overview: "Overview" }
] as const;

test("plugin fleet and routed workspace stay operational", async ({ page }, testInfo) => {
	await login(page, "admin", "Admin@123456");
	await mockDataControl(page);
	await mockDiagnostics(page);

	await page.goto("/skoll/plugin-center");
	await expect(page.locator(".content-area")).not.toHaveClass(/plugin-content-area/);
	await expect(page.locator(".main-content")).not.toHaveClass(/plugin-main-content/);
	await expect(page.locator("[data-testid='plugin-fleet-table']")).toBeVisible();
	await expect(page.locator("[data-testid='plugin-fleet-table'] .el-table__row")).not.toHaveCount(0);
	await expect(page.locator("a[href='/skoll/workflow']").first()).toHaveText("工作流");
	await expect(page.locator("a[href='/skoll/todo']").first()).toHaveText("待办中心");
	await expect(page.locator("a[href='/skoll/form-builder']").first()).toHaveText("表单设计器");
	await assertNoHorizontalOverflow(page, "fleet");

	const surfaces = [
		["overview", "[data-testid='plugin-overview']"],
		["runtime", "[data-testid='plugin-runtime']"],
		["capabilities", "[data-testid='plugin-capabilities']"],
		["data", "[data-testid='plugin-data']"],
		["migrations", "[data-testid='plugin-migrations']"],
		["jobs", "[data-testid='plugin-jobs']"],
		["audit", "[data-testid='plugin-audit']"],
		["diagnostics", "[data-testid='plugin-diagnostics']"],
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
	await page.goto(`/skoll/plugin-center/${PLUGIN_ID}/jobs`);
	await expect(page.getByText("qualification-expiry", { exact: true })).toBeVisible();
	await expect(page.locator("[data-testid='plugin-job-status-dead_letter']")).toBeVisible();
	await expect(page.locator("[data-testid='plugin-job-retry']")).toBeEnabled();
	await page.goto(`/skoll/plugin-center/${PLUGIN_ID}/diagnostics?correlation=trace-pharma-1`);
	await expect(page.getByText("trace-pharma-1", { exact: true })).toBeVisible();
	const correlatedError = page.getByText("audit:audit-pharma-1", { exact: true });
	await expect(correlatedError).toBeVisible();
	await correlatedError.scrollIntoViewIfNeeded();
	await attachScreenshot(page, testInfo, "plugin-control-center");
});

test("plugin workspace passes locale, theme, density, keyboard, and reduced-motion matrix", async ({ page }, testInfo) => {
	await page.emulateMedia({ reducedMotion: "reduce" });
	await login(page, "admin", "Admin@123456");
	await mockDataControl(page);
	await mockDiagnostics(page);

	for (const appearance of APPEARANCE_MATRIX) {
		await test.step(`${appearance.locale}-${appearance.colorScheme}-${appearance.density}`, async () => {
			await page.evaluate((value) => {
				localStorage.setItem("skoll.ui.locale", value.locale);
				localStorage.setItem("skoll.ui.colorScheme", value.colorScheme);
				localStorage.setItem("skoll.ui.density", value.density);
			}, appearance);
			await page.goto(`/skoll/plugin-center/${PLUGIN_ID}/overview`);
			await expect(page.locator(".page-shell")).toHaveAttribute("aria-busy", "false");
			await expect(page.locator("html")).toHaveAttribute("lang", appearance.locale);
			await expect(page.locator("html")).toHaveAttribute("data-theme", appearance.colorScheme);
			await expect(page.locator("html")).toHaveAttribute("data-density", appearance.density);

			const navigation = page.getByRole("navigation", { name: appearance.navigation });
			await expect(navigation).toBeVisible();
			const tabs = navigation.getByRole("tab");
			await expect(tabs.first()).toHaveText(appearance.overview);
			await tabs.first().focus();
			await expect(tabs.first()).toBeFocused();
			await page.keyboard.press("ArrowRight");
			await expect(tabs.nth(1)).toBeFocused();
			await expect(page).toHaveURL(new RegExp(`/skoll/plugin-center/${PLUGIN_ID}/runtime$`));

			const pageShell = page.locator(".page-shell");
			const titleId = await pageShell.getAttribute("aria-labelledby");
			expect(titleId).toBeTruthy();
			await expect(page.locator(`#${titleId}`)).toBeVisible();
			expect(await page.evaluate(() => matchMedia("(prefers-reduced-motion: reduce)").matches)).toBe(true);
			await assertResponsiveDescriptions(page, `${appearance.locale}-${appearance.colorScheme}-${appearance.density}`);
			await assertNoHorizontalOverflow(page, `${appearance.locale}-${appearance.colorScheme}-${appearance.density}`);
			await attachScreenshot(page, testInfo, `${appearance.locale}-${appearance.colorScheme}-${appearance.density}`);
		});
	}
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

async function assertResponsiveDescriptions(page: Page, state: string): Promise<void> {
	const violations = await page.locator(".responsive-descriptions").evaluateAll((elements) => elements.flatMap((element, index) => {
		const container = element as HTMLElement;
		const cells = [...container.querySelectorAll<HTMLElement>(".el-descriptions__cell")];
		const escaped = cells.filter((cell) => {
			const rect = cell.getBoundingClientRect();
			return rect.left < -1 || rect.right > window.innerWidth + 1;
		});
		return container.scrollWidth > container.clientWidth + 1 || escaped.length > 0
			? [`descriptions-${index}: scroll=${container.scrollWidth}/${container.clientWidth}, escaped=${escaped.length}`]
			: [];
	}));
	expect(violations, `${state}: responsive descriptions fit the viewport`).toEqual([]);
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

async function mockDiagnostics(page: Page): Promise<void> {
	await page.route(`**/skoll/v1/plugins/${PLUGIN_ID}/diagnostics*`, async (route) => {
		const now = new Date().toISOString();
		await route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify({
				code: "ok",
				message: "ok",
				data: {
					pluginId: PLUGIN_ID,
					capturedAt: now,
					health: { pluginId: PLUGIN_ID, status: "unhealthy", code: "health_timeout", checkedAt: now, latencyMillis: 2000 },
					summary: { totalJobs: 2, activeJobs: 1, deadLetters: 1, auditEvents: 2, failureCount: 2 },
					jobs: [
						{ id: "qualification-expiry", kind: "expiry_scan", status: "dead_letter", runAt: now, maxAttempts: 3, attemptCount: 3, lastError: "qualification lookup failed", createdAt: now, updatedAt: now, deadLetteredAt: now, canRetry: true },
						{ id: "stock-alert", kind: "stock_scan", status: "scheduled", runAt: now, maxAttempts: 3, attemptCount: 0, createdAt: now, updatedAt: now, canRetry: false }
					],
					audit: [
						{ id: "audit-pharma-1", source: "event", action: "pharma_oa.customer.read", result: "failure", risk: "medium", actorId: "admin", resourceType: "plugin_route", resourceId: "GET /v1/plugins/pharma_oa/api/customers", occurredAt: now, traceId: "trace-pharma-1", requestId: "request-pharma-1", method: "GET", path: "/v1/plugins/pharma_oa/api/customers" },
						{ id: "audit-pharma-2", source: "host", action: "plugin.pharma_oa.job.fail", result: "failure", risk: "high", actorId: "plugin:pharma_oa", resourceType: "plugin:pharma_oa:job", resourceId: "qualification-expiry", occurredAt: now }
					],
					errors: [
						{ id: "audit:audit-pharma-1", category: "route", severity: "medium", summary: "pharma_oa.customer.read", occurredAt: now, correlation: { routeId: "GET /v1/plugins/pharma_oa/api/customers", auditId: "audit-pharma-1", requestId: "request-pharma-1", traceId: "trace-pharma-1" } },
						{ id: "job:qualification-expiry", category: "job", severity: "high", summary: "qualification lookup failed", occurredAt: now, correlation: { jobId: "qualification-expiry" } }
					]
				}
			})
		});
	});
}
