import { describe, expect, it } from "vitest";
import { createOAEditorForm, hydrateOAEditorForm, toOARequestWriteInput, validateOAEditorForm } from "./oa-form";
import type { OARequest } from "./types";

describe("OA request form", () => {
  it("validates and maps every request kind to the public contract", () => {
    const leave = createOAEditorForm("leave");
    Object.assign(leave, { title: "年假申请", approverId: "u-2", approverName: "李经理" });
    expect(validateOAEditorForm(leave)).toBe(true);
    expect(toOARequestWriteInput(leave, { tenantId: "tenant-a", organizationId: "org-a" }).formData).toEqual({
      leaveType: "annual",
      startDate: leave.startDate,
      endDate: leave.endDate
    });

    const procurement = createOAEditorForm("procurement");
    Object.assign(procurement, { title: "冷链箱采购", approverId: "u-2", approverName: "李经理", amount: 1200, purpose: "冷链运输" });
    expect(validateOAEditorForm(procurement)).toBe(true);
    expect(toOARequestWriteInput(procurement, { tenantId: "", organizationId: "" }).formData).toEqual({ amount: 1200, purpose: "冷链运输" });

    const expense = createOAEditorForm("expense");
    Object.assign(expense, { title: "差旅报销", approverId: "u-2", approverName: "李经理", amount: 880, category: "travel" });
    expect(validateOAEditorForm(expense)).toBe(true);
    expect(toOARequestWriteInput(expense, { tenantId: "", organizationId: "" }).formData).toEqual({ amount: 880, category: "travel" });

    const contract = createOAEditorForm("contract");
    Object.assign(contract, { title: "经销合同", approverId: "u-2", approverName: "李经理", amount: 50000, counterparty: "华东医药" });
    expect(validateOAEditorForm(contract)).toBe(true);
    expect(toOARequestWriteInput(contract, { tenantId: "", organizationId: "" }).formData).toEqual({
      amount: 50000,
      counterparty: "华东医药",
      effectiveDate: contract.effectiveDate
    });

    const custom = createOAEditorForm("custom");
    Object.assign(custom, { title: "业务申请", approverId: "u-2", approverName: "李经理", formKey: "medical-visit", customContent: "重点客户拜访" });
    expect(validateOAEditorForm(custom)).toBe(true);
    expect(toOARequestWriteInput(custom, { tenantId: "", organizationId: "" }).formData).toEqual({ formKey: "medical-visit", content: "重点客户拜访" });
  });

  it("hydrates a draft without losing its optimistic version inputs", () => {
    const request = {
      id: "request-1",
      requestType: "expense",
      title: "差旅报销",
      description: "上海客户拜访",
      formData: { amount: 880, category: "travel" },
      status: "draft",
      approverId: "u-2",
      approverName: "李经理",
      attachments: [],
      comments: [],
      version: 4,
      createdAt: "2026-07-26T08:00:00Z",
      updatedAt: "2026-07-26T08:00:00Z"
    } satisfies OARequest;
    const form = hydrateOAEditorForm(request);
    expect(form.amount).toBe(880);
    expect(toOARequestWriteInput(form, { tenantId: "tenant-a", organizationId: "org-a" }, request.version).version).toBe(4);
  });
});
