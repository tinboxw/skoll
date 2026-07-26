<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
import { ElMessage } from "element-plus/es/components/message/index.mjs";
import {
  Check,
  CircleAlert,
  ClipboardList,
  Eye,
  FileUp,
  PackageCheck,
  Plus,
  RefreshCw,
  Search,
  ShoppingCart,
  Trash2,
  X
} from "lucide-vue-next";
import { api } from "../api";
import { t } from "../i18n";
import type {
  OAWorkflow,
  Party,
  Product,
  PurchaseInbound,
  PurchaseOrder,
  PurchaseRequest,
  Scope
} from "../types";

type PurchaseView = "requests" | "orders" | "inbounds";
type Decision = "approve" | "reject";
type RequestLineDraft = { key: string; productId: string; quantity: string; unitPrice: string };
type InboundLineDraft = {
  orderLineId: string;
  productName: string;
  ordered: string;
  received: string;
  quantity: string;
  batchNo: string;
  productionDate: string;
  expiresAt: string;
};

const props = defineProps<{ scope: Scope }>();
const view = ref<PurchaseView>("requests");
const loading = ref(false);
const saving = ref(false);
const error = ref("");
const keyword = ref("");
const status = ref("");
const requests = ref<PurchaseRequest[]>([]);
const orders = ref<PurchaseOrder[]>([]);
const inbounds = ref<PurchaseInbound[]>([]);
const suppliers = ref<Party[]>([]);
const products = ref<Product[]>([]);
const requestEditorOpen = ref(false);
const requestDetailOpen = ref(false);
const orderDetailOpen = ref(false);
const inboundDetailOpen = ref(false);
const inboundEditorOpen = ref(false);
const decision = ref<Decision | null>(null);
const selectedRequest = ref<PurchaseRequest | null>(null);
const selectedOrder = ref<PurchaseOrder | null>(null);
const selectedInbound = ref<PurchaseInbound | null>(null);
const requestWorkflow = ref<OAWorkflow | null>(null);
const decisionComment = ref("");
const evidenceInput = ref<HTMLInputElement | null>(null);
const requestForm = reactive({
  supplierId: "",
  reason: "",
  approverId: "",
  approverName: "",
  lines: [] as RequestLineDraft[]
});
const inboundForm = reactive({
  warehouseId: "",
  areaId: "",
  locationId: "",
  lines: [] as InboundLineDraft[],
  attachments: [] as Array<{ key: string; name: string; contentBase64: string }>
});

const visibleItems = computed(() => view.value === "requests" ? requests.value : view.value === "orders" ? orders.value : inbounds.value);
const pendingCount = computed(() => requests.value.filter((item) => item.status === "pending").length);
const openCount = computed(() => orders.value.filter((item) => item.status === "open").length);
const partialCount = computed(() => orders.value.filter((item) => item.status === "partial").length);
const completedCount = computed(() => orders.value.filter((item) => item.status === "received").length);
const currentTask = computed(() => requestWorkflow.value?.tasks.find((task) => task.status === "pending"));
const requestTotal = computed(() => requestForm.lines.reduce((total, line) => total + decimal(line.quantity) * decimal(line.unitPrice), 0));
const decisionOpen = computed({
  get: () => decision.value !== null,
  set: (open: boolean) => { if (!open) decision.value = null; }
});
const viewOptions = computed(() => [
  { value: "requests" as const, label: t("purchaseRequests"), icon: ClipboardList },
  { value: "orders" as const, label: t("purchaseOrders"), icon: ShoppingCart },
  { value: "inbounds" as const, label: t("inboundRecords"), icon: PackageCheck }
]);

onMounted(() => void loadAll());
watch(() => props.scope, () => void loadAll(), { deep: true });
watch(view, () => {
  keyword.value = "";
  status.value = "";
  error.value = "";
});

async function loadAll(): Promise<void> {
  if (loading.value) return;
  loading.value = true;
  error.value = "";
  try {
    const [requestPage, orderPage, inboundPage] = await Promise.all([
      api.purchaseRequests(keyword.value.trim(), view.value === "requests" ? status.value : ""),
      api.purchaseOrders(keyword.value.trim(), view.value === "orders" ? status.value : ""),
      api.purchaseInbounds()
    ]);
    requests.value = requestPage.items ?? [];
    orders.value = orderPage.items ?? [];
    inbounds.value = inboundPage.items ?? [];
  } catch (reason) {
    error.value = mutationError(reason);
  } finally {
    loading.value = false;
  }
}

async function applyFilters(): Promise<void> {
  await loadAll();
}

async function openRequestEditor(): Promise<void> {
  saving.value = true;
  try {
    const [supplierPage, productPage] = await Promise.all([
      api.list<Party>("supplier", "", "active"),
      api.list<Product>("product", "", "active")
    ]);
    suppliers.value = supplierPage.items ?? [];
    products.value = productPage.items ?? [];
    Object.assign(requestForm, { supplierId: "", reason: "", approverId: "", approverName: "", lines: [newRequestLine()] });
    requestEditorOpen.value = true;
  } catch (reason) {
    ElMessage.error(mutationError(reason));
  } finally {
    saving.value = false;
  }
}

function newRequestLine(): RequestLineDraft {
  return { key: globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random()}`, productId: "", quantity: "", unitPrice: "" };
}

function addRequestLine(): void {
  requestForm.lines.push(newRequestLine());
}

function removeRequestLine(index: number): void {
  if (requestForm.lines.length > 1) requestForm.lines.splice(index, 1);
}

function requestValid(): boolean {
  return Boolean(
    props.scope.tenantId.trim() &&
    props.scope.organizationId.trim() &&
    requestForm.supplierId &&
    requestForm.reason.trim() &&
    requestForm.approverId.trim() &&
    requestForm.approverName.trim() &&
    requestForm.lines.length &&
    requestForm.lines.every((line) => line.productId && positiveDecimal(line.quantity, 6) && positiveDecimal(line.unitPrice, 2))
  );
}

async function createRequest(): Promise<void> {
  if (!requestValid()) {
    ElMessage.warning(t("purchaseRequired"));
    return;
  }
  saving.value = true;
  try {
    await api.createPurchaseRequest({
      ...props.scope,
      supplierId: requestForm.supplierId,
      reason: requestForm.reason.trim(),
      currency: "CNY",
      approverId: requestForm.approverId.trim(),
      approverName: requestForm.approverName.trim(),
      lines: requestForm.lines.map(({ productId, quantity, unitPrice }) => ({ productId, quantity, unitPrice }))
    });
    requestEditorOpen.value = false;
    ElMessage.success(t("purchaseCreated"));
    await loadAll();
  } catch (reason) {
    ElMessage.error(mutationError(reason));
  } finally {
    saving.value = false;
  }
}

async function openRequest(item: PurchaseRequest): Promise<void> {
  selectedRequest.value = item;
  requestDetailOpen.value = true;
  try {
    const detail = await api.purchaseRequest(item.id);
    selectedRequest.value = detail.item;
    requestWorkflow.value = detail.workflow;
  } catch (reason) {
    ElMessage.error(mutationError(reason));
  }
}

async function openDecision(kind: Decision, item?: PurchaseRequest): Promise<void> {
  if (item) await openRequest(item);
  decisionComment.value = "";
  decision.value = kind;
}

async function runDecision(): Promise<void> {
  const item = selectedRequest.value;
  if (!item || !currentTask.value || !decision.value || (decision.value === "reject" && !decisionComment.value.trim())) {
    ElMessage.warning(t("formRequired"));
    return;
  }
  saving.value = true;
  try {
    if (decision.value === "approve") {
      await api.approvePurchaseRequest(item.id, currentTask.value.id, item.version, decisionComment.value.trim());
      ElMessage.success(t("purchaseApproved"));
    } else {
      await api.rejectPurchaseRequest(item.id, currentTask.value.id, item.version, decisionComment.value.trim());
      ElMessage.success(t("purchaseRejected"));
    }
    decision.value = null;
    requestDetailOpen.value = false;
    await loadAll();
  } catch (reason) {
    ElMessage.error(mutationError(reason));
  } finally {
    saving.value = false;
  }
}

async function openOrder(item: PurchaseOrder): Promise<void> {
  selectedOrder.value = item;
  orderDetailOpen.value = true;
  try {
    const [detail, history] = await Promise.all([api.purchaseOrder(item.id), api.purchaseInbounds(item.id)]);
    selectedOrder.value = detail.item;
    inbounds.value = [...history.items, ...inbounds.value.filter((entry) => entry.purchaseOrderId !== item.id)];
  } catch (reason) {
    ElMessage.error(mutationError(reason));
  }
}

function openInboundEditor(order?: PurchaseOrder): void {
  const item = order ?? selectedOrder.value;
  if (!item || item.status === "received") return;
  selectedOrder.value = item;
  Object.assign(inboundForm, {
    warehouseId: "",
    areaId: "",
    locationId: "",
    attachments: [],
    lines: item.lines.filter((line) => decimal(line.receivedQuantity) < decimal(line.quantity)).map((line) => ({
      orderLineId: line.id,
      productName: `${line.productCode} · ${line.productName}`,
      ordered: line.quantity,
      received: line.receivedQuantity,
      quantity: "",
      batchNo: "",
      productionDate: "",
      expiresAt: ""
    }))
  });
  inboundEditorOpen.value = true;
}

async function readEvidence(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement;
  const files = [...(input.files ?? [])].slice(0, 10);
  input.value = "";
  if (!files.length) return;
  if (files.some((file) => file.size > 5 * 1024 * 1024)) {
    ElMessage.error(t("fileLimit"));
    return;
  }
  inboundForm.attachments = await Promise.all(files.map(async (file) => ({
    key: globalThis.crypto?.randomUUID?.() ?? `${file.name}-${file.size}`,
    name: file.name,
    contentBase64: await readBase64(file)
  })));
}

function inboundValid(): boolean {
  const selected = inboundForm.lines.filter((line) => line.quantity.trim());
  return Boolean(
    props.scope.tenantId.trim() &&
    props.scope.organizationId.trim() &&
    inboundForm.warehouseId.trim() &&
    inboundForm.areaId.trim() &&
    inboundForm.locationId.trim() &&
    selected.length &&
    selected.every((line) => positiveDecimal(line.quantity, 6) && line.batchNo.trim() && line.productionDate && line.expiresAt && line.expiresAt > line.productionDate)
  );
}

async function createInbound(): Promise<void> {
  const order = selectedOrder.value;
  if (!order || !inboundValid()) {
    ElMessage.warning(t("inboundRequired"));
    return;
  }
  saving.value = true;
  try {
    await api.createPurchaseInbound({
      ...props.scope,
      purchaseOrderId: order.id,
      warehouseId: inboundForm.warehouseId.trim(),
      areaId: inboundForm.areaId.trim(),
      locationId: inboundForm.locationId.trim(),
      orderVersion: order.version,
      lines: inboundForm.lines.filter((line) => line.quantity.trim()).map((line) => ({
        orderLineId: line.orderLineId,
        quantity: line.quantity,
        batchNo: line.batchNo.trim(),
        productionDate: line.productionDate,
        expiresAt: line.expiresAt
      })),
      attachments: inboundForm.attachments.map(({ name, contentBase64 }) => ({ name, contentBase64 }))
    });
    inboundEditorOpen.value = false;
    orderDetailOpen.value = false;
    ElMessage.success(t("inboundCreated"));
    await loadAll();
  } catch (reason) {
    ElMessage.error(mutationError(reason));
  } finally {
    saving.value = false;
  }
}

function positiveDecimal(value: string, scale: number): boolean {
  const match = value.trim().match(/^(0|[1-9]\d{0,11})(?:\.(\d+))?$/);
  return Boolean(match && (match[2]?.length ?? 0) <= scale && decimal(value) > 0);
}

function decimal(value: string): number {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

function money(value: string | number): string {
  return new Intl.NumberFormat(undefined, { style: "currency", currency: "CNY" }).format(decimal(String(value)));
}

function remaining(orderLine: { quantity: string; receivedQuantity: string }): string {
  return Math.max(0, decimal(orderLine.quantity) - decimal(orderLine.receivedQuantity)).toFixed(6).replace(/\.?0+$/, "");
}

function statusLabel(value: string): string {
  const labels: Record<string, ReturnType<typeof t>> = {
    pending: t("pending"), approved: t("approved"), rejected: t("rejected"),
    open: t("open"), partial: t("partial"), received: t("received"), completed: t("completed")
  };
  return labels[value] ?? value;
}

function statusType(value: string): "success" | "warning" | "danger" | "info" {
  if (["approved", "received", "completed"].includes(value)) return "success";
  if (["pending", "open", "partial"].includes(value)) return "warning";
  if (value === "rejected") return "danger";
  return "info";
}

function mutationError(reason: unknown): string {
  const message = reason instanceof Error ? reason.message : t("failed");
  if (/qualification|supplier.*inactive|product.*inactive|manufacturer.*inactive|unprocessable/i.test(message)) return t("qualificationBlocked");
  if (/stale|version|conflict/i.test(message)) return t("staleConflict");
  return message;
}

function formatTime(value?: string): string {
  if (!value) return "-";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

async function openInbound(item: PurchaseInbound): Promise<void> {
  selectedInbound.value = item;
  inboundDetailOpen.value = true;
  try {
    selectedInbound.value = (await api.purchaseInbound(item.id)).item;
  } catch (reason) {
    ElMessage.error(mutationError(reason));
  }
}

function readBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result).split(",")[1] ?? "");
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(file);
  });
}
</script>

<template>
  <section class="purchase-workspace">
    <header class="module-heading oa-heading">
      <div>
        <p>{{ t("brand") }} · {{ t("purchaseWorkspace") }}</p>
        <h2>{{ t("purchaseTitle") }}</h2>
        <span>{{ t("purchaseDescription") }}</span>
      </div>
      <div class="heading-actions">
        <el-tooltip :content="t('refresh')"><el-button :icon="RefreshCw" circle :loading="loading" :aria-label="t('refresh')" @click="loadAll" /></el-tooltip>
        <el-button type="primary" :icon="Plus" @click="openRequestEditor">{{ t("newPurchaseRequest") }}</el-button>
      </div>
    </header>

    <section class="metrics purchase-metrics" :aria-label="t('purchaseTitle')">
      <article><span>{{ t("pendingApproval") }}</span><strong>{{ pendingCount }}</strong><ClipboardList /></article>
      <article><span>{{ t("openOrders") }}</span><strong>{{ openCount }}</strong><ShoppingCart /></article>
      <article><span>{{ t("partialOrders") }}</span><strong>{{ partialCount }}</strong><PackageCheck /></article>
      <article><span>{{ t("completedOrders") }}</span><strong>{{ completedCount }}</strong><Check /></article>
    </section>

    <nav class="purchase-switch" :aria-label="t('purchaseWorkspace')">
      <button v-for="option in viewOptions" :key="option.value" type="button" :class="{ active: view === option.value }" @click="view = option.value">
        <component :is="option.icon" aria-hidden="true" />{{ option.label }}
      </button>
    </nav>

    <section class="toolbar purchase-toolbar">
      <el-input v-model="keyword" :prefix-icon="Search" :placeholder="t('searchPurchase')" clearable @keyup.enter="applyFilters" />
      <el-select v-if="view !== 'inbounds'" v-model="status" :placeholder="t('allPurchaseStatuses')" clearable @change="applyFilters">
        <template v-if="view === 'requests'">
          <el-option :label="t('pending')" value="pending" /><el-option :label="t('approved')" value="approved" /><el-option :label="t('rejected')" value="rejected" />
        </template>
        <template v-else>
          <el-option :label="t('open')" value="open" /><el-option :label="t('partial')" value="partial" /><el-option :label="t('received')" value="received" />
        </template>
      </el-select>
      <el-button @click="applyFilters">{{ t("search") }}</el-button>
    </section>

    <section v-if="loading" class="state-panel" aria-live="polite"><el-skeleton :rows="6" animated /><span>{{ t("loading") }}</span></section>
    <section v-else-if="error && visibleItems.length === 0" class="state-panel state-error" role="alert"><CircleAlert /><strong>{{ t("failed") }}</strong><p>{{ error }}</p><el-button @click="loadAll">{{ t("retry") }}</el-button></section>
    <section v-else-if="visibleItems.length === 0" class="state-panel">
      <el-empty :description="view === 'requests' ? t('noPurchaseRequests') : view === 'orders' ? t('noPurchaseOrders') : t('noInboundRecords')" />
      <el-button v-if="view === 'requests'" type="primary" :icon="Plus" @click="openRequestEditor">{{ t("newPurchaseRequest") }}</el-button>
    </section>

    <section v-else class="data-region purchase-data-region">
      <el-table v-if="view === 'requests'" :data="requests" stripe class="desktop-table" row-key="id" @row-dblclick="openRequest">
        <el-table-column :label="t('purchaseNumber')" min-width="220"><template #default="{ row }"><button class="identity-link" type="button" @click="openRequest(row)"><strong>{{ row.number }}</strong><span>{{ row.supplierCode }} · {{ row.supplierName }}</span></button></template></el-table-column>
        <el-table-column :label="t('purchaseReason')" min-width="240" prop="reason" />
        <el-table-column :label="t('approver')" min-width="150" prop="approverName" />
        <el-table-column :label="t('purchaseTotal')" width="150"><template #default="{ row }">{{ money(row.totalAmount) }}</template></el-table-column>
        <el-table-column :label="t('status')" width="120"><template #default="{ row }"><el-tag :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag></template></el-table-column>
        <el-table-column :label="t('actions')" width="180" fixed="right"><template #default="{ row }"><el-button :icon="Eye" text circle :aria-label="t('detail')" @click="openRequest(row)" /><el-button v-if="row.status === 'pending'" :icon="Check" text circle type="success" :aria-label="t('approve')" @click="openDecision('approve', row)" /><el-button v-if="row.status === 'pending'" :icon="X" text circle type="danger" :aria-label="t('reject')" @click="openDecision('reject', row)" /></template></el-table-column>
      </el-table>

      <el-table v-else-if="view === 'orders'" :data="orders" stripe class="desktop-table" row-key="id" @row-dblclick="openOrder">
        <el-table-column :label="t('purchaseNumber')" min-width="220"><template #default="{ row }"><button class="identity-link" type="button" @click="openOrder(row)"><strong>{{ row.number }}</strong><span>{{ row.supplierCode }} · {{ row.supplierName }}</span></button></template></el-table-column>
        <el-table-column :label="t('purchaseProgress')" min-width="250"><template #default="{ row }">{{ row.lines.map((line: PurchaseOrder['lines'][number]) => `${line.productName} ${line.receivedQuantity}/${line.quantity}`).join("；") }}</template></el-table-column>
        <el-table-column :label="t('purchaseTotal')" width="150"><template #default="{ row }">{{ money(row.totalAmount) }}</template></el-table-column>
        <el-table-column :label="t('status')" width="120"><template #default="{ row }"><el-tag :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag></template></el-table-column>
        <el-table-column :label="t('actions')" width="150" fixed="right"><template #default="{ row }"><el-button :icon="Eye" text circle :aria-label="t('detail')" @click="openOrder(row)" /><el-button v-if="row.status !== 'received'" type="primary" text @click="openInboundEditor(row)">{{ t("receiveGoods") }}</el-button></template></el-table-column>
      </el-table>

      <el-table v-else :data="inbounds" stripe class="desktop-table" row-key="id" @row-dblclick="openInbound">
        <el-table-column :label="t('purchaseNumber')" min-width="210"><template #default="{ row }"><button class="identity-link" type="button" @click="openInbound(row)"><strong>{{ row.number }}</strong><span>{{ row.purchaseOrderNumber }}</span></button></template></el-table-column>
        <el-table-column :label="t('warehouseLocation')" min-width="210"><template #default="{ row }">{{ row.warehouseId }} / {{ row.areaId }} / {{ row.locationId }}</template></el-table-column>
        <el-table-column :label="t('productLines')" min-width="260"><template #default="{ row }">{{ row.lines.map((line: PurchaseInbound['lines'][number]) => `${line.productName} · ${line.batchNo} · ${line.quantity}`).join("；") }}</template></el-table-column>
        <el-table-column :label="t('createdTime')" width="180"><template #default="{ row }">{{ formatTime(row.receivedAt) }}</template></el-table-column>
        <el-table-column :label="t('status')" width="110"><template #default="{ row }"><el-tag type="success">{{ statusLabel(row.status) }}</el-tag></template></el-table-column>
        <el-table-column :label="t('actions')" width="90" fixed="right"><template #default="{ row }"><el-button :icon="Eye" text circle :aria-label="t('detail')" @click="openInbound(row)" /></template></el-table-column>
      </el-table>

      <div class="mobile-records purchase-mobile-records">
        <article v-for="item in visibleItems" :key="item.id" class="record-card">
          <button type="button" @click="view === 'requests' ? openRequest(item as PurchaseRequest) : view === 'orders' ? openOrder(item as PurchaseOrder) : openInbound(item as PurchaseInbound)"><strong>{{ item.number }}</strong><span>{{ statusLabel(item.status) }}</span></button>
          <p>{{ "supplierName" in item ? item.supplierName : item.purchaseOrderNumber }}</p>
          <footer v-if="view !== 'inbounds'"><el-button :icon="Eye" @click="view === 'requests' ? openRequest(item as PurchaseRequest) : openOrder(item as PurchaseOrder)">{{ t("detail") }}</el-button><el-button v-if="view === 'orders' && (item as PurchaseOrder).status !== 'received'" type="primary" @click="openInboundEditor(item as PurchaseOrder)">{{ t("receiveGoods") }}</el-button></footer>
        </article>
      </div>
    </section>

    <el-drawer v-model="requestEditorOpen" :title="t('newPurchaseRequest')" size="min(920px, 100vw)" destroy-on-close>
      <el-form label-position="top" class="purchase-form">
        <div class="purchase-form-grid">
          <el-form-item :label="t('supplier')" required><el-select v-model="requestForm.supplierId" filterable><el-option v-for="item in suppliers" :key="item.id" :label="`${item.code} · ${item.name}`" :value="item.id" /></el-select></el-form-item>
          <el-form-item :label="t('currencyCNY')"><el-input model-value="CNY" disabled /></el-form-item>
          <el-form-item :label="t('approverId')" required><el-input v-model="requestForm.approverId" maxlength="128" /></el-form-item>
          <el-form-item :label="t('approverName')" required><el-input v-model="requestForm.approverName" maxlength="200" /></el-form-item>
          <el-form-item class="wide" :label="t('purchaseReason')" required><el-input v-model="requestForm.reason" type="textarea" :rows="3" maxlength="1000" show-word-limit /></el-form-item>
        </div>
        <div class="purchase-section-title"><h3>{{ t("productLines") }}</h3><el-button :icon="Plus" @click="addRequestLine">{{ t("addLine") }}</el-button></div>
        <div class="purchase-line-grid purchase-line-head"><span>{{ t("product") }}</span><span>{{ t("quantity") }}</span><span>{{ t("unitPrice") }}</span><span>{{ t("lineAmount") }}</span><span /></div>
        <div v-for="(line, index) in requestForm.lines" :key="line.key" class="purchase-line-grid">
          <el-select v-model="line.productId" filterable><el-option v-for="item in products" :key="item.id" :label="`${item.code} · ${item.name} · ${item.specification}`" :value="item.id" /></el-select>
          <el-input v-model="line.quantity" inputmode="decimal" />
          <el-input v-model="line.unitPrice" inputmode="decimal" />
          <strong>{{ money(decimal(line.quantity) * decimal(line.unitPrice)) }}</strong>
          <el-tooltip :content="t('removeLine')"><el-button :icon="Trash2" text circle type="danger" :disabled="requestForm.lines.length === 1" :aria-label="t('removeLine')" @click="removeRequestLine(index)" /></el-tooltip>
        </div>
        <div class="purchase-total"><span>{{ t("purchaseTotal") }}</span><strong>{{ money(requestTotal) }}</strong></div>
      </el-form>
      <template #footer><el-button @click="requestEditorOpen = false">{{ t("cancel") }}</el-button><el-button type="primary" :loading="saving" @click="createRequest">{{ t("submitRequest") }}</el-button></template>
    </el-drawer>

    <el-drawer v-model="requestDetailOpen" :title="t('requestDetailTitle')" size="min(720px, 100vw)">
      <div v-if="selectedRequest" class="oa-detail">
        <header class="oa-detail-header"><div><span>{{ selectedRequest.number }}</span><h3>{{ selectedRequest.supplierName }}</h3></div><el-tag :type="statusType(selectedRequest.status)" size="large">{{ statusLabel(selectedRequest.status) }}</el-tag></header>
        <dl class="oa-detail-grid"><div><dt>{{ t("approver") }}</dt><dd>{{ selectedRequest.approverName }}</dd></div><div><dt>{{ t("purchaseTotal") }}</dt><dd>{{ money(selectedRequest.totalAmount) }}</dd></div><div class="wide"><dt>{{ t("purchaseReason") }}</dt><dd>{{ selectedRequest.reason }}</dd></div></dl>
        <section class="oa-detail-section"><h4>{{ t("productLines") }}</h4><ul class="purchase-detail-lines"><li v-for="line in selectedRequest.lines" :key="line.id"><span>{{ line.productCode }} · {{ line.productName }} · {{ line.specification }}</span><strong>{{ line.quantity }} × {{ money(line.unitPrice) }} = {{ money(line.amount) }}</strong></li></ul></section>
        <section v-if="selectedRequest.status === 'pending'" class="oa-action-band"><el-button type="success" :icon="Check" @click="openDecision('approve')">{{ t("approveRequest") }}</el-button><el-button type="danger" :icon="X" @click="openDecision('reject')">{{ t("rejectRequest") }}</el-button></section>
      </div>
    </el-drawer>

    <el-dialog v-model="decisionOpen" :title="decision === 'approve' ? t('approveRequest') : t('rejectRequest')" width="min(520px, calc(100vw - 28px))" align-center>
      <el-form label-position="top"><el-form-item :label="t('decisionComment')" :required="decision === 'reject'"><el-input v-model="decisionComment" type="textarea" :rows="4" maxlength="1000" show-word-limit /></el-form-item></el-form>
      <template #footer><el-button @click="decision = null">{{ t("cancel") }}</el-button><el-button :type="decision === 'reject' ? 'danger' : 'primary'" :loading="saving" @click="runDecision">{{ t("confirm") }}</el-button></template>
    </el-dialog>

    <el-drawer v-model="orderDetailOpen" :title="t('orderDetail')" size="min(760px, 100vw)">
      <div v-if="selectedOrder" class="oa-detail">
        <header class="oa-detail-header"><div><span>{{ selectedOrder.number }}</span><h3>{{ selectedOrder.supplierName }}</h3></div><el-tag :type="statusType(selectedOrder.status)" size="large">{{ statusLabel(selectedOrder.status) }}</el-tag></header>
        <section class="oa-detail-section"><h4>{{ t("purchaseProgress") }}</h4><ul class="purchase-detail-lines"><li v-for="line in selectedOrder.lines" :key="line.id"><span>{{ line.productCode }} · {{ line.productName }}</span><strong>{{ t("receivedTotal") }} {{ line.receivedQuantity }} / {{ t("orderedQuantity") }} {{ line.quantity }} · {{ t("remainingQuantity") }} {{ remaining(line) }}</strong></li></ul></section>
        <section class="oa-detail-section"><h4>{{ t("inboundHistory") }}</h4><p v-if="!inbounds.some((item) => item.purchaseOrderId === selectedOrder!.id)" class="muted-empty">{{ t("noInboundRecords") }}</p><ul v-else class="purchase-detail-lines"><li v-for="item in inbounds.filter((entry) => entry.purchaseOrderId === selectedOrder!.id)" :key="item.id"><span>{{ item.number }} · {{ item.locationId }}</span><strong>{{ formatTime(item.receivedAt) }}</strong></li></ul></section>
        <section v-if="selectedOrder.status !== 'received'" class="oa-action-band"><el-button type="primary" :icon="PackageCheck" @click="openInboundEditor()">{{ t("receiveGoods") }}</el-button></section>
      </div>
    </el-drawer>

    <el-drawer v-model="inboundEditorOpen" :title="t('receiveGoods')" size="min(980px, 100vw)" destroy-on-close>
      <el-form v-if="selectedOrder" label-position="top" class="purchase-form">
        <div class="purchase-form-grid location-grid">
          <el-form-item :label="t('warehouse')" required><el-input v-model="inboundForm.warehouseId" maxlength="128" /></el-form-item>
          <el-form-item :label="t('warehouseArea')" required><el-input v-model="inboundForm.areaId" maxlength="128" /></el-form-item>
          <el-form-item :label="t('warehouseLocation')" required><el-input v-model="inboundForm.locationId" maxlength="128" /></el-form-item>
        </div>
        <div class="purchase-line-grid inbound-line-head"><span>{{ t("product") }}</span><span>{{ t("remainingQuantity") }}</span><span>{{ t("receivedQuantity") }}</span><span>{{ t("batchNumber") }}</span><span>{{ t("productionDate") }}</span><span>{{ t("expiryDate") }}</span></div>
        <div v-for="line in inboundForm.lines" :key="line.orderLineId" class="purchase-line-grid inbound-line-grid">
          <strong>{{ line.productName }}</strong>
          <div class="mobile-line-field"><small>{{ t("remainingQuantity") }}</small><span>{{ decimal(line.ordered) - decimal(line.received) }}</span></div>
          <label class="mobile-line-field"><small>{{ t("receivedQuantity") }}</small><el-input v-model="line.quantity" inputmode="decimal" :aria-label="t('receivedQuantity')" /></label>
          <label class="mobile-line-field"><small>{{ t("batchNumber") }}</small><el-input v-model="line.batchNo" maxlength="64" :aria-label="t('batchNumber')" /></label>
          <label class="mobile-line-field"><small>{{ t("productionDate") }}</small><el-date-picker v-model="line.productionDate" type="date" value-format="YYYY-MM-DD" :aria-label="t('productionDate')" /></label>
          <label class="mobile-line-field"><small>{{ t("expiryDate") }}</small><el-date-picker v-model="line.expiresAt" type="date" value-format="YYYY-MM-DD" :aria-label="t('expiryDate')" /></label>
        </div>
        <label class="file-picker inbound-file"><FileUp /><span>{{ inboundForm.attachments.length ? inboundForm.attachments.map((item) => item.name).join("、") : t("inboundEvidence") }}</span><input ref="evidenceInput" type="file" multiple accept="application/pdf,image/jpeg,image/png" @change="readEvidence" /></label>
      </el-form>
      <template #footer><el-button @click="inboundEditorOpen = false">{{ t("cancel") }}</el-button><el-button type="primary" :icon="PackageCheck" :loading="saving" @click="createInbound">{{ t("submitInbound") }}</el-button></template>
    </el-drawer>

    <el-drawer v-model="inboundDetailOpen" :title="t('inboundRecords')" size="min(720px, 100vw)">
      <div v-if="selectedInbound" class="oa-detail">
        <header class="oa-detail-header"><div><span>{{ selectedInbound.purchaseOrderNumber }}</span><h3>{{ selectedInbound.number }}</h3></div><el-tag type="success" size="large">{{ statusLabel(selectedInbound.status) }}</el-tag></header>
        <dl class="oa-detail-grid">
          <div><dt>{{ t("warehouse") }}</dt><dd>{{ selectedInbound.warehouseId }}</dd></div><div><dt>{{ t("warehouseArea") }}</dt><dd>{{ selectedInbound.areaId }}</dd></div>
          <div><dt>{{ t("warehouseLocation") }}</dt><dd>{{ selectedInbound.locationId }}</dd></div><div><dt>{{ t("createdTime") }}</dt><dd>{{ formatTime(selectedInbound.receivedAt) }}</dd></div>
        </dl>
        <section class="oa-detail-section"><h4>{{ t("productLines") }}</h4><ul class="purchase-detail-lines"><li v-for="line in selectedInbound.lines" :key="`${line.orderLineId}-${line.batchNo}`"><span>{{ line.productCode }} · {{ line.productName }} · {{ line.batchNo }}</span><strong>{{ line.quantity }} · {{ line.productionDate.slice(0, 10) }} - {{ line.expiresAt.slice(0, 10) }}</strong></li></ul></section>
        <section class="oa-detail-section"><h4>{{ t("inboundEvidence") }}</h4><p v-if="!selectedInbound.attachments.length" class="muted-empty">{{ t("noAttachments") }}</p><ul v-else class="purchase-detail-lines"><li v-for="file in selectedInbound.attachments" :key="file.fileId"><span>{{ file.name }}</span><strong>{{ file.mime }} · {{ file.size }} B</strong></li></ul></section>
      </div>
    </el-drawer>
  </section>
</template>
