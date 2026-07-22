import { expect, test } from "@playwright/test";

const FIXTURE = "/tests/fixtures/document-kit.html";

for (const theme of ["light", "dark"] as const) {
	for (const density of ["comfortable", "compact"] as const) {
		test(`${theme} ${density} document detail remains responsive`, async ({ page }, testInfo) => {
			await page.goto(`${FIXTURE}?view=detail&theme=${theme}&density=${density}&locale=en-US`);
			await expect(page.locator(".document-detail")).toBeVisible();
			await expect(page.getByText("Cold-chain medicine purchase")).toBeVisible();
			await assertNoViewportOverflow(page);
			await testInfo.attach(`${theme}-${density}-detail`, { body: await page.screenshot({ fullPage: true }), contentType: "image/png" });
		});
	}
}

for (const state of ["ready", "loading", "empty", "error", "forbidden"] as const) {
	test(`document list ${state} state`, async ({ page }) => {
		await page.goto(`${FIXTURE}?view=list&state=${state}&locale=zh-CN`);
		await expect(page.locator(".document-list")).toBeVisible();
		if (state === "empty") await expect(page.getByText("暂无业务单据")).toBeVisible();
		if (state === "error") await expect(page.locator(".state-block--error")).toBeVisible();
		if (state === "forbidden") await expect(page.locator(".state-block--forbidden")).toBeVisible();
		await assertNoViewportOverflow(page);
	});
}

test("document form and print redaction remain usable", async ({ page }) => {
	await page.goto(`${FIXTURE}?view=form&theme=dark&density=compact&locale=zh-CN`);
	await expect(page.locator(".document-form")).toBeVisible();
	await expect(page.getByText("采购明细")).toBeVisible();
	await assertNoViewportOverflow(page);

	await page.goto(`${FIXTURE}?view=print&theme=light&locale=en-US`);
	await expect(page.locator(".document-print")).toBeVisible();
	await expect(page.getByText("Redacted", { exact: true })).toBeVisible();
	await expect(page.getByText("128000.50")).toHaveCount(0);
	await assertNoViewportOverflow(page);
});

async function assertNoViewportOverflow(page: import("@playwright/test").Page): Promise<void> {
	const result = await page.evaluate(() => ({
		documentWidth: document.documentElement.scrollWidth,
		viewportWidth: document.documentElement.clientWidth,
		overlaps: Array.from(document.querySelectorAll("button, .el-button")).filter((element) => {
			const rect = element.getBoundingClientRect();
			const scrollContainer = element.closest(".document-form__table-wrap, .document-detail__table-wrap, .data-table");
			return !scrollContainer && rect.width > 0 && (rect.left < -1 || rect.right > document.documentElement.clientWidth + 1);
		}).length
	}));
	expect(result.documentWidth).toBeLessThanOrEqual(result.viewportWidth + 1);
	expect(result.overlaps).toBe(0);
}
