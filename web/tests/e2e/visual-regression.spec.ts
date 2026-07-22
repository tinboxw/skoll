import { expect, test, type Page } from "@playwright/test";

test.describe.configure({ mode: "serial" });

test("critical surfaces preserve visual hierarchy", async ({ page, request }) => {
	const health = await request.get("/skoll/health");
	expect(health.ok()).toBeTruthy();

	await prepareVisualEnvironment(page, "light");
	await page.goto("/skoll/login");
	await expect(page.locator("input[autocomplete='username']")).toBeVisible();
	await expect(page).toHaveScreenshot("login-light.png", { fullPage: true });

	await login(page);
	await page.goto("/skoll/pharma-oa/dashboard");
	await expect(page.locator(".page-shell")).toHaveAttribute("aria-busy", "false");
	await expect(page).toHaveScreenshot("dashboard-light.png", {
		fullPage: true,
		mask: [page.locator(".updated"), page.locator(".el-date-editor")]
	});

	await page.evaluate(() => {
		localStorage.setItem("skoll.ui.colorScheme", "dark");
	});
	await page.reload();
	await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
	await page.goto("/skoll/pharma-oa/customers");
	await expect(page.locator(".page-shell")).toHaveAttribute("aria-busy", "false");
	await expect(page).toHaveScreenshot("customers-dark.png", { fullPage: true });
});

async function prepareVisualEnvironment(page: Page, scheme: "light" | "dark"): Promise<void> {
	await page.goto("/skoll/login");
	await page.evaluate((colorScheme) => {
		localStorage.clear();
		localStorage.setItem("skoll.ui.locale", "en-US");
		localStorage.setItem("skoll.ui.colorScheme", colorScheme);
		localStorage.setItem("skoll.ui.density", "comfortable");
	}, scheme);
	await page.reload();
	await page.addStyleTag({ content: "*,*::before,*::after{animation:none!important;transition:none!important;caret-color:transparent!important}" });
}

async function login(page: Page): Promise<void> {
	await page.locator("input[autocomplete='username']").fill("admin");
	await page.locator("input[autocomplete='current-password']").fill("Admin@123456");
	await page.locator("button[type='submit']").click();
	await expect(page).not.toHaveURL(/\/login/);
	await expect(page.locator(".header__status")).toHaveAttribute("aria-label", "Synced");
}
