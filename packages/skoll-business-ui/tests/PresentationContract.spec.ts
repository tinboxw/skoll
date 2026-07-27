import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

const themeCSS = readFileSync(resolve(process.cwd(), "src/theme.css"), "utf8");

describe("business presentation contract", () => {
	it("maps host tokens to Element Plus without private theme fallbacks", () => {
		expect(themeCSS).toContain("--el-color-primary: var(--color-primary)");
		expect(themeCSS).toContain("--el-component-size: var(--control-height)");
		expect(themeCSS).toContain(':root[data-density="compact"]');
		expect(themeCSS).toContain(':root[data-theme="dark"]');
		expect(themeCSS).not.toMatch(/var\(--[a-z0-9-]+,/);
	});

	it("defines keyboard, narrow viewport, reduced motion, and forced color rules", () => {
		expect(themeCSS).toContain(":focus-visible");
		expect(themeCSS).toContain("@media (max-width: 480px)");
		expect(themeCSS).toContain("@media (prefers-reduced-motion: reduce)");
		expect(themeCSS).toContain("@media (forced-colors: active)");
	});
});
