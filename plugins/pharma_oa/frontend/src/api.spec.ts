import { beforeEach, describe, expect, it, vi } from "vitest";
import { api, API_BASE, idempotencyKey, query } from "./api";
import type { HostSDK } from "./types";

const request = vi.fn<HostSDK["request"]>();

beforeEach(() => {
  request.mockReset();
  request.mockResolvedValue({ items: [] });
  window.__SKOLL_HOST__ = {
    pluginId: "pharma_oa",
    locale: "zh-CN",
    locales: ["zh-CN", "en-US"],
    theme: { colorScheme: "light", density: "comfortable", tokens: {} },
    request: request as unknown as HostSDK["request"]
  };
});

describe("pharma OA API", () => {
  it("builds bounded encoded list queries", async () => {
    expect(query({ keyword: "阿莫西林", status: "active", limit: 200 })).toBe("?keyword=%E9%98%BF%E8%8E%AB%E8%A5%BF%E6%9E%97&status=active&limit=200");
    await api.list("product", "阿莫西林", "active");
    expect(request).toHaveBeenCalledWith(`${API_BASE}/products?keyword=%E9%98%BF%E8%8E%AB%E8%A5%BF%E6%9E%97&status=active&limit=200`, expect.objectContaining({ method: "GET" }));
  });

  it("adds a bounded idempotency key to mutations", async () => {
    await api.create("customer", { name: "华东医药" });
    const options = request.mock.calls[0][1];
    expect(request.mock.calls[0][0]).toBe(`${API_BASE}/customers`);
    expect(options?.headers?.["Idempotency-Key"]).toMatch(/^ui-post-[A-Za-z0-9-]+$/);
    expect(options?.method).toBe("POST");
  });

  it("uses the public OA request routes for list and approval", async () => {
    await api.oaRequests({ requestType: "leave", status: "pending" });
    expect(request.mock.calls[0][0]).toBe(`${API_BASE}/oa-requests?requestType=leave&status=pending&limit=200`);

    request.mockClear();
    await api.approveOARequest("request/1", "task-1", 3, "同意");
    expect(request.mock.calls[0][0]).toBe(`${API_BASE}/oa-requests/request%2F1/approve`);
    expect(request.mock.calls[0][1]).toEqual(expect.objectContaining({
      method: "POST",
      body: { taskId: "task-1", version: 3, comment: "同意" }
    }));
  });

  it("fails closed outside the declared plugin host", async () => {
    window.__SKOLL_HOST__ = { ...window.__SKOLL_HOST__!, pluginId: "another_plugin" };
    expect(() => api.employeeReminders()).toThrow("SKOLL_HOST_UNAVAILABLE");
    expect(idempotencyKey("qualification").length).toBeLessThan(96);
  });
});
