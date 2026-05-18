const DEFAULT_WEB_BASE_PATH = "/skoll";

function normalizeWebBasePath(raw: string): string {
	const trimmed = raw.trim();
	if (trimmed === "") {
		return DEFAULT_WEB_BASE_PATH;
	}
	if (trimmed === "/") {
		return "/";
	}
	const withSlash = trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
	const normalized = withSlash.replace(/\/+$/, "");
	return normalized === "" ? "/" : normalized;
}

function resolveInjectedBasePath(): string {
	if (typeof __SKOLL_WEB_BASE_PATH__ === "string" && __SKOLL_WEB_BASE_PATH__.trim() !== "") {
		return __SKOLL_WEB_BASE_PATH__;
	}
	return DEFAULT_WEB_BASE_PATH;
}

export const WEB_BASE_PATH = normalizeWebBasePath(resolveInjectedBasePath());

export function withWebBasePath(path: string): string {
	const normalized = path.trim() === "" ? "/" : path.trim();
	if (WEB_BASE_PATH === "/") {
		return normalized.startsWith("/") ? normalized : `/${normalized}`;
	}
	const withSlash = normalized.startsWith("/") ? normalized : `/${normalized}`;
	if (withSlash === WEB_BASE_PATH || withSlash.startsWith(`${WEB_BASE_PATH}/`)) {
		return withSlash;
	}
	if (withSlash === "/") {
		return WEB_BASE_PATH;
	}
	return `${WEB_BASE_PATH}${withSlash}`;
}

export function stripWebBasePath(path: string): string {
	const normalized = path.trim() === "" ? "/" : path.trim();
	if (WEB_BASE_PATH === "/") {
		return normalized.startsWith("/") ? normalized : `/${normalized}`;
	}
	if (normalized === WEB_BASE_PATH) {
		return "/";
	}
	if (normalized.startsWith(`${WEB_BASE_PATH}/`)) {
		return normalized.slice(WEB_BASE_PATH.length);
	}
	return normalized.startsWith("/") ? normalized : `/${normalized}`;
}
