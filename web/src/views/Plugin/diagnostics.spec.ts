import { afterEach, describe, expect, it, vi } from "vitest";

import { getPluginDiagnostics, retryPluginDeadLetter } from "./api";

afterEach(() => vi.unstubAllGlobals());

describe("plugin diagnostics API", () => {
	it("encodes diagnostics filters without leaking raw identifiers into query syntax", async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: "ok", message: "ok", data: { jobs: [], audit: [], errors: [] } }), {
			status: 200,
			headers: { "Content-Type": "application/json" }
		}));
		vi.stubGlobal("fetch", fetchMock);

		await getPluginDiagnostics("pharma_oa", { jobStatus: "dead_letter", auditResult: "failure", correlation: "trace id/1", limit: 25 });
		const [url] = fetchMock.mock.calls[0] as [string, RequestInit];
		expect(url).toContain("/v1/plugins/pharma_oa/diagnostics?");
		expect(url).toContain("jobStatus=dead_letter");
		expect(url).toContain("auditResult=failure");
		expect(url).toContain("correlation=trace+id%2F1");
	});

	it("sends both exact retry confirmations", async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: "ok", message: "ok", data: { retryJob: {} } }), {
			status: 200,
			headers: { "Content-Type": "application/json" }
		}));
		vi.stubGlobal("fetch", fetchMock);

		await retryPluginDeadLetter("pharma_oa", "expiry/scan");
		const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
		expect(url).toContain("/v1/plugins/pharma_oa/jobs/expiry%2Fscan/retry");
		expect(JSON.parse(String(init.body))).toEqual({ confirmPluginId: "pharma_oa", confirmJobId: "expiry/scan" });
	});
});
