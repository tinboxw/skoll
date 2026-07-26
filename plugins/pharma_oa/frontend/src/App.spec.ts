import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App.vue";
import type { HostSDK } from "./types";

const request = vi.fn<HostSDK["request"]>();

beforeEach(() => {
  localStorage.clear();
  request.mockReset();
  request.mockResolvedValue({ items: [], total: 0 });
  window.__SKOLL_HOST__ = {
    pluginId: "pharma_oa",
    locale: "zh-CN",
    locales: ["zh-CN", "en-US"],
    theme: { colorScheme: "dark", density: "compact", tokens: {} },
    request: request as unknown as HostSDK["request"]
  };
});

describe("medical master-data workspace", () => {
  it("renders Chinese OA navigation and follows host theme", async () => {
    const wrapper = mount(App, { attachTo: document.body });
    await flushPromises();
    const labels = wrapper.findAll(".primary-tabs button").map((button) => button.text());
    expect(labels).toEqual(["申请中心", "待我审批", "主数据"]);
    expect(wrapper.get("h2").text()).toBe("OA 申请与审批");
    expect(document.documentElement.dataset.theme).toBe("dark");
    expect(document.documentElement.dataset.density).toBe("compact");
    wrapper.unmount();
  });

  it("switches modules and requests the independent customer endpoint", async () => {
    const wrapper = mount(App);
    await flushPromises();
    const master = wrapper.findAll(".primary-tabs button").find((button) => button.text() === "主数据");
    await master!.trigger("click");
    await flushPromises();
    const customer = wrapper.findAll(".secondary-tabs button").find((button) => button.text() === "客户管理");
    expect(customer).toBeDefined();
    await customer!.trigger("click");
    await flushPromises();
    expect(request.mock.calls.some(([path]) => path.includes("/v1/plugins/pharma_oa/api/customers"))).toBe(true);
    expect(wrapper.get("h2").text()).toBe("客户管理");
  });
});
