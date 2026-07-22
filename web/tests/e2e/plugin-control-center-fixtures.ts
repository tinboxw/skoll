import { expect, type Page } from "@playwright/test";

export const PLUGIN_ID = "pharma_oa";
export const FIXED_TIME = "2026-07-23T02:30:00.000Z";

type Appearance = {
	locale?: "zh-CN" | "en-US";
	colorScheme?: "light" | "dark";
	density?: "comfortable" | "compact";
};

export async function preparePluginControlCenter(page: Page, appearance: Appearance = {}, pluginCount = 6): Promise<void> {
	await mockPluginList(page, pluginCount);
	await mockPluginControl(page);
	await mockPluginDataControl(page);
	await mockPluginDiagnostics(page);
	await page.goto("/skoll/login");
	await page.evaluate((settings) => {
		localStorage.clear();
		localStorage.setItem("skoll.ui.locale", settings.locale || "zh-CN");
		localStorage.setItem("skoll.ui.colorScheme", settings.colorScheme || "light");
		localStorage.setItem("skoll.ui.density", settings.density || "comfortable");
	}, appearance);
	await page.reload();
	await page.locator("input[autocomplete='username']:visible").fill("admin");
	await page.locator("input[autocomplete='current-password']:visible").fill("Admin@123456");
	await page.locator("button[type='submit']:visible").click();
	await expect(page).not.toHaveURL(/\/login/);
}

export async function waitForPluginSurface(page: Page, selector: string): Promise<void> {
	await expect(page.locator(".page-shell")).toHaveAttribute("aria-busy", "false");
	await expect(page.locator(selector)).toBeVisible();
}

function pluginRecords(count: number): Array<Record<string, unknown>> {
	const backendRecordCount = Math.max(1, count - 1);
	return Array.from({ length: backendRecordCount }, (_, index) => {
		const id = index === 0 ? PLUGIN_ID : `medical_plugin_${String(index).padStart(3, "0")}`;
		return {
			id,
			name: index === 0 ? "Medical OA" : `Medical Extension ${String(index).padStart(3, "0")}`,
			nameZhCN: index === 0 ? "医药 OA" : `医药扩展 ${String(index).padStart(3, "0")}`,
			nameEnUS: index === 0 ? "Medical OA" : `Medical Extension ${String(index).padStart(3, "0")}`,
			version: `1.${index % 10}.0`,
			enabled: index % 7 !== 0 || index === 0,
			uiMode: "backend_only",
			level: "system",
			mountPolicy: "admin",
			uiNavPosition: "none",
			uiOpenMode: "integrated",
			uiTabMode: "disabled",
			systemBuiltin: false
		};
	});
}

async function mockPluginList(page: Page, count: number): Promise<void> {
	await page.route("**/skoll/v1/plugins", async (route) => {
		await route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify({ code: "ok", message: "ok", data: pluginRecords(count) })
		});
	});
}

async function mockPluginControl(page: Page): Promise<void> {
	await page.route(`**/skoll/v1/plugins/${PLUGIN_ID}/control`, async (route) => {
		await route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify({
				code: "ok",
				message: "ok",
				data: {
					plugin: { id: PLUGIN_ID, name: "Medical OA", version: "1.8.0", enabled: true, uiMode: "backend_only", level: "system" },
					capturedAt: FIXED_TIME,
					staleAfter: "2026-07-23T02:30:30.000Z",
					runtime: {
						state: "enabled",
						installedAt: "2026-07-20T08:00:00.000Z",
						enabledAt: "2026-07-23T01:00:00.000Z",
						health: { pluginId: PLUGIN_ID, status: "unhealthy", code: "qualification_expiry_lag", checkedAt: FIXED_TIME, latencyMillis: 128 }
					},
					capabilities: {
						apiVersion: "v1",
						migrationVersion: "8",
						hostServices: ["transactions", "data-scopes", "documents", "files", "audit", "workflows", "jobs"],
						permissions: [
							{ key: "pharma.employee.read", type: "data", module: "employee", name: "Read employees", risk: "low" },
							{ key: "pharma.inventory.manage", type: "action", module: "inventory", name: "Manage inventory", risk: "high" }
						],
						routes: [
							{ method: "GET", path: "/v1/plugins/pharma_oa/api/employees", summary: "List employees", permission: "pharma.employee.read", source: "manifest" },
							{ method: "POST", path: "/v1/plugins/pharma_oa/api/stock-movements", summary: "Create stock movement", permission: "pharma.inventory.manage", source: "manifest" }
						],
						dependencies: [],
						extensions: { routes: 2, middlewares: 1, events: 4, menus: 5, widgets: 2, settings: 3 }
					}
				}
			})
		});
	});
}

async function mockPluginDataControl(page: Page): Promise<void> {
	await page.route(`**/skoll/v1/plugins/${PLUGIN_ID}/data-control`, async (route) => {
		await route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify({
				code: "ok",
				message: "ok",
				data: {
					pluginId: PLUGIN_ID,
					capturedAt: FIXED_TIME,
					state: "enabled",
					schema: { available: true, registered: true, namespace: "plugin_pharma_oa", totalSizeBytes: 16384, sizeKnown: true, tables: [{ logicalName: "documents", physicalName: "plugin_pharma_oa_documents", fields: ["id", "status", "created_at"], primaryKey: ["id"], indexCount: 2, exists: true, sizeBytes: 16384, sizeKnown: true }] },
					migration: { declaredVersion: "8", currentVersion: 8, applied: [{ version: 8, name: "add_qualification_alerts", checksum: "sha256:fixed", appliedAt: FIXED_TIME }], pending: [] },
					policy: { uninstall: "retain", rollback: "automatic", effect: "retain_data" },
					actions: { canRollback: false, rollbackMaxSteps: 1, blockedReason: "plugin_must_be_disabled" }
				}
			})
		});
	});
}

async function mockPluginDiagnostics(page: Page): Promise<void> {
	await page.route(`**/skoll/v1/plugins/${PLUGIN_ID}/diagnostics*`, async (route) => {
		await route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify({
				code: "ok",
				message: "ok",
				data: {
					pluginId: PLUGIN_ID,
					capturedAt: FIXED_TIME,
					health: { pluginId: PLUGIN_ID, status: "unhealthy", code: "qualification_expiry_lag", checkedAt: FIXED_TIME, latencyMillis: 128 },
					summary: { totalJobs: 18, activeJobs: 3, deadLetters: 1, auditEvents: 42, failureCount: 2 },
					jobs: [],
					audit: [],
					errors: [
						{ id: "job:qualification-expiry", category: "job", severity: "high", summary: "Qualification expiry scan exceeded its retry budget", occurredAt: FIXED_TIME, correlation: { jobId: "qualification-expiry", traceId: "trace-medical-001" } },
						{ id: "route:stock-movement", category: "route", severity: "medium", summary: "Stock movement request was denied by qualification policy", occurredAt: FIXED_TIME, correlation: { routeId: "POST /stock-movements", requestId: "request-medical-002" } }
					]
				}
			})
		});
	});
}
