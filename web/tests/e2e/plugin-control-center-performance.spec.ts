import fs from "node:fs";
import path from "node:path";
import { expect, test, type CDPSession, type Page } from "@playwright/test";

import { PLUGIN_ID, preparePluginControlCenter, waitForPluginSurface } from "./plugin-control-center-fixtures";

type PluginRuntimeBudgets = {
	routeReadyMs: number;
	pluginListItems: number;
	pluginListReadyMs: number;
	pluginListRenderedRows: number;
	interactionMs: number;
	maxLongTaskMs: number;
	totalLongTaskMs: number;
	heapGrowthBytes: number;
};

type PluginRuntimeReport = {
	project: string;
	routes: Record<string, number>;
	pluginListReadyMs: number;
	pluginListRenderedRows: number;
	interactionMs: number;
	maxLongTaskMs: number;
	totalLongTaskMs: number;
	heapGrowthBytes: number;
};

const budgets = JSON.parse(fs.readFileSync(path.resolve(process.cwd(), "config/frontend-quality-budgets.json"), "utf8")).pluginControlCenter.runtime as PluginRuntimeBudgets;

const SURFACES = [
	["overview", "Overview", "[data-testid='plugin-overview']"],
	["runtime", "Runtime", "[data-testid='plugin-runtime']"],
	["capabilities", "Capabilities", "[data-testid='plugin-capabilities']"],
	["data", "Data", "[data-testid='plugin-data']"],
	["migrations", "Migrations", "[data-testid='plugin-migrations']"],
	["jobs", "Jobs", "[data-testid='plugin-jobs']"],
	["audit", "Audit", "[data-testid='plugin-audit']"],
	["diagnostics", "Diagnostics", "[data-testid='plugin-diagnostics']"],
	["settings", "Settings", "[data-testid='plugin-settings']"]
] as const;

test("plugin control center stays inside frozen runtime budgets", async ({ page }, testInfo) => {
	await installLongTaskObserver(page);
	await preparePluginControlCenter(page, { locale: "en-US", colorScheme: "light", density: "compact" }, budgets.pluginListItems);
	await readAndResetLongTasks(page);
	const longTasks: number[] = [];

	const listStarted = performance.now();
	await page.goto("/skoll/plugin-center");
	await waitForPluginSurface(page, "[data-testid='plugin-fleet-table']");
	const rows = page.locator("[data-testid='plugin-fleet-table'] .el-table__row");
	await expect(rows).toHaveCount(budgets.pluginListRenderedRows);
	const pluginListReadyMs = performance.now() - listStarted;
	const pluginListRenderedRows = await rows.count();
	await readAndResetLongTasks(page);

	const interactionStarted = performance.now();
	await page.locator(".fleet-toolbar input:visible").first().fill("medical_plugin_118");
	await expect(rows).toHaveCount(1);
	await page.evaluate(() => new Promise<void>((resolve) => requestAnimationFrame(() => requestAnimationFrame(() => resolve()))));
	const interactionMs = performance.now() - interactionStarted;
	longTasks.push(...await readAndResetLongTasks(page));

	const routes: Record<string, number> = {};
	for (const [index, [route, label, selector]] of SURFACES.entries()) {
		const started = performance.now();
		if (index === 0) {
			await page.goto(`/skoll/plugin-center/${PLUGIN_ID}/${route}`);
		} else {
			await page.getByRole("tab", { name: label, exact: true }).click();
			await expect(page).toHaveURL(new RegExp(`/skoll/plugin-center/${PLUGIN_ID}/${route}$`));
		}
		await waitForPluginSurface(page, selector);
		routes[route] = performance.now() - started;
		await readAndResetLongTasks(page);
	}

	const session = await page.context().newCDPSession(page);
	await session.send("Performance.enable");
	await session.send("HeapProfiler.collectGarbage");
	const heapBefore = await readHeap(session);
	for (let cycle = 0; cycle < 2; cycle += 1) {
		for (const [route, label, selector] of SURFACES) {
			await page.getByRole("tab", { name: label, exact: true }).click();
			await expect(page).toHaveURL(new RegExp(`/skoll/plugin-center/${PLUGIN_ID}/${route}$`));
			await waitForPluginSurface(page, selector);
			longTasks.push(...await readAndResetLongTasks(page));
		}
	}
	await session.send("HeapProfiler.collectGarbage");
	const heapGrowthBytes = Math.max(0, (await readHeap(session)) - heapBefore);

	const report: PluginRuntimeReport = {
		project: testInfo.project.name,
		routes,
		pluginListReadyMs,
		pluginListRenderedRows,
		interactionMs,
		maxLongTaskMs: longTasks.length === 0 ? 0 : Math.max(...longTasks),
		totalLongTaskMs: longTasks.reduce((total, duration) => total + duration, 0),
		heapGrowthBytes
	};
	await testInfo.attach("plugin-control-center-performance.json", {
		body: Buffer.from(JSON.stringify(report, null, 2)),
		contentType: "application/json"
	});

	expect(report.pluginListReadyMs, `${budgets.pluginListItems}-plugin fleet readiness`).toBeLessThanOrEqual(budgets.pluginListReadyMs);
	expect(report.pluginListRenderedRows, "plugin fleet DOM row bound").toBeLessThanOrEqual(budgets.pluginListRenderedRows);
	expect(report.interactionMs, "plugin fleet filter interaction").toBeLessThanOrEqual(budgets.interactionMs);
	for (const [route, duration] of Object.entries(report.routes)) {
		expect(duration, `${route} readiness`).toBeLessThanOrEqual(budgets.routeReadyMs);
	}
	expect(report.maxLongTaskMs, "maximum plugin workspace long task").toBeLessThanOrEqual(budgets.maxLongTaskMs);
	expect(report.totalLongTaskMs, "total plugin workspace long-task time").toBeLessThanOrEqual(budgets.totalLongTaskMs);
	expect(report.heapGrowthBytes, "plugin workspace route-cycle heap growth").toBeLessThanOrEqual(budgets.heapGrowthBytes);
});

async function installLongTaskObserver(page: Page): Promise<void> {
	await page.addInitScript(() => {
		const target = window as Window & { __skollPluginLongTasks?: number[] };
		target.__skollPluginLongTasks = [];
		try {
			new PerformanceObserver((entries) => {
				target.__skollPluginLongTasks?.push(...entries.getEntries().map((entry) => entry.duration));
			}).observe({ type: "longtask", buffered: true });
		} catch {
			target.__skollPluginLongTasks = [];
		}
	});
}

async function readAndResetLongTasks(page: Page): Promise<number[]> {
	return page.evaluate(() => {
		const target = window as Window & { __skollPluginLongTasks?: number[] };
		const entries = [...(target.__skollPluginLongTasks || [])];
		target.__skollPluginLongTasks = [];
		return entries;
	});
}

async function readHeap(session: CDPSession): Promise<number> {
	const response = await session.send("Performance.getMetrics");
	return response.metrics.find((metric) => metric.name === "JSHeapUsedSize")?.value || 0;
}
