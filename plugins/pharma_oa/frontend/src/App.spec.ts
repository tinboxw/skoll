import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { PluginHostSDK } from "@skoll/plugin-sdk";
import App from "./App.vue";
import { installPharmaTestHost } from "./test-host";

const request = vi.fn<PluginHostSDK["request"]>();

beforeEach(() => {
  localStorage.clear();
  request.mockReset();
  request.mockResolvedValue({ items: [], total: 0 });
  installPharmaTestHost(request, {
    theme: { colorScheme: "dark", density: "compact", tokens: {} }
  });
});

describe("medical master-data workspace", () => {
  it("renders Chinese OA navigation and follows host theme", async () => {
    const wrapper = mount(App, { attachTo: document.body });
    await flushPromises();
    const labels = wrapper.findAll(".primary-tabs button").map((button) => button.text());
    expect(labels).toEqual(["申请中心", "待我审批", "采购与入库", "主数据"]);
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

  it("lazy-loads the Chinese purchasing workspace from the current routes", async () => {
    const wrapper = mount(App);
    await flushPromises();
    const purchase = wrapper.findAll(".primary-tabs button").find((button) => button.text() === "采购与入库");
    await purchase!.trigger("click");
    await flushPromises();
    await vi.waitFor(() => expect(wrapper.get("h2").text()).toBe("采购执行工作台"));
    expect(request.mock.calls.some(([path]) => path.includes("/v1/plugins/pharma_oa/api/purchase-requests"))).toBe(true);
  });

  it("renders a controlled state for missing and forged host capabilities", async () => {
    delete window.__SKOLL_HOST__;
    const missing = mount(App);
    await flushPromises();
    expect(missing.get('[role="alert"]').text()).toContain("未连接 SKOLL 插件宿主");
    missing.unmount();

    const host = installPharmaTestHost(request);
    window.__SKOLL_HOST__ = {
      ...host,
      capabilities: host.capabilities.filter((capability) => capability !== "request")
    };
    const forged = mount(App);
    await flushPromises();
    expect(forged.get('[role="alert"]').text()).toContain("未连接 SKOLL 插件宿主");
    expect(request).not.toHaveBeenCalled();
    forged.unmount();
  });
});
