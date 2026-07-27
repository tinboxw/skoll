export const PLUGIN_HOST_CONTRACT = "skoll.plugin-host" as const;
export const PLUGIN_HOST_VERSION = "1.0" as const;

export const PLUGIN_HOST_CAPABILITIES = [
	"request",
	"navigation",
	"commands",
	"permissions",
	"locale",
	"theme",
	"lifecycle",
	"auth",
	"user",
	"organization",
	"dictionary",
	"file",
	"audit",
	"config"
] as const;

export type PluginHostCapability = (typeof PLUGIN_HOST_CAPABILITIES)[number];
export type PluginHostMethod = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
export type PluginHostThemeColorScheme = "light" | "dark";
export type PluginHostThemeDensity = "comfortable" | "compact";
export type PluginHostLifecycleState = "enabled" | "disabled" | "degraded";
export type PluginHostCommand = "reload" | "home";

export type PluginHostRequestOptions = {
	method?: PluginHostMethod;
	body?: unknown;
	headers?: Record<string, string>;
};

export type PluginHostTheme = {
	colorScheme: PluginHostThemeColorScheme;
	density: PluginHostThemeDensity;
	tokens: Readonly<Record<string, string>>;
};

export type PluginHostIdentity = {
	subject: string;
	roles: readonly string[];
	permissions: readonly string[];
};

export type PluginHostLifecycle = {
	state: PluginHostLifecycleState;
	health: string;
};

export type PluginHostContext = {
	contract: typeof PLUGIN_HOST_CONTRACT;
	version: typeof PLUGIN_HOST_VERSION;
	pluginId: string;
	pluginVersion: string;
	capabilities: readonly PluginHostCapability[];
	identity: PluginHostIdentity;
	locale: string;
	locales: readonly string[];
	theme: PluginHostTheme;
	lifecycle: PluginHostLifecycle;
};

export type PluginHostSDK = PluginHostContext & {
	hasCapability: (capability: PluginHostCapability) => boolean;
	requireCapability: (capability: PluginHostCapability) => void;
	request: <T = unknown>(path: string, options?: PluginHostRequestOptions) => Promise<T>;
	navigation: {
		push: (path: string) => void;
		replace: (path: string) => void;
		back: () => void;
	};
	commands: {
		execute: (command: PluginHostCommand) => void;
	};
	permissions: {
		has: (permission: string) => boolean;
		require: (permission: string) => void;
	};
	auth: { me: () => Promise<unknown> };
	user: { me: () => Promise<unknown> };
	organization: {
		departments: () => Promise<unknown>;
		positions: () => Promise<unknown>;
	};
	dictionary: {
		list: () => Promise<unknown>;
		items: (type: string) => Promise<unknown>;
	};
	file: { list: (query?: Record<string, string | number | boolean>) => Promise<unknown> };
	audit: { list: (query?: Record<string, string | number | boolean>) => Promise<unknown> };
	config: {
		get: () => Promise<unknown>;
		update: (config: Record<string, unknown>) => Promise<unknown>;
	};
};

export type PluginHostErrorCode =
	| "HOST_UNAVAILABLE"
	| "CONTRACT_MISMATCH"
	| "VERSION_UNSUPPORTED"
	| "PLUGIN_IDENTITY_MISMATCH"
	| "CAPABILITY_MISSING"
	| "PERMISSION_DENIED"
	| "REQUEST_FAILED";

export class PluginHostError extends Error {
	readonly code: PluginHostErrorCode;
	readonly detail: Readonly<Record<string, string>>;

	constructor(code: PluginHostErrorCode, message: string, detail: Record<string, string> = {}) {
		super(message);
		this.name = "PluginHostError";
		this.code = code;
		this.detail = Object.freeze({ ...detail });
	}
}

export type PluginHostResolveOptions = {
	pluginId: string;
	requiredCapabilities?: readonly PluginHostCapability[];
};

export type PluginHostResolution =
	| { ok: true; host: PluginHostSDK }
	| { ok: false; error: PluginHostError };

declare global {
	interface Window {
		__SKOLL_HOST__?: PluginHostSDK;
	}
}

export function resolvePluginHost(options: PluginHostResolveOptions): PluginHostResolution {
	const expectedPluginID = options.pluginId.trim();
	const candidate = typeof window === "undefined" ? undefined : window.__SKOLL_HOST__;
	if (!candidate) {
		return failure("HOST_UNAVAILABLE", "Skoll plugin host is unavailable");
	}
	if (candidate.contract !== PLUGIN_HOST_CONTRACT) {
		return failure("CONTRACT_MISMATCH", "Skoll plugin host contract is invalid");
	}
	if (candidate.version !== PLUGIN_HOST_VERSION) {
		return failure("VERSION_UNSUPPORTED", "Skoll plugin host version is unsupported", {
			actual: String(candidate.version),
			expected: PLUGIN_HOST_VERSION
		});
	}
	if (expectedPluginID === "" || candidate.pluginId !== expectedPluginID) {
		return failure("PLUGIN_IDENTITY_MISMATCH", "Skoll plugin host identity does not match", {
			actual: String(candidate.pluginId),
			expected: expectedPluginID
		});
	}
	if (!isPluginHostShape(candidate)) {
		return failure("CONTRACT_MISMATCH", "Skoll plugin host contract shape is invalid");
	}
	const knownCapabilities = new Set<PluginHostCapability>(PLUGIN_HOST_CAPABILITIES);
	const declaredCapabilities = new Set<PluginHostCapability>();
	for (const capability of candidate.capabilities) {
		if (!knownCapabilities.has(capability) || declaredCapabilities.has(capability) || !candidate.hasCapability(capability)) {
			return failure("CONTRACT_MISMATCH", "Skoll plugin host capability declaration is invalid");
		}
		declaredCapabilities.add(capability);
	}
	for (const capability of options.requiredCapabilities ?? []) {
		if (!declaredCapabilities.has(capability) || !candidate.hasCapability(capability)) {
			return failure("CAPABILITY_MISSING", `Skoll plugin host capability '${capability}' is unavailable`, { capability });
		}
	}
	return { ok: true, host: candidate };
}

export function getPluginHost(options: PluginHostResolveOptions): PluginHostSDK {
	const resolution = resolvePluginHost(options);
	if (!resolution.ok) {
		throw resolution.error;
	}
	return resolution.host;
}

export function isPluginHostError(error: unknown): error is PluginHostError {
	return error instanceof PluginHostError ||
		Boolean(error && typeof error === "object" && (error as { name?: unknown }).name === "PluginHostError" &&
			typeof (error as { code?: unknown }).code === "string");
}

function isPluginHostShape(candidate: PluginHostSDK): boolean {
	return (
		typeof candidate.pluginVersion === "string" &&
		Array.isArray(candidate.capabilities) &&
		typeof candidate.identity === "object" &&
		candidate.identity !== null &&
		Array.isArray(candidate.identity.roles) &&
		Array.isArray(candidate.identity.permissions) &&
		typeof candidate.locale === "string" &&
		Array.isArray(candidate.locales) &&
		typeof candidate.theme === "object" &&
		candidate.theme !== null &&
		typeof candidate.lifecycle === "object" &&
		candidate.lifecycle !== null &&
		typeof candidate.hasCapability === "function" &&
		typeof candidate.requireCapability === "function" &&
		typeof candidate.request === "function" &&
		typeof candidate.navigation?.push === "function" &&
		typeof candidate.navigation?.replace === "function" &&
		typeof candidate.navigation?.back === "function" &&
		typeof candidate.commands?.execute === "function" &&
		typeof candidate.permissions?.has === "function" &&
		typeof candidate.permissions?.require === "function"
	);
}

function failure(
	code: PluginHostErrorCode,
	message: string,
	detail: Record<string, string> = {}
): PluginHostResolution {
	return { ok: false, error: new PluginHostError(code, message, detail) };
}
