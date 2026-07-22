import type {
  Asset,
  Dashboard,
  Inspection,
  Job,
  MaintenancePlan,
  Page,
  SpareMovement,
  SparePart,
  Trend,
  WorkOrder
} from "./types";

const base = "/v1/plugins/equipment_maintenance/api";

function host() {
  const value = window.__SKOLL_HOST__;
  if (!value || value.pluginId !== "equipment_maintenance") {
    throw new Error("Skoll host context is unavailable");
  }
  return value;
}

function query(values: Record<string, string | number | undefined>): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(values)) {
    if (value !== undefined && String(value).trim() !== "") params.set(key, String(value));
  }
  const encoded = params.toString();
  return encoded ? `?${encoded}` : "";
}

const get = <T>(path: string) => host().request<T>(base + path);
const post = <T>(path: string, body: unknown = {}) => host().request<T>(base + path, { method: "POST", body });
const put = <T>(path: string, body: unknown) => host().request<T>(base + path, { method: "PUT", body });

export const api = {
  dashboard: () => get<Dashboard>("/dashboard"),
  trends: () => get<{ items: Trend[] }>("/dashboard/trends"),
  assets: (keyword = "") => get<Page<Asset>>(`/assets${query({ keyword, limit: 200 })}`),
  createAsset: (body: Record<string, unknown>) => post<{ item: Asset }>("/assets", body),
  updateAsset: (id: string, body: Record<string, unknown>) => put<{ item: Asset }>(`/assets/${id}`, body),
  retireAsset: (id: string) => post<{ item: Asset }>(`/assets/${id}/retire`),
  workOrders: (keyword = "") => get<Page<WorkOrder>>(`/work-orders${query({ keyword, limit: 200 })}`),
  createWorkOrder: (body: Record<string, unknown>) => post<{ item: WorkOrder }>("/work-orders", body),
  transitionWorkOrder: (id: string, action: string, body: Record<string, unknown> = {}) => post<{ item: WorkOrder }>(`/work-orders/${id}/${action}`, body),
  plans: () => get<Page<MaintenancePlan>>("/maintenance-plans?limit=200"),
  createPlan: (body: Record<string, unknown>) => post<{ item: MaintenancePlan }>("/maintenance-plans", body),
  inspections: () => get<Page<Inspection>>("/inspections?limit=200"),
  createInspection: (body: Record<string, unknown>) => post<{ item: Inspection }>("/inspections", body),
  completeInspection: (id: string, body: Record<string, unknown>) => post<{ item: Inspection }>(`/inspections/${id}/complete`, body),
  spareParts: (keyword = "") => get<Page<SparePart>>(`/spare-parts${query({ keyword, limit: 200 })}`),
  createSparePart: (body: Record<string, unknown>) => post<{ item: SparePart }>("/spare-parts", body),
  movements: () => get<Page<SpareMovement>>("/spare-movements?limit=200"),
  createMovement: (body: Record<string, unknown>) => post<{ item: SpareMovement; sparePart: SparePart; duplicate: boolean }>("/spare-movements", body),
  jobs: () => get<{ items: Job[] }>("/jobs"),
  scheduleJob: (kind: "maintenance-due-scan" | "spare-stock-scan", body: Record<string, unknown>) => post<{ item: Job }>(`/jobs/${kind}`, body)
};
