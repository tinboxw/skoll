export type Locale = "zh-CN" | "en-US";
export type ModuleKey = "employee" | "customer" | "supplier" | "product" | "category" | "unit" | "manufacturer" | "qualification" | "qualificationType";
export type RecordStatus = "active" | "disabled" | "on_leave" | "left" | "draft" | "pending" | "approved" | "rejected" | "revoked" | "expired";

export type Page<T> = { items: T[]; total?: number; pageInfo?: { nextCursor?: string; hasMore?: boolean; limit?: number } };
export type Scope = { tenantId: string; organizationId: string };
export type Certificate = { id: string; name: string; number: string; expiresAt: string };
export type Attachment = { id: string; name: string; size?: number };

export type Employee = {
  id: string;
  code: string;
  name: string;
  departmentId: string;
  positionId: string;
  employmentStatus: RecordStatus;
  phone: string;
  email: string;
  hireDate: string;
  leftAt?: string;
  leaveReason?: string;
  attachments: Attachment[];
  certificates: Certificate[];
  version: number;
};

export type Contact = { id: string; name: string; title: string; phone: string; email: string; primary: boolean };
export type Address = { id: string; label: string; province: string; city: string; district: string; detail: string; default: boolean };
export type SettlementTerms = { currency: string; paymentDays: number; creditLimit: number };
export type Party = {
  id: string;
  type: "customer" | "supplier";
  code: string;
  name: string;
  unifiedSocialCreditCode: string;
  region: string;
  rating: number;
  status: RecordStatus;
  disableReason?: string;
  contacts: Contact[];
  addresses: Address[];
  settlementTerms: SettlementTerms;
  version: number;
};

export type Catalog = {
  id: string;
  type: "category" | "unit" | "manufacturer";
  code: string;
  name: string;
  description: string;
  parentId?: string;
  symbol?: string;
  decimalPlaces?: number;
  unifiedSocialCreditCode?: string;
  licenseNumber?: string;
  status: RecordStatus;
  disableReason?: string;
  version: number;
};

export type Product = {
  id: string;
  code: string;
  sku: string;
  name: string;
  genericName: string;
  categoryId: string;
  unitId: string;
  manufacturerId: string;
  dosageForm: string;
  specification: string;
  approvalNumber: string;
  barcode?: string;
  storageCondition: string;
  temperatureMin: number;
  temperatureMax: number;
  status: RecordStatus;
  disableReason?: string;
  version: number;
};

export type QualificationType = {
  id: string;
  code: string;
  name: string;
  subjectType: string;
  businessGate: string;
  description: string;
  validityDays: number;
  alertDays: number;
  evidenceRequired: boolean;
  businessRequired: boolean;
  status: RecordStatus;
  disableReason?: string;
  version: number;
};

export type Qualification = {
  id: string;
  typeId: string;
  subjectType: string;
  subjectId: string;
  certificateNumber: string;
  issuer: string;
  validFrom: string;
  validTo: string;
  status: RecordStatus;
  evidenceFileId?: string;
  evidenceFileName?: string;
  reviewComment?: string;
  reviewedBy?: string;
  version: number;
};

export type WorkspaceRecord = Employee | Party | Catalog | Product | Qualification | QualificationType;

export type OAWorkspaceMode = "requests" | "inbox";
export type OARequestType = "leave" | "expense" | "procurement" | "contract" | "custom";
export type OARequestStatus = "draft" | "pending" | "approved" | "rejected" | "withdrawn" | "canceled";

export type OARequestAttachment = {
  fileId: string;
  name: string;
  size: number;
  mime: string;
  requestKey: string;
};

export type OARequestComment = {
  id: string;
  actorId: string;
  content: string;
  createdAt: string;
  requestKey: string;
};

export type OARequest = {
  id: string;
  requestType: OARequestType;
  title: string;
  description: string;
  formData: Record<string, unknown>;
  status: OARequestStatus;
  approverId: string;
  approverName: string;
  workflowDefinitionId?: string;
  workflowInstanceId?: string;
  submittedAt?: string;
  reminderAt?: string;
  attachments: OARequestAttachment[];
  comments: OARequestComment[];
  version: number;
  createdAt: string;
  updatedAt: string;
};

export type OAWorkflowActor = { id: string; name: string };
export type OAWorkflowTask = {
  id: string;
  instanceId: string;
  nodeId: string;
  assignee: OAWorkflowActor;
  status: "pending" | "approved" | "rejected" | "delegated" | "copied" | "canceled";
  createdAt: string;
  completedAt?: string;
};
export type OAWorkflowAction = {
  id: string;
  type: "start" | "approve" | "reject" | "withdraw" | "delegate" | "substitute" | "escalate" | "copy" | "cancel";
  instanceId: string;
  taskId: string;
  nodeId: string;
  actor: OAWorkflowActor;
  target: OAWorkflowActor;
  comment: string;
  createdAt: string;
};
export type OAWorkflow = {
  id: string;
  definitionId: string;
  definitionKey: string;
  businessType: string;
  businessId: string;
  title: string;
  status: "running" | "approved" | "rejected" | "withdrawn" | "canceled";
  starter: OAWorkflowActor;
  currentNode: string;
  tasks: OAWorkflowTask[];
  timeline: OAWorkflowAction[];
  createdAt: string;
  updatedAt: string;
};

export type OARequestWriteInput = Scope & {
  requestType: OARequestType;
  title: string;
  description: string;
  formData: Record<string, unknown>;
  approverId: string;
  approverName: string;
  version?: number;
};

export type OARequestDetail = { item: OARequest; workflow?: OAWorkflow };
export type OARequestMutation = { item: OARequest; workflow?: OAWorkflow; duplicate?: boolean };

export type PurchaseLine = {
  id: string;
  productId: string;
  productCode: string;
  productName: string;
  specification: string;
  unitId: string;
  quantity: string;
  unitPrice: string;
  amount: string;
  receivedQuantity: string;
};

export type PurchaseRequest = {
  id: string;
  number: string;
  supplierId: string;
  supplierCode: string;
  supplierName: string;
  requesterId: string;
  approverId: string;
  approverName: string;
  reason: string;
  currency: string;
  lines: PurchaseLine[];
  totalAmount: string;
  status: "pending" | "approved" | "rejected";
  workflowDefinitionId: string;
  workflowInstanceId: string;
  purchaseOrderId?: string;
  decisionComment?: string;
  decidedAt?: string;
  version: number;
  createdAt: string;
  updatedAt: string;
};

export type PurchaseOrder = {
  id: string;
  number: string;
  purchaseRequestId: string;
  supplierId: string;
  supplierCode: string;
  supplierName: string;
  currency: string;
  lines: PurchaseLine[];
  totalAmount: string;
  status: "open" | "partial" | "received";
  approvedBy: string;
  approvedAt: string;
  version: number;
  createdAt: string;
  updatedAt: string;
};

export type PurchaseInboundLine = {
  orderLineId: string;
  productId: string;
  productCode: string;
  productName: string;
  quantity: string;
  batchNo: string;
  productionDate: string;
  expiresAt: string;
};

export type PurchaseInbound = {
  id: string;
  number: string;
  purchaseOrderId: string;
  purchaseOrderNumber: string;
  warehouseId: string;
  areaId: string;
  locationId: string;
  lines: PurchaseInboundLine[];
  attachments: Array<{ fileId: string; name: string; size: number; mime: string }>;
  status: "completed";
  receivedBy: string;
  receivedAt: string;
  version: number;
  createdAt: string;
  updatedAt: string;
};

export type PurchaseRequestWriteInput = Scope & {
  supplierId: string;
  reason: string;
  currency: "CNY";
  approverId: string;
  approverName: string;
  lines: Array<{ productId: string; quantity: string; unitPrice: string }>;
};

export type PurchaseInboundWriteInput = Scope & {
  purchaseOrderId: string;
  warehouseId: string;
  areaId: string;
  locationId: string;
  orderVersion: number;
  lines: Array<{ orderLineId: string; quantity: string; batchNo: string; productionDate: string; expiresAt: string }>;
  attachments: Array<{ name: string; contentBase64: string }>;
};
