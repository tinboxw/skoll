import { defineStore } from "pinia";

export type ThemeMode = "light" | "dark" | "compact";
export type ThemeDensity = "comfortable" | "compact";
export type ThemeColorScheme = "light" | "dark";

export type ThemeBridgePayload = {
	mode: ThemeMode;
	colorScheme: ThemeColorScheme;
	density: ThemeDensity;
	tokens: Record<string, string>;
};

const THEME_KEY = "skoll.ui.theme";

const THEME_TOKENS: Record<ThemeMode, ThemeBridgePayload> = {
	light: {
		mode: "light",
		colorScheme: "light",
		density: "comfortable",
		tokens: {
			"--color-bg": "#f3f5f8",
			"--color-surface": "#ffffff",
			"--color-text": "#14202b",
			"--color-primary": "#0b6e67",
			"--color-border": "#d5dde6"
		}
	},
	dark: {
		mode: "dark",
		colorScheme: "dark",
		density: "comfortable",
		tokens: {
			"--color-bg": "#101721",
			"--color-surface": "#182332",
			"--color-text": "#edf3f8",
			"--color-primary": "#42b7ac",
			"--color-border": "#2c3b4f"
		}
	},
	compact: {
		mode: "compact",
		colorScheme: "light",
		density: "compact",
		tokens: {
			"--color-bg": "#eef2f6",
			"--color-surface": "#ffffff",
			"--color-text": "#14202b",
			"--color-primary": "#0b6e67",
			"--color-border": "#ccd6e2"
		}
	}
};

type ThemeState = {
	mode: ThemeMode;
};

function normalizeThemeMode(value: unknown): ThemeMode {
	return value === "dark" || value === "compact" || value === "light" ? value : "light";
}

function detectInitialTheme(): ThemeMode {
	if (typeof localStorage === "undefined") {
		return "light";
	}
	return normalizeThemeMode(localStorage.getItem(THEME_KEY));
}

function applyThemeToDocument(payload: ThemeBridgePayload): void {
	if (typeof document === "undefined") {
		return;
	}
	const root = document.documentElement;
	root.setAttribute("data-theme", payload.colorScheme);
	root.setAttribute("data-theme-mode", payload.mode);
	root.setAttribute("data-density", payload.density);
	root.style.colorScheme = payload.colorScheme;
}

export const useThemeStore = defineStore("theme", {
	state: (): ThemeState => ({
		mode: detectInitialTheme()
	}),
	getters: {
		payload: (state): ThemeBridgePayload => THEME_TOKENS[state.mode],
		colorScheme: (state): ThemeColorScheme => THEME_TOKENS[state.mode].colorScheme,
		density: (state): ThemeDensity => THEME_TOKENS[state.mode].density,
		pluginBridgeTheme: (state): ThemeBridgePayload => THEME_TOKENS[state.mode]
	},
	actions: {
		initializeTheme(): void {
			this.mode = detectInitialTheme();
			applyThemeToDocument(this.payload);
		},
		setTheme(mode: ThemeMode): void {
			const next = normalizeThemeMode(mode);
			this.mode = next;
			localStorage.setItem(THEME_KEY, next);
			applyThemeToDocument(this.payload);
		}
	}
});
