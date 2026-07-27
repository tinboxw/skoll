import { expect, test } from "@playwright/test";

const FIXTURE = "/tests/fixtures/business-ui.html";
const combinations = [
	["light", "comfortable", "zh-CN"],
	["dark", "comfortable", "en-US"],
	["light", "compact", "en-US"],
	["dark", "compact", "zh-CN"]
] as const;

for (const [theme, density, locale] of combinations) {
	test(`${theme} ${density} ${locale} business workspace inherits presentation`, async ({ page }, testInfo) => {
		await page.emulateMedia({ reducedMotion: "reduce" });
		await page.goto(`${FIXTURE}?theme=${theme}&density=${density}&locale=${locale}`);
		await expect(page.locator(".business-workspace")).toBeVisible();
		await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
		await expect(page.locator("html")).toHaveAttribute("data-density", density);
		await expect(page.locator("html")).toHaveAttribute("lang", locale);
		await expect(page.getByRole("toolbar")).toBeVisible();
		await expect(page.getByRole("status")).toContainText(locale === "en-US" ? "Completed" : "操作成功");

		const presentation = await page.evaluate(() => {
			const root = getComputedStyle(document.documentElement);
			const button = getComputedStyle(document.querySelector(".el-button") as Element);
			return {
				primary: root.getPropertyValue("--color-primary").trim(),
				controlHeight: root.getPropertyValue("--control-height").trim(),
				buttonHeight: button.height,
				reducedMotion: matchMedia("(prefers-reduced-motion: reduce)").matches
			};
		});
		expect(presentation.primary).not.toBe("");
		expect(presentation.controlHeight).toBe(density === "compact" ? "32px" : "36px");
		expect(Number.parseFloat(presentation.buttonHeight)).toBeGreaterThanOrEqual(density === "compact" ? 30 : 34);
		expect(presentation.reducedMotion).toBe(true);

		await page.keyboard.press("Tab");
		const focused = page.locator(":focus");
		await expect(focused).toBeVisible();
		expect(await focused.evaluate((element) => getComputedStyle(element).outlineStyle)).not.toBe("none");

		await page.getByRole("button", { name: locale === "en-US" ? "Delete" : "删除" }).click();
		await expect(page.getByRole("button", { name: locale === "en-US" ? "Confirm" : "确认", exact: true })).toBeVisible();
		await assertNoViewportOverflow(page);
		await testInfo.attach(`${theme}-${density}-${locale}`, {
			body: await page.screenshot({ fullPage: true }),
			contentType: "image/png"
		});
	});
}

async function assertNoViewportOverflow(page: import("@playwright/test").Page): Promise<void> {
	const result = await page.evaluate(() => ({
		documentWidth: document.documentElement.scrollWidth,
		viewportWidth: document.documentElement.clientWidth,
		outsideControls: Array.from(document.querySelectorAll("button, input, .el-select")).filter((element) => {
			const rect = element.getBoundingClientRect();
			return rect.width > 0 && (rect.left < -1 || rect.right > document.documentElement.clientWidth + 1);
		}).length
	}));
	expect(result.documentWidth).toBeLessThanOrEqual(result.viewportWidth + 1);
	expect(result.outsideControls).toBe(0);
}
