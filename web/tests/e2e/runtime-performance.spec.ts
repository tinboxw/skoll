import fs from "node:fs";
import path from "node:path";
import { expect, test, type CDPSession, type Page } from "@playwright/test";

type RuntimeBudgets = {
	loginReadyMs: number;
	routeReadyMs: number;
	interactionMs: number;
	largeListReadyMs: number;
	largeListRenderedRows: number;
	maxLongTaskMs: number;
	totalLongTaskMs: number;
	heapGrowthBytes: number;
};

type RuntimeReport = {
	project: string;
	loginReadyMs: number;
	routes: Record<string, number>;
	interactionMs: number;
	largeListReadyMs: number;
	largeListRenderedRows: number;
	maxLongTaskMs: number;
	totalLongTaskMs: number;
	heapGrowthBytes: number;
};

const budgets = JSON.parse(fs.readFileSync(path.resolve(process.cwd(), "config/frontend-quality-budgets.json"), "utf8")).runtime as RuntimeBudgets;

test("critical routes stay inside runtime budgets", async ({ page, request }, testInfo) => {
	const health = await request.get("/skoll/health");
	expect(health.ok()).toBeTruthy();
	await installLongTaskObserver(page);

	const loginStarted = performance.now();
	await login(page);
	const loginReadyMs = performance.now() - loginStarted;

	const routes: Record<string, number> = {};
	for (const route of ["/skoll/pharma-oa/dashboard", "/skoll/plugin", "/skoll/form-builder"]) {
		routes[route] = await measureRoute(page, route);
	}

	const customers = Array.from({ length: 50 }, (_, index) => ({
		id: `perf-customer-${index}`,
		code: `C${String(index).padStart(5, "0")}`,
		name: `Performance Customer ${index}`,
		region: index % 2 === 0 ? "East" : "West",
		organizationId: "org-root",
		ownerId: "admin",
		rating: (index % 5) + 1,
		status: "active",
		contacts: [],
		qualifications: []
	}));
	const requestedLimits: string[] = [];
	await page.route("**/skoll/v1/plugins/pharma_oa/api/customers?**", async (route) => {
		requestedLimits.push(new URL(route.request().url()).searchParams.get("limit") || "");
		await route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify({
				code: "ok",
				message: "ok",
				data: { items: customers, offset: 0, limit: 50, total: 10_000, hasMore: true, nextCursor: "perf-next", sort: "code:asc" }
			})
		});
	});

	const session = await page.context().newCDPSession(page);
	await session.send("Performance.enable");
	await session.send("HeapProfiler.collectGarbage");
	const heapBefore = await readHeap(session);
	const largeListStarted = performance.now();
	await page.goto("/skoll/pharma-oa/customers");
	await expect(page.locator(".page-shell")).toHaveAttribute("aria-busy", "false");
	await expect(page.locator(".data-table")).toBeVisible();
	const largeListReadyMs = performance.now() - largeListStarted;
	const largeListRenderedRows = await page.locator(".el-table-v2__row").count();
	expect(requestedLimits, "large datasets must stay server paginated").toContain("50");

	const interactionStarted = performance.now();
	const keyword = page.locator(".filter-bar input:visible").first();
	await keyword.fill("Performance Customer 4999");
	await expect(page.locator(".page-shell")).toHaveAttribute("aria-busy", "false");
	await page.evaluate(() => new Promise<void>((resolve) => requestAnimationFrame(() => requestAnimationFrame(() => resolve()))));
	const interactionMs = performance.now() - interactionStarted;

	await page.goto("/skoll/pharma-oa/dashboard");
	await expect(page.locator(".page-shell")).toHaveAttribute("aria-busy", "false");
	await page.goto("/skoll/pharma-oa/customers");
	await expect(page.locator(".page-shell")).toHaveAttribute("aria-busy", "false");
	await session.send("HeapProfiler.collectGarbage");
	const heapGrowthBytes = Math.max(0, (await readHeap(session)) - heapBefore);
	const longTasks = await page.evaluate(() => (window as Window & { __skollLongTasks?: number[] }).__skollLongTasks || []);
	const report: RuntimeReport = {
		project: testInfo.project.name,
		loginReadyMs,
		routes,
		interactionMs,
		largeListReadyMs,
		largeListRenderedRows,
		maxLongTaskMs: longTasks.length === 0 ? 0 : Math.max(...longTasks),
		totalLongTaskMs: longTasks.reduce((total, duration) => total + duration, 0),
		heapGrowthBytes
	};
	await testInfo.attach("runtime-performance.json", { body: Buffer.from(JSON.stringify(report, null, 2)), contentType: "application/json" });

	expect(report.loginReadyMs, "authenticated shell readiness").toBeLessThanOrEqual(budgets.loginReadyMs);
	for (const [route, duration] of Object.entries(report.routes)) {
		expect(duration, `${route} readiness`).toBeLessThanOrEqual(budgets.routeReadyMs);
	}
	expect(report.interactionMs, "large-list filter interaction").toBeLessThanOrEqual(budgets.interactionMs);
	expect(report.largeListReadyMs, "10,000-record dataset page readiness").toBeLessThanOrEqual(budgets.largeListReadyMs);
	expect(report.largeListRenderedRows, "virtual list DOM row bound").toBeLessThanOrEqual(budgets.largeListRenderedRows);
	expect(report.maxLongTaskMs, "maximum long task").toBeLessThanOrEqual(budgets.maxLongTaskMs);
	expect(report.totalLongTaskMs, "total long-task time").toBeLessThanOrEqual(budgets.totalLongTaskMs);
	expect(report.heapGrowthBytes, "route-cycle heap growth").toBeLessThanOrEqual(budgets.heapGrowthBytes);
});

async function installLongTaskObserver(page: Page): Promise<void> {
	await page.addInitScript(() => {
		const target = window as Window & { __skollLongTasks?: number[] };
		target.__skollLongTasks = [];
		try {
			new PerformanceObserver((entries) => {
				target.__skollLongTasks?.push(...entries.getEntries().map((entry) => entry.duration));
			}).observe({ type: "longtask", buffered: true });
		} catch {
			target.__skollLongTasks = [];
		}
	});
}

async function login(page: Page): Promise<void> {
	await page.goto("/skoll/login");
	await page.evaluate(() => {
		localStorage.clear();
		localStorage.setItem("skoll.ui.locale", "en-US");
	});
	await page.reload();
	await page.locator("input[autocomplete='username']").fill("admin");
	await page.locator("input[autocomplete='current-password']").fill("Admin@123456");
	await page.locator("button[type='submit']").click();
	await expect(page).not.toHaveURL(/\/login/);
	await expect(page.locator(".header__status")).toHaveAttribute("aria-label", "Synced");
}

async function measureRoute(page: Page, route: string): Promise<number> {
	const started = performance.now();
	await page.goto(route);
	await expect(page.locator(".page-shell")).toHaveAttribute("aria-busy", "false");
	return performance.now() - started;
}

async function readHeap(session: CDPSession): Promise<number> {
	const response = await session.send("Performance.getMetrics");
	return response.metrics.find((metric) => metric.name === "JSHeapUsedSize")?.value || 0;
}
