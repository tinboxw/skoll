import { defineConfig } from "@playwright/test";

export default defineConfig({
	testDir: "./tests/e2e",
	outputDir: "test-results/pr4-quality",
	snapshotPathTemplate: "{testDir}/visual-baselines/{projectName}/{testFilePath}/{arg}{ext}",
	fullyParallel: false,
	workers: 1,
	timeout: 90_000,
	expect: {
		timeout: 12_000,
		toHaveScreenshot: {
			animations: "disabled",
			caret: "hide",
			maxDiffPixelRatio: 0.03,
			threshold: 0.2
		}
	},
	reporter: [
		["line"],
		["json", { outputFile: "test-results/pr4-quality/results.json" }]
	],
	use: {
		baseURL: process.env.SKOLL_E2E_BASE_URL ?? "http://127.0.0.1:5174",
		browserName: "chromium",
		channel: process.env.SKOLL_E2E_BROWSER_CHANNEL || "chrome",
		headless: true,
		reducedMotion: "reduce",
		screenshot: "only-on-failure",
		trace: "retain-on-failure"
	},
	projects: [
		{
			name: "desktop",
			use: { viewport: { width: 1440, height: 1000 } }
		},
		{
			name: "mobile",
			use: { viewport: { width: 390, height: 844 }, hasTouch: true }
		}
	]
});
