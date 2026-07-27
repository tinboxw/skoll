export type Scope = {
  tenantId: string;
  organizationId: string;
  ownerId: string;
};

export type Asset = Scope & {
  id: string;
  code: string;
  name: string;
  category: string;
  model: string;
  serialNumber: string;
  location: string;
  status: string;
  nextMaintenanceAt?: string;
  createdAt: string;
  updatedAt: string;
};

export type WorkOrder = Scope & {
  id: string;
  assetId: string;
  number: string;
  title: string;
  priority: string;
  status: string;
  assigneeId?: string;
  workflowInstanceId?: string;
  estimatedCost: number;
  startedAt?: string;
  completedAt?: string;
  closedAt?: string;
  createdAt: string;
  updatedAt: string;
};

export type MaintenancePlan = Scope & {
  id: string;
  assetId: string;
  name: string;
  intervalDays: number;
  nextRunAt: string;
  status: string;
  createdAt: string;
  updatedAt: string;
};

export type Inspection = Scope & {
  id: string;
  planId: string;
  assetId: string;
  workOrderId?: string;
  status: string;
  result?: string;
  finding?: string;
  attachmentIds: string[];
  inspectedAt: string;
  createdAt: string;
  updatedAt: string;
};

export type SparePart = Scope & {
  id: string;
  sku: string;
  name: string;
  unit: string;
  quantity: number;
  minimumQuantity: number;
  status: string;
  createdAt: string;
  updatedAt: string;
};

export type SpareMovement = Scope & {
  id: string;
  sparePartId: string;
  workOrderId?: string;
  movementType: string;
  quantity: number;
  idempotencyKey: string;
  occurredAt: string;
  createdAt: string;
};

export type Job = {
  id: string;
  kind: string;
  status: string;
  attempts?: number;
  runAt?: string;
  updatedAt?: string;
};

export type Page<T> = { items: T[]; total: number; offset: number; limit: number };
export type Dashboard = {
  metrics: Record<"assets" | "openWorkOrders" | "overduePlans" | "lowStockParts" | "completedInspections", number>;
  generatedAt: string;
};
export type Trend = { month: string; closedWorkOrders: number; inspections: number };
