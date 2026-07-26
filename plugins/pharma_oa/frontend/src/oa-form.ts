import type { OARequest, OARequestType, OARequestWriteInput, Scope } from "./types";

export type OAEditorForm = {
  requestType: OARequestType;
  title: string;
  description: string;
  approverId: string;
  approverName: string;
  leaveType: string;
  startDate: string;
  endDate: string;
  amount: number;
  category: string;
  purpose: string;
  counterparty: string;
  effectiveDate: string;
  formKey: string;
  customContent: string;
};

export function createOAEditorForm(requestType: OARequestType = "leave"): OAEditorForm {
  const today = new Date().toISOString().slice(0, 10);
  return {
    requestType,
    title: "",
    description: "",
    approverId: "",
    approverName: "",
    leaveType: "annual",
    startDate: today,
    endDate: today,
    amount: 0,
    category: "travel",
    purpose: "",
    counterparty: "",
    effectiveDate: today,
    formKey: "",
    customContent: ""
  };
}

export function hydrateOAEditorForm(request: OARequest): OAEditorForm {
  const form = createOAEditorForm(request.requestType);
  const data = request.formData;
  form.title = request.title;
  form.description = request.description;
  form.approverId = request.approverId;
  form.approverName = request.approverName;
  form.leaveType = stringValue(data.leaveType, form.leaveType);
  form.startDate = stringValue(data.startDate, form.startDate);
  form.endDate = stringValue(data.endDate, form.endDate);
  form.amount = numberValue(data.amount);
  form.category = stringValue(data.category, form.category);
  form.purpose = stringValue(data.purpose);
  form.counterparty = stringValue(data.counterparty);
  form.effectiveDate = stringValue(data.effectiveDate, form.effectiveDate);
  form.formKey = stringValue(data.formKey);
  form.customContent = stringValue(data.content);
  return form;
}

export function validateOAEditorForm(form: OAEditorForm): boolean {
  if (!form.title.trim() || !form.approverId.trim() || !form.approverName.trim()) return false;
  switch (form.requestType) {
    case "leave":
      return Boolean(form.leaveType && form.startDate && form.endDate && form.endDate >= form.startDate);
    case "expense":
      return form.amount > 0 && Boolean(form.category);
    case "procurement":
      return form.amount > 0 && Boolean(form.purpose.trim());
    case "contract":
      return form.amount > 0 && Boolean(form.counterparty.trim() && form.effectiveDate);
    case "custom":
      return Boolean(form.formKey.trim());
  }
}

export function toOARequestWriteInput(form: OAEditorForm, scope: Scope, version?: number): OARequestWriteInput {
  const base: OARequestWriteInput = {
    tenantId: scope.tenantId.trim(),
    organizationId: scope.organizationId.trim(),
    requestType: form.requestType,
    title: form.title.trim(),
    description: form.description.trim(),
    approverId: form.approverId.trim(),
    approverName: form.approverName.trim(),
    formData: formData(form)
  };
  if (version) base.version = version;
  return base;
}

function formData(form: OAEditorForm): Record<string, unknown> {
  switch (form.requestType) {
    case "leave":
      return { leaveType: form.leaveType, startDate: form.startDate, endDate: form.endDate };
    case "expense":
      return { amount: Number(form.amount), category: form.category };
    case "procurement":
      return { amount: Number(form.amount), purpose: form.purpose.trim() };
    case "contract":
      return { amount: Number(form.amount), counterparty: form.counterparty.trim(), effectiveDate: form.effectiveDate };
    case "custom":
      return { formKey: form.formKey.trim(), content: form.customContent.trim() };
  }
}

function stringValue(value: unknown, fallback = ""): string {
  return typeof value === "string" ? value : fallback;
}

function numberValue(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}
