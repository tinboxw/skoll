export const PLUGIN_PAGE_LOAD_TIMEOUT_MS = 5_000;
export const PLUGIN_PAGE_RENDER_TIMEOUT_MS = 8_000;
export const PLUGIN_PAGE_MAX_BYTES = 1_048_576;

export type PluginPageLoadErrorCode = "aborted" | "http" | "invalid" | "oversized" | "timeout";

export class PluginPageLoadError extends Error {
	readonly code: PluginPageLoadErrorCode;

	constructor(code: PluginPageLoadErrorCode, message: string) {
		super(message);
		this.name = "PluginPageLoadError";
		this.code = code;
	}
}

export type LoadPluginPageOptions = {
	fetcher?: typeof fetch;
	signal?: AbortSignal;
	timeoutMs?: number;
	maxBytes?: number;
	token?: string;
};

export async function loadPluginPageHTML(url: string, options: LoadPluginPageOptions = {}): Promise<string> {
	const fetcher = options.fetcher ?? fetch;
	const timeoutMs = options.timeoutMs ?? PLUGIN_PAGE_LOAD_TIMEOUT_MS;
	const maxBytes = options.maxBytes ?? PLUGIN_PAGE_MAX_BYTES;
	if (!Number.isFinite(timeoutMs) || timeoutMs <= 0 || !Number.isFinite(maxBytes) || maxBytes <= 0) {
		throw new PluginPageLoadError("invalid", "plugin page runtime limits are invalid");
	}

	const controller = new AbortController();
	let timedOut = false;
	const abortFromCaller = () => controller.abort();
	options.signal?.addEventListener("abort", abortFromCaller, { once: true });
	const timeout = window.setTimeout(() => {
		timedOut = true;
		controller.abort();
	}, timeoutMs);

	try {
		const headers = new Headers();
		const token = options.token?.trim() ?? "";
		if (token !== "") {
			headers.set("Authorization", token.toLowerCase().startsWith("bearer ") ? token : `Bearer ${token}`);
		}
		const response = await fetcher(url, {
			credentials: "same-origin",
			headers,
			signal: controller.signal
		});
		if (!response.ok) {
			throw new PluginPageLoadError("http", `plugin page request failed: ${response.status}`);
		}
		const declaredBytes = Number.parseInt(response.headers.get("content-length") ?? "", 10);
		if (Number.isFinite(declaredBytes) && declaredBytes > maxBytes) {
			throw new PluginPageLoadError("oversized", `plugin page exceeds ${maxBytes} bytes`);
		}
		const content = await response.text();
		if (new TextEncoder().encode(content).byteLength > maxBytes) {
			throw new PluginPageLoadError("oversized", `plugin page exceeds ${maxBytes} bytes`);
		}
		if (!/<html[\s>]/i.test(content) || !/<body[\s>]/i.test(content)) {
			throw new PluginPageLoadError("invalid", "plugin page is not a complete HTML document");
		}
		return content;
	} catch (error) {
		if (error instanceof PluginPageLoadError) {
			throw error;
		}
		if (controller.signal.aborted) {
			throw new PluginPageLoadError(timedOut ? "timeout" : "aborted", timedOut
				? `plugin page request exceeded ${timeoutMs}ms`
				: "plugin page request was cancelled");
		}
		throw new PluginPageLoadError("http", error instanceof Error ? error.message : "plugin page request failed");
	} finally {
		window.clearTimeout(timeout);
		options.signal?.removeEventListener("abort", abortFromCaller);
	}
}
