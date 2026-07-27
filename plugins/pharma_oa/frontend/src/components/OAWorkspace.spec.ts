import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { PluginHostSDK } from "@skoll/plugin-sdk";
import OAWorkspace from "./OAWorkspace.vue";
import { installPharmaTestHost } from "../test-host";
import type { OARequest, OARequestDetail } from "../types";

const pending: OARequest = {
  id: "request-1",
  requestType: "leave",
  title: "王敏年假申请",
  description: "家庭安排",
  formData: { leaveType: "annual", startDate: "2026-07-28", endDate: "2026-07-30" },
  status: "pending",
  approverId: "manager-1",
  approverName: "李经理",
  workflowInstanceId: "workflow-1",
  submittedAt: "2026-07-26T08:10:00Z",
  attachments: [],
  comments: [],
  version: 2,
  createdAt: "2026-07-26T08:00:00Z",
  updatedAt: "2026-07-26T08:10:00Z"
};

const detail: OARequestDetail = {
  item: pending,
  workflow: {
    id: "workflow-1",
    definitionId: "definition-1",
    definitionKey: "oa-request-1",
    businessType: "oa_request",
    businessId: pending.id,
    title: pending.title,
    status: "running",
    starter: { id: "employee-1", name: "王敏" },
    currentNode: "approval",
    tasks: [{ id: "task-1", instanceId: "workflow-1", nodeId: "approval", assignee: { id: "manager-1", name: "李经理" }, status: "pending", createdAt: pending.submittedAt! }],
    timeline: [{ id: "action-1", type: "start", instanceId: "workflow-1", taskId: "", nodeId: "start", actor: { id: "employee-1", name: "王敏" }, target: { id: "", name: "" }, comment: "", createdAt: pending.submittedAt! }],
    createdAt: pending.submittedAt!,
    updatedAt: pending.submittedAt!
  }
};

const request = vi.fn<PluginHostSDK["request"]>();

beforeEach(() => {
  request.mockReset();
  request.mockImplementation(async (path, options) => {
    if (path.endsWith("/oa-requests") || path.includes("/oa-requests?")) return { items: [pending], total: 1, offset: 0, limit: 200 };
    if (path.endsWith(`/oa-requests/${pending.id}`)) return detail;
    if (path.endsWith(`/oa-requests/${pending.id}/approve`) && options?.method === "POST") {
      return { item: { ...pending, status: "approved", version: 3 }, workflow: { ...detail.workflow!, status: "approved" } };
    }
    return { items: [], total: 0 };
  });
  installPharmaTestHost(request);
});

describe("OA workspace", () => {
  it("renders a Chinese-first request list with complete filters", async () => {
    const wrapper = mount(OAWorkspace, { props: { mode: "requests", scope: { tenantId: "tenant-a", organizationId: "org-a" } } });
    await flushPromises();
    expect(wrapper.get("h2").text()).toBe("OA 申请与审批");
    expect(wrapper.text()).toContain("王敏年假申请");
    expect(wrapper.text()).toContain("请假申请");
    expect(wrapper.text()).toContain("新建申请");
  });

  it("opens a pending task and completes approval through the public route", async () => {
    const wrapper = mount(OAWorkspace, { attachTo: document.body, props: { mode: "inbox", scope: { tenantId: "tenant-a", organizationId: "org-a" } } });
    await flushPromises();
    const detailButton = wrapper.findAll("button").find((button) => button.attributes("aria-label") === "查看详情");
    await detailButton!.trigger("click");
    await flushPromises();
    expect(wrapper.text()).toContain("审批轨迹");

    const approve = wrapper.findAll("button").find((button) => button.text() === "通过");
    await approve!.trigger("click");
    await flushPromises();
    const confirm = document.body.querySelector(".el-dialog__footer .el-button--primary") as HTMLButtonElement;
    confirm.click();
    await flushPromises();
    expect(request.mock.calls.some(([path]) => path.endsWith("/oa-requests/request-1/approve"))).toBe(true);
    wrapper.unmount();
  });

  it("renders empty and permission-denied states without false success", async () => {
    request.mockResolvedValueOnce({ items: [], total: 0, offset: 0, limit: 200 });
    const empty = mount(OAWorkspace, { props: { mode: "requests", scope: { tenantId: "", organizationId: "" } } });
    await flushPromises();
    expect(empty.text()).toContain("暂无申请");
    empty.unmount();

    request.mockRejectedValueOnce(new Error("forbidden: permission denied"));
    const denied = mount(OAWorkspace, { props: { mode: "requests", scope: { tenantId: "", organizationId: "" } } });
    await flushPromises();
    expect(denied.text()).toContain("缺少当前模块访问权限");
    expect(denied.text()).not.toContain("申请草稿已保存");
  });
});
