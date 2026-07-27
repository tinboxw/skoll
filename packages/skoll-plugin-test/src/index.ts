export const PLUGIN_UI_STATES = [
	"loading",
	"empty",
	"error",
	"forbidden",
	"conflict",
	"success"
] as const;

export type PluginUIState = (typeof PLUGIN_UI_STATES)[number];

export type PluginBrowserPage = {
	addInitScript(script: () => void): Promise<void>;
	evaluate<Result>(script: () => Result): Promise<Result>;
};

export type PluginPerformanceBudget = {
	routeReadyMs: number;
	interactionMs: number;
	maxLongTaskMs: number;
	totalLongTaskMs: number;
	heapGrowthBytes: number;
};

export type PluginPerformanceMeasurement = {
	routeReadyMs: number;
	interactionMs: number;
	longTasks: number[];
	heapGrowthBytes: number;
};

export type PluginVisualCase = {
	state: PluginUIState;
	viewport: string;
	theme: string;
	locale: string;
	snapshot: string;
};

export async function runPluginStateMatrix<State extends string, Context>(
	states: readonly State[],
	activate: (state: State) => Promise<Context>,
	verify: (state: State, context: Context) => Promise<void>
): Promise<void> {
	if (states.length === 0) throw new Error("plugin state matrix requires at least one state");
	for (const state of states) {
		const context = await activate(state);
		await verify(state, context);
	}
}

export function createPluginVisualMatrix(
	pluginId: string,
	options: {
		states?: readonly PluginUIState[];
		viewports?: readonly string[];
		themes?: readonly string[];
		locales?: readonly string[];
	} = {}
): PluginVisualCase[] {
	const id = normalizedSegment(pluginId, "plugin id");
	const states = options.states ?? PLUGIN_UI_STATES;
	const viewports = options.viewports ?? ["desktop", "mobile"];
	const themes = options.themes ?? ["light", "dark"];
	const locales = options.locales ?? ["zh-CN", "en-US"];
	const cases: PluginVisualCase[] = [];
	for (const state of states) {
		for (const viewport of viewports) {
			for (const theme of themes) {
				for (const locale of locales) {
					const segments = [id, state, viewport, theme, locale].map((value) => normalizedSegment(value, "visual dimension"));
					cases.push({ state, viewport, theme, locale, snapshot: `${segments.join("-")}.png` });
				}
			}
		}
	}
	return cases;
}

export async function installPluginPerformanceObserver(page: PluginBrowserPage): Promise<void> {
	await page.addInitScript(() => {
		const target = window as Window & { __skollPluginLongTasks?: number[] };
		target.__skollPluginLongTasks = [];
		try {
			new PerformanceObserver((entries) => {
				target.__skollPluginLongTasks?.push(...entries.getEntries().map((entry) => entry.duration));
			}).observe({ type: "longtask", buffered: true });
		} catch {
			target.__skollPluginLongTasks = [];
		}
	});
}

export async function readAndResetPluginLongTasks(page: PluginBrowserPage): Promise<number[]> {
	return page.evaluate(() => {
		const target = window as Window & { __skollPluginLongTasks?: number[] };
		const entries = [...(target.__skollPluginLongTasks ?? [])];
		target.__skollPluginLongTasks = [];
		return entries;
	});
}

export async function measurePluginAction(
	action: () => Promise<void>,
	ready: () => Promise<void>,
	now: () => number = () => performance.now()
): Promise<number> {
	const startedAt = now();
	await action();
	await ready();
	return now() - startedAt;
}

export function assertPluginPerformanceBudget(
	measurement: PluginPerformanceMeasurement,
	budget: PluginPerformanceBudget
): void {
	const maximumLongTask = Math.max(0, ...measurement.longTasks);
	const totalLongTasks = measurement.longTasks.reduce((total, value) => total + value, 0);
	const failures = [
		budgetFailure("route readiness", measurement.routeReadyMs, budget.routeReadyMs),
		budgetFailure("interaction readiness", measurement.interactionMs, budget.interactionMs),
		budgetFailure("maximum long task", maximumLongTask, budget.maxLongTaskMs),
		budgetFailure("total long tasks", totalLongTasks, budget.totalLongTaskMs),
		budgetFailure("heap growth", measurement.heapGrowthBytes, budget.heapGrowthBytes)
	].filter((failure): failure is string => failure !== undefined);
	if (failures.length > 0) {
		throw new Error(`plugin performance budget failed:\n${failures.join("\n")}`);
	}
}

export async function assertPluginViewportFits(page: PluginBrowserPage): Promise<void> {
	const overflow = await page.evaluate(() => {
		const root = document.documentElement;
		return {
			viewportWidth: root.clientWidth,
			documentWidth: Math.max(root.scrollWidth, document.body?.scrollWidth ?? 0)
		};
	});
	if (overflow.documentWidth > overflow.viewportWidth + 1) {
		throw new Error(`plugin viewport overflow: document ${overflow.documentWidth}px > viewport ${overflow.viewportWidth}px`);
	}
}

function budgetFailure(label: string, actual: number, maximum: number): string | undefined {
	if (!Number.isFinite(actual) || actual < 0) return `${label}: invalid measurement ${actual}`;
	if (!Number.isFinite(maximum) || maximum < 0) return `${label}: invalid budget ${maximum}`;
	return actual > maximum ? `${label}: ${actual} > ${maximum}` : undefined;
}

function normalizedSegment(value: string, label: string): string {
	const normalized = String(value).trim().replaceAll("_", "-").replace(/[^A-Za-z0-9-]+/g, "-").replace(/-+/g, "-").replace(/^-|-$/g, "");
	if (normalized === "") throw new Error(`${label} is required`);
	return normalized.toLowerCase();
}
