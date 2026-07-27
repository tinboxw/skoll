import { createServer as createNetServer } from "node:net";
import { resolve } from "node:path";
import process from "node:process";

import { chromium } from "playwright";
import { preview } from "vite";

const pluginWebRoot = resolve(process.argv[2] || "");
const pluginId = String(process.argv[3] || "").trim();
if (!pluginWebRoot || !pluginId) {
	throw new Error("usage: node ff4-generated-plugin-browser.mjs <plugin-web-root> <plugin-id>");
}

const port = await reservePort();
const server = await preview({
	root: pluginWebRoot,
	configFile: resolve(pluginWebRoot, "vite.config.ts"),
	preview: { host: "127.0.0.1", port, strictPort: true }
});
const baseURL = `http://127.0.0.1:${port}`;
const browser = await chromium.launch({
	channel: process.env.SKOLL_E2E_BROWSER_CHANNEL || "chrome",
	headless: true
});
const report = [];

try {
	await fetch(baseURL);
	for (const target of [
		{ name: "desktop", viewport: { width: 1440, height: 900 } },
		{ name: "mobile", viewport: { width: 390, height: 844 } }
	]) {
		const context = await browser.newContext({ viewport: target.viewport });
		await context.addInitScript(pluginHostFixture, {
			pluginId,
			permissions: ["pharma_oa.product.read", "pharma_oa.product.create", "pharma_oa.product.update", "pharma_oa.product.delete"]
		});
		const warmup = await context.newPage();
		await warmup.goto(baseURL, { waitUntil: "domcontentloaded" });
		await warmup.locator(':text-is("Aspirin"):visible').first().waitFor({ state: "visible" });
		if (target.name === "desktop") {
			await warmup.getByRole("button", { name: "新建" }).click();
			await warmup.locator(".el-drawer.open").waitFor({ state: "visible" });
		}
		await warmup.close();
		const page = await context.newPage();
		await page.addInitScript(() => {
			window.__FF4_LONG_TASKS__ = [];
			new PerformanceObserver((list) => {
				for (const entry of list.getEntries()) window.__FF4_LONG_TASKS__.push(entry.duration);
			}).observe({ type: "longtask", buffered: true });
		});
		const startedAt = Date.now();
		await page.goto(baseURL, { waitUntil: "domcontentloaded" });
		await page.locator(".plugin-generated-page").waitFor({ state: "visible" });
		await page.locator(':text-is("Aspirin"):visible').first().waitFor({ state: "visible" });
		const readyMs = Date.now() - startedAt;

		assert(readyMs <= 2500, `${target.name}: route ready ${readyMs}ms exceeds 2500ms`);
		assert(await page.locator("html").getAttribute("lang") === "zh-CN", `${target.name}: initial locale was not applied`);
		assert(await page.locator("html").getAttribute("data-theme") === "light", `${target.name}: initial theme was not applied`);
		await assertLayout(page, target.name);
		const initialLongTasks = await page.evaluate(() => {
			const values = window.__FF4_LONG_TASKS__;
			window.__FF4_LONG_TASKS__ = [];
			return values;
		});
		const initialMaxLongTaskMs = Math.max(0, ...initialLongTasks);
		assert(initialMaxLongTaskMs <= 1000, `${target.name}: initial long task ${initialMaxLongTaskMs}ms exceeds 1000ms`);

		if (target.name === "desktop") {
			await page.getByRole("button", { name: "新建" }).click();
			await page.locator(".el-drawer.open").waitFor({ state: "visible" });
			await page.keyboard.press("Escape");
			await page.evaluate(() => window.__FF4_SET_HOST__({ locale: "en-US", colorScheme: "dark" }));
			await page.getByRole("button", { name: "Search", exact: true }).waitFor({ state: "visible" });
			assert(await page.locator("html").getAttribute("lang") === "en-US", "desktop: locale update was not applied");
			assert(await page.locator("html").getAttribute("data-theme") === "dark", "desktop: theme update was not applied");
			await page.evaluate(() => window.__FF4_SET_HOST__({ lifecycle: "degraded" }));
			await page.locator(".business-state--conflict").waitFor({ state: "visible" });
			await page.evaluate(() => window.__FF4_SET_HOST__({ lifecycle: "enabled" }));
			await page.locator(':text-is("Aspirin"):visible').first().waitFor({ state: "visible" });
		} else {
			await page.locator(".business-list__mobile").waitFor({ state: "visible" });
		}

		const interactionLongTasks = await page.evaluate(() => window.__FF4_LONG_TASKS__);
		const interactionMaxLongTaskMs = Math.max(0, ...interactionLongTasks);
		assert(interactionMaxLongTaskMs <= 200, `${target.name}: interaction long task ${interactionMaxLongTaskMs}ms exceeds 200ms`);
		report.push({ target: target.name, readyMs, initialMaxLongTaskMs, interactionMaxLongTaskMs });
		await context.close();
	}

	const restricted = await browser.newContext({ viewport: { width: 1440, height: 900 } });
	await restricted.addInitScript(pluginHostFixture, { pluginId, permissions: [] });
	const restrictedPage = await restricted.newPage();
	await restrictedPage.goto(baseURL, { waitUntil: "domcontentloaded" });
	await restrictedPage.locator(".business-state--forbidden").waitFor({ state: "visible" });
	const requestCount = await restrictedPage.evaluate(() => window.__FF4_REQUESTS__);
	assert(requestCount === 0, `restricted role loaded business data ${requestCount} time(s)`);
	report.push({ target: "restricted", requestCount });
	await restricted.close();
} finally {
	await browser.close();
	await server.httpServer.close();
}

console.log(JSON.stringify({ pluginId, report }));

async function reservePort() {
	const socket = createNetServer();
	await new Promise((resolveReady, reject) => {
		socket.once("error", reject);
		socket.listen(0, "127.0.0.1", resolveReady);
	});
	const address = socket.address();
	const selected = typeof address === "object" && address ? address.port : 0;
	await new Promise((resolveClosed) => socket.close(resolveClosed));
	if (!selected) throw new Error("failed to reserve generated plugin preview port");
	return selected;
}

async function assertLayout(page, target) {
	const issues = await page.evaluate(() => {
		const visible = (element) => {
			const style = getComputedStyle(element);
			const rect = element.getBoundingClientRect();
			return style.display !== "none" && style.visibility !== "hidden" && rect.width > 0 && rect.height > 0;
		};
		const unlabeledButtons = [...document.querySelectorAll("button")]
			.filter(visible)
			.filter((button) => !(button.textContent || "").trim() && !button.getAttribute("aria-label") && !button.getAttribute("title"))
			.length;
		const controlOverflow = [...document.querySelectorAll("button,input,textarea,select,[role='button'],[role='combobox']")]
			.filter(visible)
			.filter((element) => {
				const rect = element.getBoundingClientRect();
				return rect.left < -1 || rect.right > window.innerWidth + 1;
			})
			.map((element) => {
				const rect = element.getBoundingClientRect();
				return { label: (element.textContent || element.getAttribute("aria-label") || element.tagName).trim().slice(0, 80), left: rect.left, right: rect.right };
			});
		return {
			documentOverflow: Math.max(0, document.documentElement.scrollWidth - document.documentElement.clientWidth),
			unlabeledButtons,
			controlOverflow
		};
	});
	assert(issues.documentOverflow <= 1, `${target}: document overflows by ${issues.documentOverflow}px`);
	assert(issues.controlOverflow.length === 0, `${target}: controls leave the viewport ${JSON.stringify(issues.controlOverflow)}`);
	assert(issues.unlabeledButtons === 0, `${target}: ${issues.unlabeledButtons} visible button(s) are unlabeled`);
}

function assert(condition, message) {
	if (!condition) throw new Error(message);
}

function pluginHostFixture({ pluginId, permissions }) {
	const capabilities = [
		"request", "navigation", "commands", "permissions", "locale", "theme", "lifecycle", "auth", "user",
		"organization", "dictionary", "file", "audit", "config"
	];
	const tokens = {
		"--color-bg": "#f4f7f9",
		"--color-surface": "#ffffff",
		"--color-surface-soft": "#f8fafb",
		"--color-border": "#d8e0e5",
		"--color-border-strong": "#aebbc4",
		"--color-text": "#172126",
		"--color-text-muted": "#5d6b73",
		"--color-primary": "#176b5b",
		"--color-primary-strong": "#0f574a",
		"--color-on-primary": "#ffffff",
		"--color-tag-bg": "#e5f3ef",
		"--color-tag-text": "#155f51",
		"--color-warning": "#b7791f",
		"--color-warning-soft": "#fff8e6",
		"--color-warning-text": "#7a4d0b",
		"--color-warning-border": "#e3bd73",
		"--color-danger": "#c2413b",
		"--color-success": "#23845e",
		"--color-success-soft": "#e7f7ef",
		"--color-success-text": "#176244",
		"--color-success-border": "#86c9a8",
		"--color-info": "#3f6f8f",
		"--color-shadow": "rgba(23, 33, 38, 0.14)",
		"--color-code-surface": "#eef2f4",
		"--color-loading-surface": "#edf2f4",
		"--radius-sm": "4px",
		"--radius-md": "6px",
		"--radius-lg": "8px",
		"--layout-gap": "16px",
		"--content-padding": "20px",
		"--control-height": "32px",
		"--font-size-base": "14px"
	};
	const ctx = {
		locale: "zh-CN",
		locales: ["zh-CN", "en-US"],
		theme: { colorScheme: "light", density: "comfortable", tokens },
		lifecycle: { state: "enabled", health: "healthy" }
	};
	const identity = { subject: "ff4-user", roles: ["tester"], permissions };
	const applyTheme = () => {
		const root = document.documentElement;
		if (!root) return;
		root.dataset.theme = ctx.theme.colorScheme;
		root.dataset.density = ctx.theme.density;
		root.style.colorScheme = ctx.theme.colorScheme;
		for (const [name, value] of Object.entries(ctx.theme.tokens)) root.style.setProperty(name, value);
	};
	const response = (path, options = {}) => {
		window.__FF4_REQUESTS__ += 1;
		const method = options.method || "GET";
		const item = { id: "product-1", name: "Aspirin" };
		if (method === "GET" && /\/products\/[^/?]+/.test(path)) return { code: "ok", message: "", data: { item } };
		if (method === "GET") return { code: "ok", message: "", data: { items: [item], offset: 0, limit: 20 } };
		if (method === "DELETE") return { code: "ok", message: "", data: null };
		return { code: "ok", message: "", data: { item: { ...item, ...(options.body || {}) } } };
	};
	const host = {
		contract: "skoll.plugin-host",
		version: "1.0",
		pluginId,
		pluginVersion: "1.0.0",
		capabilities,
		identity,
		get locale() { return ctx.locale; },
		get locales() { return ctx.locales; },
		get theme() { return ctx.theme; },
		get lifecycle() { return ctx.lifecycle; },
		hasCapability: (capability) => capabilities.includes(capability),
		requireCapability: (capability) => { if (!capabilities.includes(capability)) throw new Error("capability missing"); },
		request: async (path, options) => response(path, options),
		navigation: { push() {}, replace() {}, back() {} },
		commands: { execute() {} },
		permissions: {
			has: (permission) => permissions.includes("*") || permissions.includes(permission),
			require: (permission) => { if (!permissions.includes("*") && !permissions.includes(permission)) throw new Error("permission denied"); }
		},
		auth: { me: async () => identity },
		user: { me: async () => identity },
		organization: { departments: async () => [], positions: async () => [] },
		dictionary: { list: async () => [], items: async () => [] },
		file: { list: async () => [] },
		audit: { list: async () => [] },
		config: { get: async () => ({}), update: async (value) => value }
	};
	window.__FF4_REQUESTS__ = 0;
	Object.defineProperty(window, "__SKOLL_HOST__", { value: host, configurable: false, writable: false });
	if (document.documentElement) document.documentElement.lang = ctx.locale;
	applyTheme();
	window.__FF4_SET_HOST__ = (change) => {
		if (change.locale) {
			ctx.locale = change.locale;
			if (document.documentElement) document.documentElement.lang = ctx.locale;
			window.dispatchEvent(new CustomEvent("skoll:locale", { detail: { locale: ctx.locale, locales: ctx.locales } }));
		}
		if (change.colorScheme) {
			ctx.theme = { ...ctx.theme, colorScheme: change.colorScheme };
			applyTheme();
			window.dispatchEvent(new CustomEvent("skoll:theme", { detail: ctx.theme }));
		}
		if (change.lifecycle) {
			ctx.lifecycle = { state: change.lifecycle, health: change.lifecycle === "enabled" ? "healthy" : change.lifecycle };
			window.dispatchEvent(new CustomEvent("skoll:lifecycle", { detail: ctx.lifecycle }));
		}
	};
}
