import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, it } from "vitest";

import { useThemeStore, type ThemeColorScheme, type ThemeDensity } from "./theme";

describe("theme store", () => {
	beforeEach(() => {
		localStorage.clear();
		document.documentElement.removeAttribute("data-theme");
		document.documentElement.removeAttribute("data-density");
		document.documentElement.removeAttribute("data-theme-mode");
		document.documentElement.removeAttribute("style");
		setActivePinia(createPinia());
	});

	it("ignores the removed single-mode key and initializes current defaults", () => {
		localStorage.setItem("skoll.ui.theme", "dark");
		const store = useThemeStore();
		store.initializeTheme();

		expect(store.colorScheme).toBe("light");
		expect(store.density).toBe("comfortable");
		expect(document.documentElement.dataset.theme).toBe("light");
		expect(document.documentElement.dataset.density).toBe("comfortable");
		expect(document.documentElement.hasAttribute("data-theme-mode")).toBe(false);
	});

	it.each([
		["light", "comfortable"],
		["light", "compact"],
		["dark", "comfortable"],
		["dark", "compact"]
	] satisfies Array<[ThemeColorScheme, ThemeDensity]>)('applies the %s and %s combination independently', (colorScheme, density) => {
		const store = useThemeStore();
		document.documentElement.style.setProperty("--color-primary", "test-primary");
		store.setColorScheme(colorScheme);
		store.setDensity(density);

		expect(store.colorScheme).toBe(colorScheme);
		expect(store.density).toBe(density);
		expect(localStorage.getItem("skoll.ui.colorScheme")).toBe(colorScheme);
		expect(localStorage.getItem("skoll.ui.density")).toBe(density);
		expect(document.documentElement.dataset.theme).toBe(colorScheme);
		expect(document.documentElement.dataset.density).toBe(density);
		expect(store.pluginBridgeTheme).toEqual({
			colorScheme,
			density,
			tokens: { "--color-primary": "test-primary" }
		});
	});
});
