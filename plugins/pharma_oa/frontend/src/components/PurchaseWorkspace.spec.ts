import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { PluginHostSDK } from "@skoll/plugin-sdk";
import PurchaseWorkspace from "./PurchaseWorkspace.vue";
import { installPharmaTestHost } from "../test-host";
import type { PurchaseOrder, PurchaseRequest } from "../types";

const purchaseRequest: PurchaseRequest = {
  id: "request-1",
  number: "PR-20260726-0001",
  supplierId: "supplier-1",
  supplierCode: "SUP-001",
  supplierName: "华东医药",
  requesterId: "employee-1",
  approverId: "approver-1",
  approverName: "李审批",
  reason: "门店补货",
  currency: "CNY",
  lines: [{ id: "line-1", productId: "product-1", productCode: "MED-001", productName: "阿莫西林胶囊", specification: "0.25g*24粒", unitId: "unit-1", quantity: "2.5", unitPrice: "16.80", amount: "42", receivedQuantity: "0" }],
  totalAmount: "42",
  status: "pending",
  workflowDefinitionId: "definition-1",
  workflowInstanceId: "workflow-1",
  version: 1,
  createdAt: "2026-07-26T08:00:00Z",
  updatedAt: "2026-07-26T08:00:00Z"
};

const purchaseOrder: PurchaseOrder = {
  id: "order-1",
  number: "PO-20260726-0001",
  purchaseRequestId: purchaseRequest.id,
  supplierId: purchaseRequest.supplierId,
  supplierCode: purchaseRequest.supplierCode,
  supplierName: purchaseRequest.supplierName,
  currency: "CNY",
  lines: purchaseRequest.lines,
  totalAmount: purchaseRequest.totalAmount,
  status: "open",
  approvedBy: "approver-1",
  approvedAt: "2026-07-26T08:10:00Z",
  version: 1,
  createdAt: "2026-07-26T08:10:00Z",
  updatedAt: "2026-07-26T08:10:00Z"
};

const request = vi.fn<PluginHostSDK["request"]>();

beforeEach(() => {
  request.mockReset();
  request.mockImplementation(async (path) => {
    if (path.includes("/purchase-requests/request-1")) {
      return {
        item: purchaseRequest,
        workflow: { tasks: [{ id: "task-1", status: "pending" }], timeline: [] }
      };
    }
    if (path.includes("/purchase-requests")) return { items: [purchaseRequest], total: 1 };
    if (path.includes("/purchase-orders")) return { items: [purchaseOrder], total: 1 };
    if (path.includes("/purchase-inbounds")) return { items: [], total: 0 };
    return { items: [] };
  });
  installPharmaTestHost(request);
});

describe("purchase workspace", () => {
  it("renders the Chinese request queue and all operational views", async () => {
    const wrapper = mount(PurchaseWorkspace, { props: { scope: { tenantId: "tenant-1", organizationId: "org-1" } } });
    await flushPromises();
    expect(wrapper.get("h2").text()).toBe("采购执行工作台");
    expect(wrapper.findAll(".purchase-switch button").map((item) => item.text())).toEqual(["采购申请", "采购订单", "入库记录"]);
    expect(wrapper.text()).toContain("PR-20260726-0001");
    expect(wrapper.text()).toContain("华东医药");
  });

  it("opens the current approval task and calls the public decision route", async () => {
    request.mockImplementation(async (path) => {
      if (path.endsWith("/purchase-requests/request-1/approve")) {
        return { item: { ...purchaseRequest, status: "approved", version: 2 }, order: purchaseOrder, workflow: { tasks: [], timeline: [] }, duplicate: false };
      }
      if (path.includes("/purchase-requests/request-1")) return { item: purchaseRequest, workflow: { tasks: [{ id: "task-1", status: "pending" }], timeline: [] } };
      if (path.includes("/purchase-requests")) return { items: [purchaseRequest], total: 1 };
      if (path.includes("/purchase-orders")) return { items: [purchaseOrder], total: 1 };
      return { items: [], total: 0 };
    });
    const wrapper = mount(PurchaseWorkspace, { attachTo: document.body, props: { scope: { tenantId: "tenant-1", organizationId: "org-1" } } });
    await flushPromises();
    await wrapper.get('button[aria-label="审核通过"]').trigger("click");
    await flushPromises();
    const confirm = document.body.querySelector(".el-dialog__footer .el-button--primary") as HTMLButtonElement;
    confirm.click();
    await flushPromises();
    expect(request.mock.calls.some(([path, options]) =>
      path.endsWith("/purchase-requests/request-1/approve") &&
      (options?.body as { taskId?: string })?.taskId === "task-1"
    )).toBe(true);
    wrapper.unmount();
  });

  it("shows order receiving progress without losing exact backend values", async () => {
    const wrapper = mount(PurchaseWorkspace, { props: { scope: { tenantId: "tenant-1", organizationId: "org-1" } } });
    await flushPromises();
    await wrapper.findAll(".purchase-switch button")[1].trigger("click");
    expect(wrapper.text()).toContain("PO-20260726-0001");
    expect(wrapper.text()).toContain("0/2.5");
    expect(wrapper.text()).toContain("办理入库");
  });

  it("opens persisted inbound batch facts from the current detail route", async () => {
    const inbound = {
      id: "inbound-1",
      number: "PI-20260726-0001",
      purchaseOrderId: purchaseOrder.id,
      purchaseOrderNumber: purchaseOrder.number,
      warehouseId: "WH-1",
      areaId: "A-1",
      locationId: "L-1",
      lines: [{ orderLineId: "line-1", productId: "product-1", productCode: "MED-001", productName: "阿莫西林胶囊", quantity: "1.25", batchNo: "LOT-1", productionDate: "2026-01-01T00:00:00Z", expiresAt: "2028-01-01T00:00:00Z" }],
      attachments: [],
      status: "completed" as const,
      receivedBy: "employee-1",
      receivedAt: "2026-07-26T09:00:00Z",
      version: 1,
      createdAt: "2026-07-26T09:00:00Z",
      updatedAt: "2026-07-26T09:00:00Z"
    };
    request.mockImplementation(async (path) => {
      if (path.endsWith("/purchase-inbounds/inbound-1")) return { item: inbound };
      if (path.includes("/purchase-inbounds")) return { items: [inbound], total: 1 };
      if (path.includes("/purchase-requests")) return { items: [purchaseRequest], total: 1 };
      if (path.includes("/purchase-orders")) return { items: [purchaseOrder], total: 1 };
      return { items: [] };
    });
    const wrapper = mount(PurchaseWorkspace, { props: { scope: { tenantId: "tenant-1", organizationId: "org-1" } } });
    await flushPromises();
    await wrapper.findAll(".purchase-switch button")[2].trigger("click");
    await wrapper.get(".mobile-records .record-card > button").trigger("click");
    await flushPromises();
    expect(request.mock.calls.some(([path]) => path.endsWith("/purchase-inbounds/inbound-1"))).toBe(true);
    expect(wrapper.text()).toContain("LOT-1");
  });
});
