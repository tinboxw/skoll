import type { Catalog, Employee, ModuleKey, Page, Party, Product, Qualification, QualificationType, Scope, WorkspaceRecord } from "./types";

export const API_BASE = "/v1/plugins/pharma_oa/api";

const collections: Record<ModuleKey, string> = {
  employee: "/employees",
  customer: "/customers",
  supplier: "/suppliers",
  product: "/products",
  category: "/categories",
  unit: "/units",
  manufacturer: "/manufacturers",
  qualification: "/qualifications",
  qualificationType: "/qualification-types"
};

function host() {
  const value = window.__SKOLL_HOST__;
  if (!value || value.pluginId !== "pharma_oa") throw new Error("SKOLL_HOST_UNAVAILABLE");
  return value;
}

export function query(values: Record<string, string | number | undefined>): string {
  const params = new URLSearchParams();
  Object.entries(values).forEach(([key, value]) => {
    if (value !== undefined && String(value).trim() !== "") params.set(key, String(value));
  });
  const encoded = params.toString();
  return encoded ? `?${encoded}` : "";
}

export function idempotencyKey(prefix: string): string {
  const suffix = globalThis.crypto?.randomUUID?.() ?? `${Date.now().toString(36)}${Math.random().toString(36).slice(2)}`;
  return `${prefix}-${suffix}`;
}

function request<T>(path: string, method: "GET" | "POST" | "PUT" = "GET", body?: unknown, mutation = false): Promise<T> {
  return host().request<T>(API_BASE + path, {
    method,
    body,
    headers: mutation ? { "Idempotency-Key": idempotencyKey(`ui-${method.toLowerCase()}`) } : undefined
  });
}

export const api = {
  list: <T extends WorkspaceRecord>(module: ModuleKey, keyword = "", status = "") =>
    request<Page<T>>(`${collections[module]}${query({ keyword, status, limit: 200 })}`, "GET"),
  create: <T extends WorkspaceRecord>(module: ModuleKey, body: Record<string, unknown>) =>
    request<{ item: T }>(collections[module], "POST", body, true),
  update: <T extends WorkspaceRecord>(module: ModuleKey, id: string, body: Record<string, unknown>) =>
    request<{ item: T }>(`${collections[module]}/${id}`, "PUT", body, true),
  transition: <T extends WorkspaceRecord>(module: ModuleKey, id: string, action: string, body: Record<string, unknown>) =>
    request<{ item: T }>(`${collections[module]}/${id}/${action}`, "POST", body, true),
  employeeReminders: () => request<Page<Employee>>("/employees/qualification-reminders?days=30"),
  attachEmployee: (id: string, body: Record<string, unknown>) => request<{ item: Employee }>(`/employees/${id}/attachments`, "POST", body, true),
  categories: () => request<Page<Catalog>>("/categories?status=active&limit=200"),
  units: () => request<Page<Catalog>>("/units?status=active&limit=200"),
  manufacturers: () => request<Page<Catalog>>("/manufacturers?status=active&limit=200"),
  qualificationTypes: () => request<Page<QualificationType>>("/qualification-types?status=active&limit=200"),
  scanQualificationExpiry: (days = 30) => request<{ item?: unknown; items?: Qualification[] }>("/qualifications/expiry-scan", "POST", { days }, true)
};

export type ListTypes = Employee | Party | Catalog | Product | Qualification | QualificationType;
