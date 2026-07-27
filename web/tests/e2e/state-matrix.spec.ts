import { expect, test, type Page, type TestInfo } from "@playwright/test";

type Locale = "zh-CN" | "en-US";
type Theme = "light" | "dark";

const locales: Locale[] = ["zh-CN", "en-US"];
const themes: Theme[] = ["light", "dark"];
const surfaces = [
	{ name: "plugin-center", path: "/skoll/plugin-center" },
	{ name: "form-builder", path: "/skoll/form-builder" },
	{ name: "workflow", path: "/skoll/workflow" },
	{ name: "todo-center", path: "/skoll/todo" }
];

test.describe.configure({ mode: "serial" });

for (const locale of locales) {
	for (const theme of themes) {
		test(`${locale} ${theme} platform surface matrix`, async ({ page, request }, testInfo) => {
			const health = await request.get("/skoll/health");
			expect(health.ok()).toBeTruthy();

			await login(page, locale, theme);
			await expect(page.locator("html")).toHaveAttribute("lang", locale);
			await expect(page.locator("html")).toHaveAttribute("data-theme", theme);

			for (const surface of surfaces) {
				await page.goto(surface.path);
				const shell = page.locator(".page-shell").first();
				await expect(shell).toBeVisible();
				await expect(shell).toHaveAttribute("aria-busy", "false");
				await assertLayout(page, `${locale}-${theme}-${surface.name}`);
				await attachScreenshot(page, testInfo, `${locale}-${theme}-${surface.name}`);
			}

			await page.goto("/skoll/forbidden");
			await expect(page.locator(".state-block--forbidden")).toBeVisible();
			await assertLayout(page, `${locale}-${theme}-forbidden`, false);
		});
	}
}

async function login(page: Page, locale: Locale, theme: Theme): Promise<void> {
	await page.goto("/skoll/login");
	await page.evaluate(({ nextLocale, nextTheme }) => {
		localStorage.clear();
		localStorage.setItem("skoll.ui.locale", nextLocale);
		localStorage.setItem("skoll.ui.colorScheme", nextTheme);
	}, { nextLocale: locale, nextTheme: theme });
	await page.reload();
	await page.locator("input[autocomplete='username']:visible").fill("admin");
	await page.locator("input[autocomplete='current-password']:visible").fill("Admin@123456");
	await page.locator("button[type='submit']:visible").click();
	await expect(page).not.toHaveURL(/\/login/);
}

async function attachScreenshot(page: Page, testInfo: TestInfo, name: string): Promise<void> {
	const screenshot = await page.screenshot({ fullPage: true });
	await testInfo.attach(name, { body: screenshot, contentType: "image/png" });
}

async function assertLayout(page: Page, state: string, requirePageShell = true): Promise<void> {
	const first = await collectLayout(page);
	await page.waitForTimeout(180);
	const second = await collectLayout(page);

	expect(first.pageOverflow, `${state}: document horizontal overflow`).toBeLessThanOrEqual(1);
	expect(first.textOverflows, `${state}: visible text overflow`).toEqual([]);
	expect(first.controlOverflows, `${state}: controls leave the viewport`).toEqual([]);
	expect(first.regionOverlap, `${state}: page header overlaps body`).toBe(false);
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
		const landmarks: Record<string, Rect> = {};
		for (const selector of [".page-shell", ".page-shell__header", ".page-shell__body", ".page-shell > .state-block"]) {
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
