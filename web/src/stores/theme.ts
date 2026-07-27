import { defineStore } from "pinia";
import { PLUGIN_HOST_THEME_TOKENS } from "@skoll/plugin-sdk";

export type ThemeDensity = "comfortable" | "compact";
export type ThemeColorScheme = "light" | "dark";

export type ThemeBridgePayload = {
	colorScheme: ThemeColorScheme;
	density: ThemeDensity;
	tokens: Record<string, string>;
};

const COLOR_SCHEME_KEY = "skoll.ui.colorScheme";
const DENSITY_KEY = "skoll.ui.density";
type ThemeState = {
	colorScheme: ThemeColorScheme;
	density: ThemeDensity;
};

function normalizeColorScheme(value: unknown): ThemeColorScheme {
	return value === "dark" ? "dark" : "light";
}

function normalizeDensity(value: unknown): ThemeDensity {
	return value === "compact" ? "compact" : "comfortable";
}

function detectColorScheme(): ThemeColorScheme {
	if (typeof localStorage === "undefined") {
		return "light";
	}
	return normalizeColorScheme(localStorage.getItem(COLOR_SCHEME_KEY));
}

function detectDensity(): ThemeDensity {
	if (typeof localStorage === "undefined") {
		return "comfortable";
	}
	return normalizeDensity(localStorage.getItem(DENSITY_KEY));
}

function applyThemeToDocument(colorScheme: ThemeColorScheme, density: ThemeDensity): void {
	if (typeof document === "undefined") {
		return;
	}
	const root = document.documentElement;
	root.setAttribute("data-theme", colorScheme);
	root.setAttribute("data-density", density);
	root.style.colorScheme = colorScheme;
}

function readPluginThemeTokens(): Record<string, string> {
	if (typeof document === "undefined" || typeof getComputedStyle === "undefined") {
		return {};
	}
	const styles = getComputedStyle(document.documentElement);
	return Object.fromEntries(
		PLUGIN_HOST_THEME_TOKENS.map((name) => [name, styles.getPropertyValue(name).trim()]).filter(([, value]) => value !== "")
	);
}

function createBridgePayload(state: ThemeState): ThemeBridgePayload {
	return {
		colorScheme: state.colorScheme,
		density: state.density,
		tokens: readPluginThemeTokens()
	};
}

export const useThemeStore = defineStore("theme", {
	state: (): ThemeState => ({
		colorScheme: detectColorScheme(),
		density: detectDensity()
	}),
	getters: {
		pluginBridgeTheme: (state): ThemeBridgePayload => createBridgePayload(state)
	},
	actions: {
		initializeTheme(): void {
			this.colorScheme = detectColorScheme();
			this.density = detectDensity();
			applyThemeToDocument(this.colorScheme, this.density);
		},
		setColorScheme(colorScheme: ThemeColorScheme): void {
			this.colorScheme = normalizeColorScheme(colorScheme);
			localStorage.setItem(COLOR_SCHEME_KEY, this.colorScheme);
			applyThemeToDocument(this.colorScheme, this.density);
		},
		setDensity(density: ThemeDensity): void {
			this.density = normalizeDensity(density);
			localStorage.setItem(DENSITY_KEY, this.density);
			applyThemeToDocument(this.colorScheme, this.density);
		}
	}
});
