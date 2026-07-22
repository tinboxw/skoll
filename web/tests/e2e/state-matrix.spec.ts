import { expect, test, type Page, type Route, type TestInfo } from "@playwright/test";

const CUSTOMER_PATH = "/skoll/pharma-oa/customers";
const DASHBOARD_PATH = "/skoll/pharma-oa/dashboard";
const QUALIFICATION_PATH = "/skoll/pharma-oa/qualifications";
const CUSTOMER_API = "**/skoll/v1/plugins/pharma_oa/api/customers**";
const EXPIRY_SCAN_API = "**/skoll/v1/plugins/pharma_oa/api/qualifications/expiry-scan**";

type Locale = "zh-CN" | "en-US";

type LocaleLabels = {
	login: string;
	synced: string;
	refresh: string;
	expiryScan: string;
	runExpiryScan: string;
	networkError: string;
};

const LOCALES: Record<Locale, LocaleLabels> = {
	"zh-CN": {
		login: "登录",
		synced: "已同步",
		refresh: "刷新",
		expiryScan: "到期扫描",
		runExpiryScan: "执行资质到期扫描",
		networkError: "网络连接失败，请检查后端服务或网络设置。"
	},
	"en-US": {
		login: "Sign In",
		synced: "Synced",
		refresh: "Refresh",
		expiryScan: "Expiry scan",
		runExpiryScan: "Run qualification expiry scan",
		networkError: "Network connection failed. Please check backend service and network settings."
	}
};

test.describe.configure({ mode: "serial" });

for (const locale of Object.keys(LOCALES) as Locale[]) {
	test(`${locale} responsive state matrix`, async ({ page, request }, testInfo) => {
		const health = await request.get("/skoll/health");
		expect(health.ok()).toBeTruthy();

		await login(page, locale, "admin", "Admin@123456");

		await test.step("primary responsive surfaces", async () => {
			const surfaces = [
				{ name: "plugin-workspace", path: "/skoll/plugin-center", selector: ".page-shell", pageShell: true },
				{ name: "form-builder", path: "/skoll/form-builder", selector: ".page-shell", pageShell: true },
				{ name: "developer-portal", path: "/skoll/plugins/developer-portal", selector: ".remote-plugin-fullpage iframe", pageShell: false }
			];
			for (const surface of surfaces) {
				await page.goto(surface.path);
				await expect(page.locator(surface.selector).first()).toBeVisible();
				if (surface.pageShell) {
					await expect(page.locator(".page-shell")).toHaveAttribute("aria-busy", "false");
				}
				await assertLayout(page, `surface-${surface.name}`, surface.pageShell);
				const screenshot = await page.screenshot({ fullPage: true });
				await testInfo.attach(`${locale}-surface-${surface.name}`, { body: screenshot, contentType: "image/png" });
			}
		});

		await test.step("responsive", async () => {
			await openPage(page, DASHBOARD_PATH);
			await expect(page.locator(".page-shell")).toHaveAttribute("aria-busy", "false");
			await captureState(page, testInfo, locale, "responsive", ".page-shell__body");
		});

		await test.step("loading", async () => {
			const loadingHandler = async (route: Route) => {
				await new Promise((resolve) => setTimeout(resolve, 2_000));
				await route.continue();
			};
			await page.route(CUSTOMER_API, loadingHandler);
			await page.goto(CUSTOMER_PATH);
			await expect(page.locator(".page-shell__loading")).toBeVisible();
			await captureState(page, testInfo, locale, "loading", ".page-shell__loading");
			await expect(page.locator(".page-shell")).toHaveAttribute("aria-busy", "false");
			await page.unroute(CUSTOMER_API, loadingHandler);
		});

		await test.step("empty", async () => {
			const keyword = page.locator(".filter-bar input").filter({ visible: true }).first();
			await keyword.fill("__pr4_state_empty__");
			await expect(page.locator(".data-table .state-block--empty")).toBeVisible();
			await captureState(page, testInfo, locale, "empty", ".data-table .state-block--empty");
		});

		await test.step("error", async () => {
			const errorHandler = async (route: Route) => route.abort("failed");
			await page.route(CUSTOMER_API, errorHandler);
			await page.locator(".page-shell__actions").getByRole("button", { name: LOCALES[locale].refresh, exact: true }).click();
			await expect(page.locator(".page-shell .state-block--error")).toBeVisible();
			await captureState(page, testInfo, locale, "error", ".page-shell .state-block--error");
			await page.unroute(CUSTOMER_API, errorHandler);
			await page.getByRole("button", { name: /retry|重试/i }).click();
			await expect(page.locator(".page-shell__body")).toBeVisible();
		});

		await test.step("offline", async () => {
			await page.context().setOffline(true);
			await page.locator(".page-shell__actions").getByRole("button", { name: LOCALES[locale].refresh, exact: true }).click();
			const offlineState = page.locator(".page-shell .state-block--error");
			await expect(offlineState).toContainText(LOCALES[locale].networkError);
			await captureState(page, testInfo, locale, "offline", ".page-shell .state-block--error");
			await page.context().setOffline(false);
			await page.getByRole("button", { name: /retry|重试/i }).click();
			await expect(page.locator(".page-shell__body")).toBeVisible();
		});

		await test.step("destructive", async () => {
			await openPage(page, QUALIFICATION_PATH);
			await page.getByRole("button", { name: LOCALES[locale].expiryScan, exact: true }).click();
			await expect(page.locator(".el-message-box")).toBeVisible();
			await expect(page.locator(".el-message-box")).toContainText(LOCALES[locale].runExpiryScan);
			await captureState(page, testInfo, locale, "destructive", ".el-message-box");
		});

		await test.step("saving", async () => {
			const savingHandler = async (route: Route) => {
				await new Promise((resolve) => setTimeout(resolve, 2_000));
				await route.continue();
			};
			await page.route(EXPIRY_SCAN_API, savingHandler);
			await page.getByRole("button", { name: LOCALES[locale].runExpiryScan, exact: true }).click();
			await expect(page.locator(".page-shell__actions button.is-loading")).toBeVisible();
			await captureState(page, testInfo, locale, "saving", ".page-shell__actions button.is-loading");
			await expect(page.locator(".el-message--success")).toBeVisible();
			await page.unroute(EXPIRY_SCAN_API, savingHandler);
		});

		await test.step("success", async () => {
			await captureState(page, testInfo, locale, "success", ".el-message--success");
		});

		await test.step("no_permission", async () => {
			await login(page, locale, "dept_admin", "Dept@123456");
			await page.goto(CUSTOMER_PATH);
			await expect(page).toHaveURL(/\/skoll\/forbidden/);
			await expect(page.locator(".state-block--forbidden")).toBeVisible();
			await captureState(page, testInfo, locale, "no_permission", ".state-block--forbidden");
		});
	});
}

async function login(page: Page, locale: Locale, account: string, password: string): Promise<void> {
	await page.goto("/skoll/login");
	await page.evaluate((nextLocale) => {
		localStorage.clear();
		localStorage.setItem("skoll.ui.locale", nextLocale);
	}, locale);
	await page.reload();
	await page.locator("input[autocomplete='username']:visible").fill(account);
	await page.locator("input[autocomplete='current-password']:visible").fill(password);
	await page.getByRole("button", { name: LOCALES[locale].login, exact: true }).click();
	await expect(page).not.toHaveURL(/\/login/);
	await expect(page.locator(".header__status")).toHaveAttribute("aria-label", LOCALES[locale].synced);
}

async function openPage(page: Page, path: string): Promise<void> {
	await page.goto(path);
	await expect(page.locator(".page-shell")).toBeVisible();
	await expect(page.locator(".page-shell")).toHaveAttribute("aria-busy", "false");
}

async function captureState(page: Page, testInfo: TestInfo, locale: Locale, state: string, selector: string): Promise<void> {
	await expect(page.locator(selector).first()).toBeVisible();
	await assertLayout(page, state);
	const screenshot = await page.screenshot({ fullPage: true });
	await testInfo.attach(`${locale}-${state}`, { body: screenshot, contentType: "image/png" });
}

async function assertLayout(page: Page, state: string, requirePageShell = true): Promise<void> {
	const first = await collectLayout(page);
	await page.waitForTimeout(180);
	const second = await collectLayout(page);

	expect(first.pageOverflow, `${state}: document horizontal overflow`).toBeLessThanOrEqual(1);
	expect(first.textOverflows, `${state}: visible text overflow`).toEqual([]);
	expect(first.controlOverflows, `${state}: controls leave the viewport`).toEqual([]);
	expect(first.regionOverlap, `${state}: page header overlaps state/body region`).toBe(false);
	expect(first.lang, `${state}: document language`).toMatch(/^(zh-CN|en-US)$/);
	if (requirePageShell) {
		expect(first.pageShellLabelled, `${state}: page shell heading association`).toBe(true);
	}
	expect(stabilityDelta(first.landmarks, second.landmarks), `${state}: layout moved after stabilization`).toBeLessThanOrEqual(1);
}

type Rect = { x: number; y: number; width: number; height: number };

type LayoutSnapshot = {
	pageOverflow: number;
	textOverflows: string[];
	controlOverflows: string[];
	regionOverlap: boolean;
	lang: string;
	pageShellLabelled: boolean;
	landmarks: Record<string, Rect>;
};

async function collectLayout(page: Page): Promise<LayoutSnapshot> {
	return page.evaluate(() => {
		const visible = (element: Element): element is HTMLElement => {
			const target = element as HTMLElement;
			const style = getComputedStyle(target);
			const rect = target.getBoundingClientRect();
			return style.display !== "none" && style.visibility !== "hidden" && rect.width > 0 && rect.height > 0;
		};
		const toRect = (element: Element): Rect => {
			const rect = element.getBoundingClientRect();
			return { x: rect.x, y: rect.y, width: rect.width, height: rect.height };
		};
		const label = (element: Element) => (element.textContent || element.getAttribute("aria-label") || element.tagName).trim().slice(0, 80);
		const textOverflows = [...document.querySelectorAll("button,.el-tag,h1,h2,h3,.state-block p,.page-shell__description")]
			.filter(visible)
			.filter((element) => element.scrollWidth > element.clientWidth + 2 || element.scrollHeight > element.clientHeight + 2)
			.map(label);
		const controlOverflows = [...document.querySelectorAll("button,input,textarea,select,[role='button'],[role='combobox']")]
			.filter(visible)
			.filter((element) => {
				const rect = element.getBoundingClientRect();
				return rect.left < -1 || rect.right > window.innerWidth + 1;
			})
			.map(label);
		const header = document.querySelector(".page-shell__header");
		const content = document.querySelector(".page-shell__body,.page-shell > .state-block,.page-shell__loading");
		const headerRect = header && visible(header) ? header.getBoundingClientRect() : null;
		const contentRect = content && visible(content) ? content.getBoundingClientRect() : null;
		const shell = document.querySelector(".page-shell");
		const titleId = shell?.getAttribute("aria-labelledby") || "";
		const landmarkSelectors = [".page-shell", ".page-shell__header", ".page-shell__body", ".page-shell > .state-block", ".page-shell__loading", ".el-message-box", ".el-message--success"];
		const landmarks: Record<string, Rect> = {};
		for (const selector of landmarkSelectors) {
			const element = document.querySelector(selector);
			if (element && visible(element)) landmarks[selector] = toRect(element);
		}
		return {
			pageOverflow: Math.max(0, document.documentElement.scrollWidth - document.documentElement.clientWidth),
			textOverflows,
			controlOverflows,
			regionOverlap: Boolean(headerRect && contentRect && contentRect.top < headerRect.bottom - 1),
			lang: document.documentElement.lang,
			pageShellLabelled: Boolean(titleId && document.getElementById(titleId)?.textContent?.trim()),
			landmarks
		};
	});
}

function stabilityDelta(before: Record<string, Rect>, after: Record<string, Rect>): number {
	let maximum = 0;
	for (const [selector, first] of Object.entries(before)) {
		const second = after[selector];
		if (!second) continue;
		maximum = Math.max(
			maximum,
			Math.abs(first.x - second.x),
			Math.abs(first.y - second.y),
			Math.abs(first.width - second.width),
			Math.abs(first.height - second.height)
		);
	}
	return maximum;
}
