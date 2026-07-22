<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
import zhCn from "element-plus/es/locale/lang/zh-cn";
import en from "element-plus/es/locale/lang/en";
import { ElMessage } from "element-plus/es/components/message/index.mjs";
import { ElMessageBox } from "element-plus/es/components/message-box/index.mjs";
import "element-plus/es/components/message/style/css";
import "element-plus/es/components/message-box/style/css";
import {
  Activity,
  Boxes,
  CheckCircle2,
  ClipboardCheck,
  Clock3,
  PackagePlus,
  Play,
  Plus,
  RefreshCw,
  Search,
  Settings2,
  ShieldCheck,
  TriangleAlert,
  Wrench
} from "lucide-vue-next";
import { api } from "./api";
import { language, t } from "./i18n";
import type { Asset, Dashboard, Inspection, Job, MaintenancePlan, SpareMovement, SparePart, Trend, WorkOrder } from "./types";

type Tab = "dashboard" | "assets" | "workOrders" | "inspections" | "spares" | "jobs";
type Dialog = "asset" | "workOrder" | "plan" | "inspection" | "inspectionComplete" | "spare" | "movement" | null;

const activeTab = ref<Tab>("dashboard");
const loading = ref(false);
const saving = ref(false);
const error = ref("");
const keyword = ref("");
const dialog = ref<Dialog>(null);
const selectedInspection = ref<Inspection | null>(null);
const dashboard = ref<Dashboard | null>(null);
const trends = ref<Trend[]>([]);
const assets = ref<Asset[]>([]);
const workOrders = ref<WorkOrder[]>([]);
const plans = ref<MaintenancePlan[]>([]);
const inspections = ref<Inspection[]>([]);
const spareParts = ref<SparePart[]>([]);
const movements = ref<SpareMovement[]>([]);
const jobs = ref<Job[]>([]);
const scope = reactive({ tenantId: "", organizationId: "" });
const form = reactive<Record<string, string | number>>({});

const elementLocale = computed(() => (language.value === "en-US" ? en : zhCn));
const dialogVisible = computed({
  get: () => dialog.value !== null,
  set: (visible: boolean) => {
    if (!visible) dialog.value = null;
  }
});
const maxTrend = computed(() => Math.max(1, ...trends.value.flatMap((item) => [item.closedWorkOrders, item.inspections])));
const metricCards = computed(() => [
  { key: "assets", label: t("activeAssets"), value: dashboard.value?.metrics.assets ?? 0, icon: Boxes, tone: "teal" },
  { key: "orders", label: t("openOrders"), value: dashboard.value?.metrics.openWorkOrders ?? 0, icon: Wrench, tone: "blue" },
  { key: "plans", label: t("overduePlans"), value: dashboard.value?.metrics.overduePlans ?? 0, icon: Clock3, tone: "amber" },
  { key: "stock", label: t("lowStock"), value: dashboard.value?.metrics.lowStockParts ?? 0, icon: TriangleAlert, tone: "red" },
  { key: "inspections", label: t("completedInspections"), value: dashboard.value?.metrics.completedInspections ?? 0, icon: ShieldCheck, tone: "green" }
]);

const dialogTitle = computed(() => {
  const key: Record<Exclude<Dialog, null>, string> = {
    asset: t("assets"),
    workOrder: t("workOrders"),
    plan: t("plan"),
    inspection: t("startInspection"),
    inspectionComplete: t("completeInspection"),
    spare: t("spares"),
    movement: t("stockMovement")
  };
  return dialog.value ? key[dialog.value] : "";
});

function resetForm(values: Record<string, string | number> = {}) {
  for (const key of Object.keys(form)) delete form[key];
  Object.assign(form, values);
}

function openDialog(type: Exclude<Dialog, null>, item?: Inspection) {
  dialog.value = type;
  selectedInspection.value = item ?? null;
  const defaults: Record<Exclude<Dialog, null>, Record<string, string | number>> = {
    asset: { code: "", name: "", category: "", model: "", serialNumber: "", location: "", nextMaintenanceAt: "" },
    workOrder: { assetId: "", number: "", title: "", priority: "normal", estimatedCost: 0 },
    plan: { assetId: "", name: "", intervalDays: 30, nextRunAt: "", status: "active" },
    inspection: { planId: "", workOrderId: "" },
    inspectionComplete: { result: "passed", finding: "", attachmentIds: "" },
    spare: { sku: "", name: "", unit: "pcs", minimumQuantity: 0, status: "active" },
    movement: { sparePartId: "", workOrderId: "", movementType: "inbound", quantity: 1, idempotencyKey: crypto.randomUUID() }
  };
  resetForm(defaults[type]);
}

async function loadCurrent() {
  loading.value = true;
  error.value = "";
  try {
    if (activeTab.value === "dashboard") {
      const [summary, trend] = await Promise.all([api.dashboard(), api.trends()]);
      dashboard.value = summary;
      trends.value = trend.items;
    } else if (activeTab.value === "assets") {
      assets.value = (await api.assets(keyword.value)).items;
    } else if (activeTab.value === "workOrders") {
      const [orders, assetPage] = await Promise.all([api.workOrders(keyword.value), api.assets()]);
      workOrders.value = orders.items;
      assets.value = assetPage.items;
    } else if (activeTab.value === "inspections") {
      const [planPage, inspectionPage, assetPage, orderPage] = await Promise.all([api.plans(), api.inspections(), api.assets(), api.workOrders()]);
      plans.value = planPage.items;
      inspections.value = inspectionPage.items;
      assets.value = assetPage.items;
      workOrders.value = orderPage.items;
    } else if (activeTab.value === "spares") {
      const [partPage, movementPage, orderPage] = await Promise.all([api.spareParts(keyword.value), api.movements(), api.workOrders()]);
      spareParts.value = partPage.items;
      movements.value = movementPage.items;
      workOrders.value = orderPage.items;
    } else {
      jobs.value = (await api.jobs()).items;
    }
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause);
  } finally {
    loading.value = false;
  }
}

function withScope(body: Record<string, unknown>) {
  return { ...body, tenantId: scope.tenantId, organizationId: scope.organizationId };
}

async function saveDialog() {
  if (!dialog.value) return;
  saving.value = true;
  try {
    if (dialog.value === "asset") {
      await api.createAsset(withScope({ ...form, nextMaintenanceAt: form.nextMaintenanceAt ? new Date(String(form.nextMaintenanceAt)).toISOString() : null }));
    } else if (dialog.value === "workOrder") {
      await api.createWorkOrder(withScope({ ...form, estimatedCost: Number(form.estimatedCost) }));
    } else if (dialog.value === "plan") {
      await api.createPlan(withScope({ ...form, intervalDays: Number(form.intervalDays), nextRunAt: new Date(String(form.nextRunAt)).toISOString() }));
    } else if (dialog.value === "inspection") {
      await api.createInspection(withScope({ ...form }));
    } else if (dialog.value === "inspectionComplete" && selectedInspection.value) {
      await api.completeInspection(selectedInspection.value.id, {
        result: form.result,
        finding: form.finding,
        attachmentIds: String(form.attachmentIds || "").split(",").map((item) => item.trim()).filter(Boolean)
      });
    } else if (dialog.value === "spare") {
      await api.createSparePart(withScope({ ...form, minimumQuantity: Number(form.minimumQuantity) }));
    } else if (dialog.value === "movement") {
      await api.createMovement(withScope({ ...form, quantity: Number(form.quantity) }));
    }
    dialog.value = null;
    ElMessage.success(t("success"));
    await loadCurrent();
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : String(cause));
  } finally {
    saving.value = false;
  }
}

async function retireAsset(item: Asset) {
  await ElMessageBox.confirm(t("confirmRetire"), t("retire"), { type: "warning" });
  try {
    await api.retireAsset(item.id);
    ElMessage.success(t("success"));
    await loadCurrent();
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : String(cause));
  }
}

function nextAction(item: WorkOrder): { action: string; label: string } | null {
  if (item.status === "draft") return { action: "dispatch", label: t("dispatch") };
  if (item.status === "dispatched") return { action: "start", label: t("start") };
  if (item.status === "in_progress") return { action: "submit", label: t("submit") };
  if (item.status === "approved") return { action: "complete", label: t("complete") };
  if (item.status === "completed") return { action: "close", label: t("close") };
  return null;
}

async function advanceWorkOrder(item: WorkOrder) {
  const next = nextAction(item);
  if (!next) return;
  let body: Record<string, unknown> = {};
  if (next.action === "dispatch") {
    const result = await ElMessageBox.prompt(t("assignee"), next.label, { inputPattern: /\S+/, inputErrorMessage: t("assignee") });
    body = { assigneeId: result.value };
  } else {
    await ElMessageBox.confirm(t("confirmAction"), next.label, { type: "warning" });
  }
  try {
    await api.transitionWorkOrder(item.id, next.action, body);
    ElMessage.success(t("success"));
    await loadCurrent();
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : String(cause));
  }
}

async function schedule(kind: "maintenance-due-scan" | "spare-stock-scan") {
  try {
    await api.scheduleJob(kind, withScope({}));
    ElMessage.success(t("success"));
    await loadCurrent();
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : String(cause));
  }
}

function formatDate(value?: string) {
  if (!value) return "-";
  return new Intl.DateTimeFormat(language.value, { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
}

function assetName(id: string) {
  return assets.value.find((item) => item.id === id)?.name ?? id;
}

function partName(id: string) {
  return spareParts.value.find((item) => item.id === id)?.name ?? id;
}

function statusType(status: string): "success" | "warning" | "danger" | "info" | "primary" {
  if (["active", "approved", "completed", "closed", "succeeded", "passed"].includes(status)) return "success";
  if (["draft", "pending", "scheduled", "dispatched"].includes(status)) return "info";
  if (["failed", "retired", "rejected", "cancelled"].includes(status)) return "danger";
  if (["in_progress", "awaiting_approval", "running"].includes(status)) return "warning";
  return "primary";
}

watch(activeTab, () => {
  keyword.value = "";
  void loadCurrent();
});
let searchTimer: number | undefined;
watch(keyword, () => {
  if (!["assets", "workOrders", "spares"].includes(activeTab.value)) return;
  window.clearTimeout(searchTimer);
  searchTimer = window.setTimeout(() => void loadCurrent(), 300);
});
onMounted(() => void loadCurrent());
</script>

<template>
  <el-config-provider :locale="elementLocale">
    <main class="maintenance-app">
      <header class="app-header">
        <div class="brand-block">
          <span class="brand-mark"><Wrench :size="22" /></span>
          <div>
            <h1>{{ t("title") }}</h1>
            <p>{{ t("subtitle") }}</p>
          </div>
        </div>
        <div class="header-actions">
          <el-popover placement="bottom-end" :width="320" trigger="click">
            <template #reference>
              <el-tooltip :content="t('scope')">
                <el-button circle aria-label="Business scope"><Settings2 :size="18" /></el-button>
              </el-tooltip>
            </template>
            <div class="scope-fields">
              <el-input v-model="scope.tenantId" :placeholder="t('tenantId')" clearable />
              <el-input v-model="scope.organizationId" :placeholder="t('organizationId')" clearable />
            </div>
          </el-popover>
          <el-tooltip :content="t('refresh')">
            <el-button circle :loading="loading" aria-label="Refresh" @click="loadCurrent"><RefreshCw :size="18" /></el-button>
          </el-tooltip>
        </div>
      </header>

      <el-tabs v-model="activeTab" class="domain-tabs">
        <el-tab-pane name="dashboard"><template #label><Activity :size="17" />{{ t("dashboard") }}</template></el-tab-pane>
        <el-tab-pane name="assets"><template #label><Boxes :size="17" />{{ t("assets") }}</template></el-tab-pane>
        <el-tab-pane name="workOrders"><template #label><Wrench :size="17" />{{ t("workOrders") }}</template></el-tab-pane>
        <el-tab-pane name="inspections"><template #label><ClipboardCheck :size="17" />{{ t("inspections") }}</template></el-tab-pane>
        <el-tab-pane name="spares"><template #label><PackagePlus :size="17" />{{ t("spares") }}</template></el-tab-pane>
        <el-tab-pane name="jobs"><template #label><Play :size="17" />{{ t("jobs") }}</template></el-tab-pane>
      </el-tabs>

      <el-alert v-if="error" :title="t('error')" :description="error" type="error" show-icon :closable="false" />

      <section v-if="activeTab === 'dashboard'" v-loading="loading" class="dashboard-view">
        <div class="metric-grid">
          <article v-for="metric in metricCards" :key="metric.key" class="metric-card" :data-tone="metric.tone">
            <span class="metric-icon"><component :is="metric.icon" :size="20" /></span>
            <div><strong>{{ metric.value }}</strong><span>{{ metric.label }}</span></div>
          </article>
        </div>
        <section class="trend-panel">
          <header><div><h2>{{ t("recentTrend") }}</h2><p>{{ formatDate(dashboard?.generatedAt) }}</p></div></header>
          <div v-if="trends.length" class="trend-chart" role="img" :aria-label="t('recentTrend')">
            <div v-for="item in trends" :key="item.month" class="trend-column">
              <div class="trend-bars">
                <el-tooltip :content="`${t('closedOrders')}: ${item.closedWorkOrders}`"><span class="trend-bar orders" :style="{ height: `${Math.max(4, item.closedWorkOrders / maxTrend * 100)}%` }" /></el-tooltip>
                <el-tooltip :content="`${t('inspections')}: ${item.inspections}`"><span class="trend-bar checks" :style="{ height: `${Math.max(4, item.inspections / maxTrend * 100)}%` }" /></el-tooltip>
              </div>
              <span>{{ item.month.slice(5) }}</span>
            </div>
          </div>
          <el-empty v-else :description="t('empty')" />
          <footer class="trend-legend"><span class="orders">{{ t("closedOrders") }}</span><span class="checks">{{ t("inspections") }}</span></footer>
        </section>
      </section>

      <section v-else class="data-view">
        <div class="table-toolbar">
          <el-input v-if="['assets', 'workOrders', 'spares'].includes(activeTab)" v-model="keyword" class="search-input" :placeholder="t('search')" clearable><template #prefix><Search :size="17" /></template></el-input>
          <span v-else />
          <div class="toolbar-actions">
            <template v-if="activeTab === 'assets'"><el-button type="primary" @click="openDialog('asset')"><Plus :size="17" />{{ t("create") }}</el-button></template>
            <template v-else-if="activeTab === 'workOrders'"><el-button type="primary" @click="openDialog('workOrder')"><Plus :size="17" />{{ t("create") }}</el-button></template>
            <template v-else-if="activeTab === 'inspections'">
              <el-button @click="openDialog('plan')"><Plus :size="17" />{{ t("plan") }}</el-button>
              <el-button type="primary" @click="openDialog('inspection')"><ClipboardCheck :size="17" />{{ t("startInspection") }}</el-button>
            </template>
            <template v-else-if="activeTab === 'spares'">
              <el-button @click="openDialog('movement')"><PackagePlus :size="17" />{{ t("stockMovement") }}</el-button>
              <el-button type="primary" @click="openDialog('spare')"><Plus :size="17" />{{ t("create") }}</el-button>
            </template>
            <template v-else>
              <el-button @click="schedule('maintenance-due-scan')"><Clock3 :size="17" />{{ t("maintenanceScan") }}</el-button>
              <el-button type="primary" @click="schedule('spare-stock-scan')"><TriangleAlert :size="17" />{{ t("stockScan") }}</el-button>
            </template>
          </div>
        </div>

        <el-table v-if="activeTab === 'assets'" v-loading="loading" :data="assets" row-key="id" height="calc(100vh - 224px)" stripe>
          <el-table-column prop="code" :label="t('code')" min-width="130" />
          <el-table-column prop="name" :label="t('name')" min-width="160" />
          <el-table-column prop="category" :label="t('category')" min-width="110" />
          <el-table-column prop="model" :label="t('model')" min-width="120" />
          <el-table-column prop="location" :label="t('location')" min-width="140" />
          <el-table-column :label="t('nextMaintenance')" min-width="170"><template #default="{ row }">{{ formatDate(row.nextMaintenanceAt) }}</template></el-table-column>
          <el-table-column :label="t('status')" width="110"><template #default="{ row }"><el-tag :type="statusType(row.status)" effect="plain">{{ row.status }}</el-tag></template></el-table-column>
          <el-table-column :label="t('actions')" width="88" fixed="right"><template #default="{ row }"><el-tooltip :content="t('retire')"><el-button text type="danger" :disabled="row.status === 'retired'" circle @click="retireAsset(row)"><TriangleAlert :size="17" /></el-button></el-tooltip></template></el-table-column>
          <template #empty><el-empty :description="t('empty')" /></template>
        </el-table>

        <el-table v-else-if="activeTab === 'workOrders'" v-loading="loading" :data="workOrders" row-key="id" height="calc(100vh - 224px)" stripe>
          <el-table-column prop="number" :label="t('number')" min-width="140" />
          <el-table-column prop="title" :label="t('titleField')" min-width="200" />
          <el-table-column :label="t('asset')" min-width="150"><template #default="{ row }">{{ assetName(row.assetId) }}</template></el-table-column>
          <el-table-column prop="priority" :label="t('priority')" width="100" />
          <el-table-column prop="assigneeId" :label="t('assignee')" min-width="120" />
          <el-table-column :label="t('estimatedCost')" width="130"><template #default="{ row }">{{ Number(row.estimatedCost).toLocaleString() }}</template></el-table-column>
          <el-table-column :label="t('status')" min-width="150"><template #default="{ row }"><el-tag :type="statusType(row.status)" effect="plain">{{ row.status }}</el-tag></template></el-table-column>
          <el-table-column :label="t('actions')" width="110" fixed="right"><template #default="{ row }"><el-button v-if="nextAction(row)" size="small" type="primary" @click="advanceWorkOrder(row)">{{ nextAction(row)?.label }}</el-button></template></el-table-column>
          <template #empty><el-empty :description="t('empty')" /></template>
        </el-table>

        <div v-else-if="activeTab === 'inspections'" v-loading="loading" class="split-tables">
          <section><h2>{{ t("plan") }}</h2><el-table :data="plans" row-key="id" height="calc(50vh - 145px)" stripe><el-table-column prop="name" :label="t('name')" min-width="160" /><el-table-column :label="t('asset')" min-width="140"><template #default="{ row }">{{ assetName(row.assetId) }}</template></el-table-column><el-table-column prop="intervalDays" :label="t('intervalDays')" width="120" /><el-table-column :label="t('nextRun')" min-width="170"><template #default="{ row }">{{ formatDate(row.nextRunAt) }}</template></el-table-column><el-table-column :label="t('status')" width="110"><template #default="{ row }"><el-tag :type="statusType(row.status)" effect="plain">{{ row.status }}</el-tag></template></el-table-column><template #empty><el-empty :description="t('empty')" /></template></el-table></section>
          <section><h2>{{ t("inspections") }}</h2><el-table :data="inspections" row-key="id" height="calc(50vh - 145px)" stripe><el-table-column :label="t('asset')" min-width="150"><template #default="{ row }">{{ assetName(row.assetId) }}</template></el-table-column><el-table-column prop="result" :label="t('result')" min-width="110" /><el-table-column prop="finding" :label="t('finding')" min-width="220" show-overflow-tooltip /><el-table-column :label="t('status')" width="120"><template #default="{ row }"><el-tag :type="statusType(row.status)" effect="plain">{{ row.status }}</el-tag></template></el-table-column><el-table-column :label="t('actions')" width="90" fixed="right"><template #default="{ row }"><el-tooltip :content="t('completeInspection')"><el-button text type="primary" circle :disabled="row.status === 'completed'" @click="openDialog('inspectionComplete', row)"><CheckCircle2 :size="18" /></el-button></el-tooltip></template></el-table-column><template #empty><el-empty :description="t('empty')" /></template></el-table></section>
        </div>

        <div v-else-if="activeTab === 'spares'" v-loading="loading" class="split-tables">
          <section><h2>{{ t("spares") }}</h2><el-table :data="spareParts" row-key="id" height="calc(50vh - 145px)" stripe><el-table-column prop="sku" :label="t('sku')" min-width="130" /><el-table-column prop="name" :label="t('name')" min-width="160" /><el-table-column :label="t('quantity')" width="120"><template #default="{ row }"><strong :class="{ 'low-stock': row.quantity <= row.minimumQuantity }">{{ row.quantity }} {{ row.unit }}</strong></template></el-table-column><el-table-column prop="minimumQuantity" :label="t('minimumQuantity')" width="130" /><el-table-column :label="t('status')" width="110"><template #default="{ row }"><el-tag :type="statusType(row.status)" effect="plain">{{ row.status }}</el-tag></template></el-table-column><template #empty><el-empty :description="t('empty')" /></template></el-table></section>
          <section><h2>{{ t("stockMovement") }}</h2><el-table :data="movements" row-key="id" height="calc(50vh - 145px)" stripe><el-table-column :label="t('name')" min-width="160"><template #default="{ row }">{{ partName(row.sparePartId) }}</template></el-table-column><el-table-column prop="movementType" :label="t('movementType')" min-width="120" /><el-table-column prop="quantity" :label="t('quantity')" width="100" /><el-table-column prop="idempotencyKey" :label="t('idempotencyKey')" min-width="200" show-overflow-tooltip /><el-table-column :label="t('runAt')" min-width="170"><template #default="{ row }">{{ formatDate(row.occurredAt) }}</template></el-table-column><template #empty><el-empty :description="t('empty')" /></template></el-table></section>
        </div>

        <el-table v-else v-loading="loading" :data="jobs" row-key="id" height="calc(100vh - 224px)" stripe>
          <el-table-column prop="id" label="ID" min-width="220" show-overflow-tooltip />
          <el-table-column prop="kind" :label="t('kind')" min-width="180" />
          <el-table-column :label="t('status')" width="130"><template #default="{ row }"><el-tag :type="statusType(row.status)" effect="plain">{{ row.status }}</el-tag></template></el-table-column>
          <el-table-column prop="attempts" :label="t('attempts')" width="100" />
          <el-table-column :label="t('runAt')" min-width="180"><template #default="{ row }">{{ formatDate(row.runAt || row.updatedAt) }}</template></el-table-column>
          <template #empty><el-empty :description="t('empty')" /></template>
        </el-table>
      </section>

      <el-dialog v-model="dialogVisible" :title="dialogTitle" width="min(620px, calc(100vw - 24px))" destroy-on-close>
        <el-form label-position="top" @submit.prevent="saveDialog">
          <template v-if="dialog === 'asset'">
            <div class="form-grid"><el-form-item :label="t('code')" required><el-input v-model="form.code" /></el-form-item><el-form-item :label="t('name')" required><el-input v-model="form.name" /></el-form-item><el-form-item :label="t('category')" required><el-input v-model="form.category" /></el-form-item><el-form-item :label="t('model')"><el-input v-model="form.model" /></el-form-item><el-form-item :label="t('serialNumber')"><el-input v-model="form.serialNumber" /></el-form-item><el-form-item :label="t('location')"><el-input v-model="form.location" /></el-form-item></div><el-form-item :label="t('nextMaintenance')"><el-date-picker v-model="form.nextMaintenanceAt" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" /></el-form-item>
          </template>
          <template v-else-if="dialog === 'workOrder'">
            <el-form-item :label="t('asset')" required><el-select v-model="form.assetId" filterable><el-option v-for="item in assets.filter((value) => value.status !== 'retired')" :key="item.id" :label="`${item.code} · ${item.name}`" :value="item.id" /></el-select></el-form-item><div class="form-grid"><el-form-item :label="t('number')" required><el-input v-model="form.number" /></el-form-item><el-form-item :label="t('priority')"><el-select v-model="form.priority"><el-option label="Low" value="low" /><el-option label="Normal" value="normal" /><el-option label="High" value="high" /><el-option label="Urgent" value="urgent" /></el-select></el-form-item></div><el-form-item :label="t('titleField')" required><el-input v-model="form.title" /></el-form-item><el-form-item :label="t('estimatedCost')"><el-input-number v-model="form.estimatedCost" :min="0" :precision="2" /></el-form-item>
          </template>
          <template v-else-if="dialog === 'plan'">
            <el-form-item :label="t('asset')" required><el-select v-model="form.assetId" filterable><el-option v-for="item in assets.filter((value) => value.status !== 'retired')" :key="item.id" :label="`${item.code} · ${item.name}`" :value="item.id" /></el-select></el-form-item><el-form-item :label="t('name')" required><el-input v-model="form.name" /></el-form-item><div class="form-grid"><el-form-item :label="t('intervalDays')" required><el-input-number v-model="form.intervalDays" :min="1" :max="3650" /></el-form-item><el-form-item :label="t('nextRun')" required><el-date-picker v-model="form.nextRunAt" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" /></el-form-item></div>
          </template>
          <template v-else-if="dialog === 'inspection'">
            <el-form-item :label="t('plan')" required><el-select v-model="form.planId" filterable><el-option v-for="item in plans.filter((value) => value.status === 'active')" :key="item.id" :label="`${item.name} · ${assetName(item.assetId)}`" :value="item.id" /></el-select></el-form-item><el-form-item :label="t('workOrder')"><el-select v-model="form.workOrderId" clearable filterable><el-option v-for="item in workOrders.filter((value) => value.status !== 'closed')" :key="item.id" :label="`${item.number} · ${item.title}`" :value="item.id" /></el-select></el-form-item>
          </template>
          <template v-else-if="dialog === 'inspectionComplete'">
            <el-form-item :label="t('result')" required><el-select v-model="form.result"><el-option label="Passed" value="passed" /><el-option label="Attention" value="attention" /><el-option label="Failed" value="failed" /></el-select></el-form-item><el-form-item :label="t('finding')"><el-input v-model="form.finding" type="textarea" :rows="4" /></el-form-item><el-form-item :label="t('attachments')"><el-input v-model="form.attachmentIds" /></el-form-item>
          </template>
          <template v-else-if="dialog === 'spare'">
            <div class="form-grid"><el-form-item :label="t('sku')" required><el-input v-model="form.sku" /></el-form-item><el-form-item :label="t('name')" required><el-input v-model="form.name" /></el-form-item><el-form-item :label="t('unit')" required><el-input v-model="form.unit" /></el-form-item><el-form-item :label="t('minimumQuantity')"><el-input-number v-model="form.minimumQuantity" :min="0" :precision="2" /></el-form-item></div>
          </template>
          <template v-else-if="dialog === 'movement'">
            <el-form-item :label="t('name')" required><el-select v-model="form.sparePartId" filterable><el-option v-for="item in spareParts.filter((value) => value.status === 'active')" :key="item.id" :label="`${item.sku} · ${item.name}`" :value="item.id" /></el-select></el-form-item><div class="form-grid"><el-form-item :label="t('movementType')" required><el-select v-model="form.movementType"><el-option label="Inbound" value="inbound" /><el-option label="Outbound" value="outbound" /><el-option label="Usage" value="usage" /><el-option label="Adjustment" value="adjustment" /></el-select></el-form-item><el-form-item :label="t('quantity')" required><el-input-number v-model="form.quantity" :min="0.01" :precision="2" /></el-form-item></div><el-form-item :label="t('workOrder')"><el-select v-model="form.workOrderId" clearable filterable><el-option v-for="item in workOrders" :key="item.id" :label="`${item.number} · ${item.title}`" :value="item.id" /></el-select></el-form-item><el-form-item :label="t('idempotencyKey')" required><el-input v-model="form.idempotencyKey" /></el-form-item>
          </template>
        </el-form>
        <template #footer><el-button @click="dialog = null">{{ t("cancel") }}</el-button><el-button type="primary" :loading="saving" @click="saveDialog">{{ t("save") }}</el-button></template>
      </el-dialog>
    </main>
  </el-config-provider>
</template>
