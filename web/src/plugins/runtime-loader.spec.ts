import { describe, expect, it, vi } from "vitest";

import { loadPluginPageHTML, type PluginPageLoadError } from "./runtime-loader";

const validHTML = "<!doctype html><html><head></head><body><main>ready</main></body></html>";

describe("plugin page runtime loader", () => {
	it("loads a complete document with same-origin credentials and authorization", async () => {
		const fetcher = vi.fn(async (_url: string | URL | Request, init?: RequestInit) => {
			expect(init?.credentials).toBe("same-origin");
			expect(new Headers(init?.headers).get("Authorization")).toBe("Bearer test-token");
			return new Response(validHTML, { status: 200 });
		}) as typeof fetch;

		await expect(loadPluginPageHTML("/plugin", { fetcher, token: "test-token" })).resolves.toBe(validHTML);
		expect(fetcher).toHaveBeenCalledOnce();
	});

	it("rejects HTTP failures and incomplete documents", async () => {
		await expect(loadPluginPageHTML("/plugin", {
			fetcher: vi.fn(async () => new Response("missing", { status: 503 })) as typeof fetch
		})).rejects.toMatchObject({ code: "http" });

		await expect(loadPluginPageHTML("/plugin", {
			fetcher: vi.fn(async () => new Response("<main>fragment</main>", { status: 200 })) as typeof fetch
		})).rejects.toMatchObject({ code: "invalid" });
	});

	it("rejects declared and measured oversized documents", async () => {
		await expect(loadPluginPageHTML("/plugin", {
			maxBytes: 32,
			fetcher: vi.fn(async () => new Response(validHTML, {
				status: 200,
				headers: { "Content-Length": "1024" }
			})) as typeof fetch
		})).rejects.toMatchObject({ code: "oversized" });

		await expect(loadPluginPageHTML("/plugin", {
			maxBytes: 32,
			fetcher: vi.fn(async () => new Response(validHTML, { status: 200 })) as typeof fetch
		})).rejects.toMatchObject({ code: "oversized" });
	});

	it("distinguishes timeout from caller cancellation", async () => {
		const pendingFetcher = vi.fn((_url: string | URL | Request, init?: RequestInit) => new Promise<Response>((_resolve, reject) => {
			init?.signal?.addEventListener("abort", () => reject(new DOMException("aborted", "AbortError")), { once: true });
		})) as typeof fetch;

		await expect(loadPluginPageHTML("/plugin", {
			fetcher: pendingFetcher,
			timeoutMs: 5
		})).rejects.toMatchObject({ code: "timeout" });

		const controller = new AbortController();
		const result = loadPluginPageHTML("/plugin", {
			fetcher: pendingFetcher,
			signal: controller.signal,
			timeoutMs: 1_000
		});
		controller.abort();
		await expect(result).rejects.toEqual(expect.objectContaining<Partial<PluginPageLoadError>>({ code: "aborted" }));
	});
});
