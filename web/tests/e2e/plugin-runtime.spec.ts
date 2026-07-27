import fs from "node:fs";
import path from "node:path";
import { expect, test, type CDPSession, type Page } from "@playwright/test";

type RuntimeBudgets = {
	routeReadyMs: number;
	interactionMs: number;
	failureReadyMs: number;
	maxLongTaskMs: number;
	totalLongTaskMs: number;
	heapGrowthBytes: number;
};

type PageMode = "error" | "oversized" | "slow" | "success";

const PLUGIN_ID = "runtime_probe";
const PLUGIN_PATH = `/skoll/plugins/${PLUGIN_ID}`;
const budgets = JSON.parse(
	fs.readFileSync(path.resolve(process.cwd(), "config/frontend-quality-budgets.json"), "utf8")
).pluginRuntime.runtime as RuntimeBudgets;

const pluginHTML = `<!doctype html>
<html lang="zh-CN">
	<head><meta charset="UTF-8"><title>Runtime Probe</title></head>
	<body>
		<main>
			<h1>插件运行正常</h1>
			<button id="increment" type="button">执行操作</button>
			<output id="result">0</output>
		</main>
		<script>
			document.getElementById('increment').addEventListener('click', function () {
				var output = document.getElementById('result');
				output.textContent = String(Number(output.textContent) + 1);
			});
		</script>
	</body>
</html>`;

test("plugin runtime isolates failures, enforces limits, and recovers through retry", async ({ page }) => {
	let mode: PageMode = "error";
	await prepareRuntime(page, "runtime.access", () => mode);

	const failureStarted = performance.now();
	await page.goto(PLUGIN_PATH);
	await expect(page.locator("[data-testid='plugin-runtime-error']")).toBeVisible();
	expect(performance.now() - failureStarted).toBeLessThanOrEqual(budgets.failureReadyMs);
	await expect(page.locator(".header")).toBeVisible();

	mode = "success";
	await page.getByRole("button", { name: /重试|Retry/ }).click();
	const frame = page.locator("iframe.plugin-page-frame");
	await expect(frame).toBeVisible();
	await expect(frame).toHaveAttribute("loading", "lazy");
	await expect(frame).toHaveAttribute("sandbox", "allow-downloads allow-forms allow-same-origin allow-scripts");
	await expect(page.frameLocator("iframe.plugin-page-frame").getByRole("heading", { name: "插件运行正常" })).toBeVisible();

	mode = "oversized";
	await page.reload();
	await expect(page.locator("[data-testid='plugin-runtime-error']")).toContainText("1048576");

	mode = "slow";
	const slowStarted = performance.now();
	await page.getByRole("button", { name: /重试|Retry/ }).click();
	await expect(page.locator("[data-testid='plugin-runtime-error']")).toContainText(/5000ms|5,000ms/);
	expect(performance.now() - slowStarted).toBeLessThanOrEqual(budgets.failureReadyMs);
	await expect(page.locator(".header")).toBeVisible();
});

test("plugin runtime denies an unauthorized route before loading plugin HTML", async ({ page }) => {
	let pageRequests = 0;
	await prepareRuntime(page, "", () => {
		pageRequests += 1;
		return "success";
	});

	await page.goto(PLUGIN_PATH);
	await expect(page).toHaveURL(/\/skoll\/forbidden/);
	await expect(page.locator(".state-block--forbidden")).toBeVisible();
	expect(pageRequests).toBe(0);
});

test("plugin runtime stays inside route, interaction, long-task, and memory budgets", async ({ page }) => {
	await installLongTaskObserver(page);
	await prepareRuntime(page, "runtime.access", () => "success");
	await page.goto(PLUGIN_PATH);
	await expect(page.frameLocator("iframe.plugin-page-frame").getByRole("heading", { name: "插件运行正常" })).toBeVisible();
	await page.goto("/skoll/dashboard");
	await readAndResetLongTasks(page);

	const routeStarted = performance.now();
	await navigateThroughShell(page, PLUGIN_PATH);
	const frame = page.frameLocator("iframe.plugin-page-frame");
	await expect(frame.getByRole("heading", { name: "插件运行正常" })).toBeVisible();
	const routeReadyMs = performance.now() - routeStarted;
	const longTasks: number[] = await readAndResetLongTasks(page);

	const interactionStarted = performance.now();
	await frame.getByRole("button", { name: "执行操作" }).click();
	await expect(frame.locator("#result")).toHaveText("1");
	const interactionMs = performance.now() - interactionStarted;
	longTasks.push(...await readAndResetLongTasks(page));

	const session = await page.context().newCDPSession(page);
	await session.send("Performance.enable");
	await session.send("HeapProfiler.collectGarbage");
	const heapBefore = await readHeap(session);
	for (let cycle = 0; cycle < 3; cycle += 1) {
		await navigateThroughShell(page, "/skoll/dashboard");
		await navigateThroughShell(page, PLUGIN_PATH);
		await expect(page.frameLocator("iframe.plugin-page-frame").getByRole("heading", { name: "插件运行正常" })).toBeVisible();
		longTasks.push(...await readAndResetLongTasks(page));
	}
	await session.send("HeapProfiler.collectGarbage");
	const heapGrowthBytes = Math.max(0, (await readHeap(session)) - heapBefore);

	expect(routeReadyMs, "plugin route readiness").toBeLessThanOrEqual(budgets.routeReadyMs);
	expect(interactionMs, "plugin interaction readiness").toBeLessThanOrEqual(budgets.interactionMs);
	expect(longTasks.length === 0 ? 0 : Math.max(...longTasks), "maximum plugin runtime long task").toBeLessThanOrEqual(budgets.maxLongTaskMs);
	expect(longTasks.reduce((total, value) => total + value, 0), "total plugin runtime long tasks").toBeLessThanOrEqual(budgets.totalLongTaskMs);
	expect(heapGrowthBytes, "plugin runtime route-cycle heap growth").toBeLessThanOrEqual(budgets.heapGrowthBytes);
});

async function prepareRuntime(page: Page, permission: string, resolveMode: () => PageMode): Promise<void> {
	const permissions = permission ? [permission] : [];
	await page.route("**/skoll/v1/auth/login", (route) => route.fulfill({
		status: 200,
		contentType: "application/json",
		body: JSON.stringify({
			code: "ok",
			message: "ok",
			data: {
				token: "runtime-token",
				permissions,
				user: {
					id: "runtime-user",
					account: "runtime-user",
					name: "Runtime User",
					role: "user",
					roles: ["user"],
					organizationId: "org-runtime",
					organizationPath: ["org-runtime"]
				}
			}
		})
	}));
	await page.route("**/skoll/v1/auth/me", (route) => route.fulfill({
		status: 200,
		contentType: "application/json",
		body: JSON.stringify({
			code: "ok",
			message: "ok",
			data: {
				id: "runtime-user",
				name: "Runtime User",
				role: "user",
				roles: ["user"],
				permissions,
				organizationId: "org-runtime",
				organizationPath: ["org-runtime"]
			}
		})
	}));
	await page.route("**/skoll/v1/plugins", (route) => route.fulfill({
		status: 200,
		contentType: "application/json",
		body: JSON.stringify({
			code: "ok",
			message: "ok",
			data: [{
				id: PLUGIN_ID,
				name: "Runtime Probe",
				version: "1.0.0",
				enabled: true,
				uiMode: "frontend_only",
				level: "system",
				uiNavPosition: "sidebar",
				uiOpenMode: "integrated",
				frontendEntry: `/plugins/${PLUGIN_ID}`,
				i18nLocales: ["zh-CN", "en-US"],
				uiMenu: {
					labelZhCN: "运行探针",
					labelEnUS: "Runtime Probe",
					path: `/plugins/${PLUGIN_ID}`,
					requiredPermissions: ["runtime.access"]
				}
			}]
		})
	}));
	await page.route(`**/skoll/v1/plugins/${PLUGIN_ID}/page*`, async (route) => {
		const mode = resolveMode();
		if (mode === "error") {
			await route.fulfill({ status: 503, body: "unavailable" });
			return;
		}
		if (mode === "oversized") {
			await route.fulfill({
				status: 200,
				contentType: "text/html",
				headers: { "Content-Length": "1048577" },
				body: pluginHTML
			});
			return;
		}
		if (mode === "slow") {
			await new Promise((resolve) => setTimeout(resolve, 5_200));
		}
		await route.fulfill({ status: 200, contentType: "text/html", body: pluginHTML }).catch(() => undefined);
	});

	await page.goto("/skoll/login");
	await page.evaluate(() => localStorage.clear());
	await page.reload();
	await page.locator("input[autocomplete='username']:visible").fill("runtime-user");
	await page.locator("input[autocomplete='current-password']:visible").fill("Runtime@123");
	await page.locator("button[type='submit']:visible").click();
	await expect(page).not.toHaveURL(/\/login/);
}

async function installLongTaskObserver(page: Page): Promise<void> {
	await page.addInitScript(() => {
		const target = window as Window & { __skollRuntimeLongTasks?: number[] };
		target.__skollRuntimeLongTasks = [];
		try {
			new PerformanceObserver((entries) => {
				target.__skollRuntimeLongTasks?.push(...entries.getEntries().map((entry) => entry.duration));
			}).observe({ type: "longtask", buffered: true });
		} catch {
			target.__skollRuntimeLongTasks = [];
		}
	});
}

async function navigateThroughShell(page: Page, path: string): Promise<void> {
	const href = path.startsWith("/skoll/plugins/") ? path.slice("/skoll".length) : path;
	await page.locator(`a[href='${href}']`).first().evaluate((link: HTMLAnchorElement) => link.click());
	await expect(page).toHaveURL(new RegExp(`${path.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}$`));
}

async function readAndResetLongTasks(page: Page): Promise<number[]> {
	return page.evaluate(() => {
		const target = window as Window & { __skollRuntimeLongTasks?: number[] };
		const entries = [...(target.__skollRuntimeLongTasks || [])];
		target.__skollRuntimeLongTasks = [];
		return entries;
	});
}

async function readHeap(session: CDPSession): Promise<number> {
	const response = await session.send("Performance.getMetrics");
	return response.metrics.find((metric) => metric.name === "JSHeapUsedSize")?.value || 0;
}
