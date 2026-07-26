<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
import zhCn from "element-plus/es/locale/lang/zh-cn";
import en from "element-plus/es/locale/lang/en";
import { ElMessage } from "element-plus/es/components/message/index.mjs";
import { ElMessageBox } from "element-plus/es/components/message-box/index.mjs";
import {
  BadgeCheck,
  Boxes,
  Building2,
  CircleAlert,
  ClipboardCheck,
  Database,
  Edit3,
  Eye,
  FileCheck2,
  FileText,
  FileUp,
  Inbox,
  Languages,
  PackageSearch,
  Plus,
  RefreshCw,
  ScanSearch,
  Search,
  ShieldCheck,
  Tags,
  Truck,
  UserRound,
  UsersRound
} from "lucide-vue-next";
import { api } from "./api";
import OAWorkspace from "./components/OAWorkspace.vue";
import { language, setLocale, t } from "./i18n";
import type { MessageKey } from "./i18n";
import type { Catalog, Employee, ModuleKey, OAWorkspaceMode, Party, Product, Qualification, QualificationType, WorkspaceRecord } from "./types";

type FieldType = "text" | "textarea" | "number" | "date" | "select" | "checkbox";
type Option = { value: string; zh: string; en: string };
type Field = { key: string; zh: string; en: string; type?: FieldType; required?: boolean; min?: number; max?: number; span?: 1 | 2; options?: Option[]; lookup?: "categories" | "units" | "manufacturers" | "qualificationTypes" };
type DialogKind = "editor" | "status" | "review" | "attachment" | null;

const nav = [
  { key: "employee", icon: UserRound },
  { key: "customer", icon: UsersRound },
  { key: "supplier", icon: Truck },
  { key: "product", icon: Boxes },
  { key: "category", icon: Tags },
  { key: "unit", icon: PackageSearch },
  { key: "manufacturer", icon: Building2 },
  { key: "qualification", icon: FileCheck2 },
  { key: "qualificationType", icon: ShieldCheck }
] as const;

const subjectOptions: Option[] = [
  { value: "customer", zh: "客户", en: "Customer" },
  { value: "supplier", zh: "供应商", en: "Supplier" },
  { value: "product", zh: "药品", en: "Product" },
  { value: "manufacturer", zh: "生产企业", en: "Manufacturer" }
];
const gateOptions: Option[] = [
  { value: "sales", zh: "销售准入", en: "Sales" },
  { value: "purchase", zh: "采购准入", en: "Purchase" },
  { value: "sale", zh: "药品销售", en: "Product sale" },
  { value: "supply", zh: "生产供应", en: "Supply" }
];

const fieldMap: Record<ModuleKey, Field[]> = {
  employee: [
    { key: "code", zh: "员工工号", en: "Employee code", required: true }, { key: "name", zh: "姓名", en: "Name", required: true },
    { key: "departmentId", zh: "部门", en: "Department", required: true }, { key: "positionId", zh: "职位", en: "Position", required: true },
    { key: "phone", zh: "手机号", en: "Phone" }, { key: "email", zh: "邮箱", en: "Email" },
    { key: "hireDate", zh: "入职日期", en: "Hire date", type: "date", required: true },
    { key: "certificateName", zh: "执业资质", en: "Qualification" }, { key: "certificateNumber", zh: "证书编号", en: "Certificate number" },
    { key: "certificateExpiresAt", zh: "资质有效期", en: "Qualification expiry", type: "date" }
  ],
  customer: partyFields(), supplier: partyFields(),
  product: [
    { key: "code", zh: "药品编码", en: "Product code", required: true }, { key: "sku", zh: "SKU", en: "SKU", required: true },
    { key: "name", zh: "商品名称", en: "Product name", required: true }, { key: "genericName", zh: "通用名称", en: "Generic name", required: true },
    { key: "categoryId", zh: "药品分类", en: "Category", type: "select", lookup: "categories", required: true },
    { key: "unitId", zh: "计量单位", en: "Unit", type: "select", lookup: "units", required: true },
    { key: "manufacturerId", zh: "生产企业", en: "Manufacturer", type: "select", lookup: "manufacturers", required: true },
    { key: "dosageForm", zh: "剂型", en: "Dosage form", required: true }, { key: "specification", zh: "规格", en: "Specification", required: true },
    { key: "approvalNumber", zh: "批准文号", en: "Approval number", required: true }, { key: "barcode", zh: "商品条码", en: "Barcode" },
    { key: "storageCondition", zh: "储存要求", en: "Storage condition", required: true, span: 2 },
    { key: "temperatureMin", zh: "最低温度（℃）", en: "Minimum temperature (C)", type: "number", min: -80, max: 80, required: true },
    { key: "temperatureMax", zh: "最高温度（℃）", en: "Maximum temperature (C)", type: "number", min: -80, max: 80, required: true }
  ],
  category: catalogFields("category"), unit: catalogFields("unit"), manufacturer: catalogFields("manufacturer"),
  qualificationType: [
    { key: "code", zh: "类型编码", en: "Type code", required: true }, { key: "name", zh: "类型名称", en: "Type name", required: true },
    { key: "subjectType", zh: "主体类型", en: "Subject type", type: "select", options: subjectOptions, required: true },
    { key: "businessGate", zh: "业务准入", en: "Business gate", type: "select", options: gateOptions, required: true },
    { key: "validityDays", zh: "有效天数", en: "Validity days", type: "number", min: 1, max: 3650, required: true },
    { key: "alertDays", zh: "预警天数", en: "Alert days", type: "number", min: 1, max: 3650, required: true },
    { key: "evidenceRequired", zh: "必须上传证据", en: "Evidence required", type: "checkbox" },
    { key: "businessRequired", zh: "业务准入必备", en: "Required for business", type: "checkbox" },
    { key: "description", zh: "规则说明", en: "Description", type: "textarea", span: 2 }
  ],
  qualification: [
    { key: "typeId", zh: "资质类型", en: "Qualification type", type: "select", lookup: "qualificationTypes", required: true },
    { key: "subjectType", zh: "主体类型", en: "Subject type", type: "select", options: subjectOptions, required: true },
    { key: "subjectId", zh: "业务主体 ID", en: "Business subject ID", required: true },
    { key: "certificateNumber", zh: "证照编号", en: "Certificate number", required: true },
    { key: "issuer", zh: "发证机关", en: "Issuer", required: true },
    { key: "validFrom", zh: "生效日期", en: "Valid from", type: "date", required: true },
    { key: "validTo", zh: "有效期至", en: "Valid to", type: "date", required: true },
    { key: "evidenceName", zh: "证据文件", en: "Evidence file", span: 2 }
  ]
};

function partyFields(): Field[] {
  return [
    { key: "code", zh: "单位编码", en: "Party code", required: true }, { key: "name", zh: "单位名称", en: "Party name", required: true },
    { key: "unifiedSocialCreditCode", zh: "统一社会信用代码", en: "Unified social credit code", required: true },
    { key: "region", zh: "所在地区", en: "Region", required: true }, { key: "rating", zh: "合作评级", en: "Rating", type: "number", min: 1, max: 5, required: true },
    { key: "contactName", zh: "主联系人", en: "Primary contact", required: true }, { key: "contactTitle", zh: "联系人职务", en: "Contact title" },
    { key: "contactPhone", zh: "联系电话", en: "Contact phone" }, { key: "contactEmail", zh: "联系邮箱", en: "Contact email" },
    { key: "province", zh: "省份", en: "Province", required: true }, { key: "city", zh: "城市", en: "City", required: true },
    { key: "district", zh: "区县", en: "District" }, { key: "addressDetail", zh: "详细地址", en: "Address", required: true },
    { key: "currency", zh: "结算币种", en: "Currency", required: true },
    { key: "paymentDays", zh: "账期（天）", en: "Payment days", type: "number", min: 0, max: 365, required: true },
    { key: "creditLimit", zh: "信用额度", en: "Credit limit", type: "number", min: 0, max: 100000000, required: true }
  ];
}

function catalogFields(kind: "category" | "unit" | "manufacturer"): Field[] {
  const common: Field[] = [{ key: "code", zh: "目录编码", en: "Catalog code", required: true }, { key: "name", zh: "目录名称", en: "Catalog name", required: true }];
  if (kind === "category") common.push({ key: "parentId", zh: "上级分类 ID", en: "Parent category ID" });
  if (kind === "unit") common.push({ key: "symbol", zh: "单位符号", en: "Symbol", required: true }, { key: "decimalPlaces", zh: "小数精度", en: "Decimal places", type: "number", min: 0, max: 6, required: true });
  if (kind === "manufacturer") common.push({ key: "unifiedSocialCreditCode", zh: "统一社会信用代码", en: "Unified social credit code", required: true }, { key: "licenseNumber", zh: "生产许可证号", en: "Manufacturing license", required: true });
  common.push({ key: "description", zh: "说明", en: "Description", type: "textarea", span: 2 });
  return common;
}

const activeModule = ref<ModuleKey>("employee");
const workspaceArea = ref<OAWorkspaceMode | "master">("requests");
const loading = ref(false);
const saving = ref(false);
const error = ref("");
const denied = ref(false);
const keyword = ref("");
const status = ref("");
const items = ref<WorkspaceRecord[]>([]);
const reminders = ref<WorkspaceRecord[]>([]);
const selected = ref<WorkspaceRecord | null>(null);
const detailVisible = ref(false);
const dialog = ref<DialogKind>(null);
const transitionAction = ref("");
const form = reactive<Record<string, string | number | boolean>>({});
const scope = reactive({ tenantId: "", organizationId: "" });
const lookups = reactive<Record<"categories" | "units" | "manufacturers" | "qualificationTypes", Array<Catalog | QualificationType>>>({ categories: [], units: [], manufacturers: [], qualificationTypes: [] });
const evidenceContent = ref("");
const attachmentContent = ref("");

const elementLocale = computed(() => language.value === "en-US" ? en : zhCn);
const isEnglish = computed(() => language.value === "en-US");
const moduleLabel = computed(() => t(activeModule.value));
const fields = computed(() => fieldMap[activeModule.value]);
const dialogVisible = computed({ get: () => dialog.value !== null, set: (value: boolean) => { if (!value) dialog.value = null; } });
const activeCount = computed(() => items.value.filter((item) => recordStatus(item) === "active" || recordStatus(item) === "approved").length);
const attentionCount = computed(() => items.value.filter((item) => ["disabled", "pending", "rejected", "revoked", "expired", "left"].includes(recordStatus(item))).length + reminders.value.length);
const complianceRate = computed(() => items.value.length ? Math.round(activeCount.value / items.value.length * 100) : 100);
const statusOptions = computed(() => activeModule.value === "employee" ? ["active", "on_leave", "left"] : activeModule.value === "qualification" ? ["draft", "pending", "approved", "rejected", "revoked", "expired"] : ["active", "disabled"]);
const dialogTitle = computed(() => {
  if (dialog.value === "editor") {
    const action = selected.value ? t("edit") : t("create");
    return isEnglish.value ? `${action} ${moduleLabel.value}` : `${action}${moduleLabel.value}`;
  }
  if (dialog.value === "attachment") return t("upload");
  if (dialog.value === "review") return t(transitionAction.value as "submit" | "approve" | "reject" | "revoke");
  return activeModule.value === "employee" ? t("leave") : t("statusReason");
});

function label(field: Field): string { return isEnglish.value ? field.en : field.zh; }
function optionLabel(option: Option): string { return isEnglish.value ? option.en : option.zh; }
function recordStatus(item: WorkspaceRecord): string { return "employmentStatus" in item ? item.employmentStatus : item.status; }
function recordCode(item: WorkspaceRecord): string { return "certificateNumber" in item ? item.certificateNumber : item.code; }
function recordName(item: WorkspaceRecord): string {
  if ("certificateNumber" in item) return lookupName("qualificationTypes", item.typeId);
  return item.name;
}
function statusType(value: string): "success" | "warning" | "danger" | "info" | "primary" {
  if (["active", "approved"].includes(value)) return "success";
  if (["pending", "on_leave", "draft"].includes(value)) return "warning";
  if (["rejected", "revoked", "expired"].includes(value)) return "danger";
  return "info";
}
function statusLabel(value: string): string { return t(value as MessageKey); }
function lookupName(key: keyof typeof lookups, id: string): string { return lookups[key].find((item) => item.id === id)?.name ?? id ?? "-"; }
function fieldOptions(field: Field): Array<Option | Catalog | QualificationType> { return field.lookup ? lookups[field.lookup] : field.options ?? []; }
function fieldOptionValue(option: Option | Catalog | QualificationType): string { return "value" in option ? option.value : option.id; }
function fieldOptionLabel(option: Option | Catalog | QualificationType): string { return "value" in option ? optionLabel(option) : `${option.code} · ${option.name}`; }

function applyTheme(): void {
  const theme = window.__SKOLL_HOST__?.theme;
  document.documentElement.dataset.theme = theme?.colorScheme ?? "light";
  document.documentElement.dataset.density = theme?.density ?? "comfortable";
  Object.entries(theme?.tokens ?? {}).forEach(([key, value]) => document.documentElement.style.setProperty(key.startsWith("--") ? key : `--${key}`, value));
}

async function loadCurrent(): Promise<void> {
  if (loading.value) return;
  loading.value = true; error.value = ""; denied.value = false;
  try {
    const tasks: Promise<unknown>[] = [api.list(activeModule.value, keyword.value.trim(), status.value)];
    if (activeModule.value === "employee") tasks.push(api.employeeReminders());
    if (activeModule.value === "product") tasks.push(api.categories(), api.units(), api.manufacturers());
    if (activeModule.value === "qualification") tasks.push(api.qualificationTypes());
    const result = await Promise.all(tasks);
    const page = result[0] as { items?: WorkspaceRecord[] };
    items.value = page.items ?? [];
    reminders.value = activeModule.value === "employee" ? ((result[1] as { items?: WorkspaceRecord[] }).items ?? []) : [];
    if (activeModule.value === "product") {
      lookups.categories = (result[1] as { items: Catalog[] }).items ?? [];
      lookups.units = (result[2] as { items: Catalog[] }).items ?? [];
      lookups.manufacturers = (result[3] as { items: Catalog[] }).items ?? [];
    }
    if (activeModule.value === "qualification") lookups.qualificationTypes = (result[1] as { items: QualificationType[] }).items ?? [];
  } catch (reason) {
    error.value = errorMessage(reason);
    denied.value = /forbidden|permission|denied|权限|拒绝/i.test(error.value);
  } finally { loading.value = false; }
}

function errorMessage(reason: unknown): string {
  if (reason instanceof Error && reason.message === "SKOLL_HOST_UNAVAILABLE") return isEnglish.value ? "SKOLL host context is unavailable" : "未连接 SKOLL 插件宿主";
  return reason instanceof Error ? reason.message : t("failed");
}

function clearForm(): void { Object.keys(form).forEach((key) => delete form[key]); evidenceContent.value = ""; attachmentContent.value = ""; }
function openCreate(): void {
  selected.value = null; clearForm();
  Object.assign(form, defaults(activeModule.value));
  dialog.value = "editor";
}
function defaults(module: ModuleKey): Record<string, string | number | boolean> {
  const today = new Date().toISOString().slice(0, 10);
  if (module === "employee") return { hireDate: today };
  if (module === "customer" || module === "supplier") return { rating: 3, currency: "CNY", paymentDays: 30, creditLimit: 0 };
  if (module === "product") return { temperatureMin: 2, temperatureMax: 25 };
  if (module === "unit") return { decimalPlaces: 0 };
  if (module === "qualificationType") return { validityDays: 365, alertDays: 30, evidenceRequired: true, businessRequired: true, subjectType: "customer", businessGate: "sales" };
  if (module === "qualification") return { validFrom: today };
  return {};
}
function openEdit(item: WorkspaceRecord): void {
  selected.value = item; clearForm(); Object.assign(form, item);
  if ("certificates" in item && item.certificates[0]) Object.assign(form, { certificateName: item.certificates[0].name, certificateNumber: item.certificates[0].number, certificateExpiresAt: item.certificates[0].expiresAt.slice(0, 10) });
  if ("contacts" in item) {
    const contact = item.contacts.find((entry) => entry.primary) ?? item.contacts[0]; const address = item.addresses.find((entry) => entry.default) ?? item.addresses[0];
    Object.assign(form, { contactName: contact?.name ?? "", contactTitle: contact?.title ?? "", contactPhone: contact?.phone ?? "", contactEmail: contact?.email ?? "", province: address?.province ?? "", city: address?.city ?? "", district: address?.district ?? "", addressDetail: address?.detail ?? "", ...item.settlementTerms });
  }
  dialog.value = "editor";
}

function validateForm(): boolean {
  const valid = fields.value.every((field) => !field.required || form[field.key] === 0 || form[field.key] === false || String(form[field.key] ?? "").trim() !== "");
  if (!valid) ElMessage.warning(t("formRequired"));
  return valid;
}
function scopePayload(): Record<string, unknown> { return { tenantId: scope.tenantId.trim(), organizationId: scope.organizationId.trim() }; }
function editorPayload(): Record<string, unknown> {
  const payload: Record<string, unknown> = { ...scopePayload(), version: selected.value?.version };
  fields.value.forEach((field) => { if (!["certificateName", "certificateNumber", "certificateExpiresAt", "contactName", "contactTitle", "contactPhone", "contactEmail", "province", "city", "district", "addressDetail", "currency", "paymentDays", "creditLimit", "evidenceName"].includes(field.key)) payload[field.key] = form[field.key] ?? (field.type === "checkbox" ? false : ""); });
  if (activeModule.value === "employee") payload.certificates = form.certificateName ? [{ id: "certificate-1", name: form.certificateName, number: form.certificateNumber ?? "", expiresAt: form.certificateExpiresAt ?? "" }] : [];
  if (activeModule.value === "customer" || activeModule.value === "supplier") {
    payload.contacts = [{ id: "contact-1", name: form.contactName, title: form.contactTitle ?? "", phone: form.contactPhone ?? "", email: form.contactEmail ?? "", primary: true }];
    payload.addresses = [{ id: "address-1", label: isEnglish.value ? "Primary" : "主要地址", province: form.province, city: form.city, district: form.district ?? "", detail: form.addressDetail, default: true }];
    payload.settlementTerms = { currency: form.currency, paymentDays: form.paymentDays, creditLimit: form.creditLimit };
  }
  if (activeModule.value === "qualification") payload.evidence = evidenceContent.value ? { name: form.evidenceName, contentBase64: evidenceContent.value } : {};
  return payload;
}
async function saveEditor(): Promise<void> {
  if (!validateForm() || saving.value) return;
  saving.value = true;
  try {
    const payload = editorPayload();
    if (selected.value) await api.update(activeModule.value, selected.value.id, payload);
    else await api.create(activeModule.value, payload);
    ElMessage.success(t("saved")); dialog.value = null; await loadCurrent();
  } catch (reason) { ElMessage.error(errorMessage(reason)); }
  finally { saving.value = false; }
}

async function readFile(file: File, target: "evidence" | "attachment"): Promise<void> {
  const maximum = target === "evidence" ? 5 * 1024 * 1024 : 1024 * 1024;
  if (file.size > maximum) { ElMessage.error(isEnglish.value ? `File exceeds ${maximum / 1024 / 1024} MB` : `文件不能超过 ${maximum / 1024 / 1024} MB`); return; }
  const content = await new Promise<string>((resolve, reject) => { const reader = new FileReader(); reader.onload = () => resolve(String(reader.result).split(",")[1] ?? ""); reader.onerror = () => reject(reader.error); reader.readAsDataURL(file); });
  form.evidenceName = file.name;
  if (target === "evidence") evidenceContent.value = content; else { attachmentContent.value = content; form.attachmentName = file.name; }
}

function openStatus(item: WorkspaceRecord, action?: string): void {
  selected.value = item; clearForm(); transitionAction.value = action ?? (recordStatus(item) === "disabled" ? "enable" : activeModule.value === "employee" ? "leave" : "disable");
  if (transitionAction.value === "enable") void submitTransition(); else dialog.value = activeModule.value === "qualification" ? "review" : "status";
}
async function submitTransition(): Promise<void> {
  if (!selected.value || saving.value) return;
  const dangerous = ["leave", "disable", "reject", "revoke"].includes(transitionAction.value);
  if (dangerous) {
    try { await ElMessageBox.confirm(t("dangerConfirm"), moduleLabel.value, { type: "warning", confirmButtonText: t("confirm"), cancelButtonText: t("cancel") }); }
    catch { return; }
  }
  saving.value = true;
  try {
    const body = { version: selected.value.version, reason: form.reason ?? "", comment: form.comment ?? "" };
    await api.transition(activeModule.value, selected.value.id, transitionAction.value, body);
    ElMessage.success(t("updated")); dialog.value = null; await loadCurrent();
  } catch (reason) { ElMessage.error(errorMessage(reason)); }
  finally { saving.value = false; }
}
function openAttachment(item: Employee): void { selected.value = item; clearForm(); dialog.value = "attachment"; }
async function uploadAttachment(): Promise<void> {
  if (!selected.value || !("attachments" in selected.value) || !attachmentContent.value) { ElMessage.warning(t("formRequired")); return; }
  saving.value = true;
  try { await api.attachEmployee(selected.value.id, { name: form.attachmentName, contentBase64: attachmentContent.value, version: selected.value.version }); ElMessage.success(t("saved")); dialog.value = null; await loadCurrent(); }
  catch (reason) { ElMessage.error(errorMessage(reason)); } finally { saving.value = false; }
}
async function scanExpiry(): Promise<void> {
  saving.value = true;
  try { await api.scanQualificationExpiry(30); ElMessage.success(t("updated")); await loadCurrent(); }
  catch (reason) { ElMessage.error(errorMessage(reason)); } finally { saving.value = false; }
}
function showDetails(item: WorkspaceRecord): void { selected.value = item; detailVisible.value = true; }
function saveScope(): void { localStorage.setItem("pharma_oa.scope", JSON.stringify(scope)); ElMessage.success(t("updated")); void loadCurrent(); }
function formatDate(value?: string): string { return value ? value.slice(0, 10) : "-"; }
function secondary(item: WorkspaceRecord): string {
  if ("departmentId" in item) return `${item.departmentId} · ${item.positionId}`;
  if ("unifiedSocialCreditCode" in item && "region" in item) return `${item.region} · ${item.unifiedSocialCreditCode}`;
  if ("sku" in item) return `${item.genericName} · ${item.specification}`;
  if ("validTo" in item) return `${item.subjectType} · ${item.subjectId}`;
  if ("subjectType" in item) return `${item.subjectType} · ${item.businessGate}`;
  return "description" in item ? item.description || "-" : "-";
}
function statusActions(item: WorkspaceRecord): Array<{ key: string; label: string }> {
  if (activeModule.value === "qualification") {
    const map: Record<string, string[]> = { draft: ["submit"], rejected: ["submit"], pending: ["approve", "reject"], approved: ["revoke"] };
    return (map[recordStatus(item)] ?? []).map((key) => ({ key, label: t(key as "submit" | "approve" | "reject" | "revoke") }));
  }
  if (activeModule.value === "employee") return recordStatus(item) === "left" ? [] : [{ key: "leave", label: t("leave") }];
  return [{ key: recordStatus(item) === "disabled" ? "enable" : "disable", label: recordStatus(item) === "disabled" ? t("active") : t("disabled") }];
}

watch(activeModule, () => { keyword.value = ""; status.value = ""; selected.value = null; void loadCurrent(); });
watch(workspaceArea, (value) => {
  if (value === "master") void loadCurrent();
});
let searchTimer: number | undefined;
watch([keyword, status], () => { window.clearTimeout(searchTimer); searchTimer = window.setTimeout(() => void loadCurrent(), 280); });
onMounted(() => {
  try { Object.assign(scope, JSON.parse(localStorage.getItem("pharma_oa.scope") ?? "{}")); } catch { localStorage.removeItem("pharma_oa.scope"); }
  applyTheme(); setLocale(window.__SKOLL_HOST__?.locale ?? "zh-CN");
});
window.addEventListener("skoll:theme", applyTheme);
window.addEventListener("skoll:host-ready", applyTheme);
</script>

<template>
  <el-config-provider :locale="elementLocale">
    <main class="workspace" data-testid="pharma-workspace">
      <header class="workspace-header">
        <div class="brand-lockup">
          <span class="brand-mark"><BadgeCheck aria-hidden="true" /></span>
          <div><p>{{ t("brand") }}</p><h1>{{ t("businessWorkspace") }}</h1></div>
        </div>
        <div class="header-actions">
          <el-tooltip :content="t('locale')"><el-button :icon="Languages" circle :aria-label="t('locale')" @click="setLocale(language === 'zh-CN' ? 'en-US' : 'zh-CN')" /></el-tooltip>
          <el-tooltip :content="t('refresh')"><el-button :icon="RefreshCw" circle :loading="loading" :aria-label="t('refresh')" @click="loadCurrent" /></el-tooltip>
        </div>
      </header>

      <nav class="module-tabs primary-tabs" :aria-label="t('businessWorkspace')">
        <button type="button" :class="{ active: workspaceArea === 'requests' }" :aria-current="workspaceArea === 'requests' ? 'page' : undefined" @click="workspaceArea = 'requests'">
          <FileText aria-hidden="true" /><span>{{ t("requestCenter") }}</span>
        </button>
        <button type="button" :class="{ active: workspaceArea === 'inbox' }" :aria-current="workspaceArea === 'inbox' ? 'page' : undefined" @click="workspaceArea = 'inbox'">
          <Inbox aria-hidden="true" /><span>{{ t("approvalInbox") }}</span>
        </button>
        <button type="button" :class="{ active: workspaceArea === 'master' }" :aria-current="workspaceArea === 'master' ? 'page' : undefined" @click="workspaceArea = 'master'">
          <Database aria-hidden="true" /><span>{{ t("masterData") }}</span>
        </button>
      </nav>

      <nav v-if="workspaceArea === 'master'" class="module-tabs secondary-tabs" :aria-label="t('workspace')">
        <button v-for="item in nav" :key="item.key" type="button" :class="{ active: activeModule === item.key }" :aria-current="activeModule === item.key ? 'page' : undefined" @click="activeModule = item.key">
          <component :is="item.icon" aria-hidden="true" /><span>{{ t(item.key) }}</span>
        </button>
      </nav>

      <section class="scope-band" :aria-label="t('scope')">
        <strong>{{ t("scope") }}</strong>
        <el-input v-model="scope.tenantId" :placeholder="t('tenant')" :aria-label="t('tenant')" clearable />
        <el-input v-model="scope.organizationId" :placeholder="t('organization')" :aria-label="t('organization')" clearable />
        <el-button @click="saveScope">{{ t("apply") }}</el-button>
      </section>

      <template v-if="workspaceArea === 'master'">
        <section class="module-heading">
          <div><p>{{ t("brand") }} · {{ t("workspace") }}</p><h2>{{ moduleLabel }}</h2></div>
          <div class="heading-actions">
            <el-button v-if="activeModule === 'qualification'" :icon="ScanSearch" :loading="saving" @click="scanExpiry">{{ t("scan") }}</el-button>
            <el-button type="primary" :icon="Plus" @click="openCreate">{{ t("create") }}</el-button>
          </div>
        </section>

        <section class="metrics" aria-label="Key metrics">
          <article><span>{{ t("total") }}</span><strong>{{ items.length }}</strong><Boxes /></article>
          <article><span>{{ t("enabled") }}</span><strong>{{ activeCount }}</strong><ClipboardCheck /></article>
          <article><span>{{ t("attention") }}</span><strong>{{ attentionCount }}</strong><CircleAlert /></article>
          <article><span>{{ t("compliance") }}</span><strong>{{ complianceRate }}%</strong><ShieldCheck /></article>
        </section>

        <section class="toolbar">
          <el-input v-model="keyword" :prefix-icon="Search" :placeholder="t('search')" clearable aria-label="Search" />
          <el-select v-model="status" :placeholder="t('all')" clearable aria-label="Status">
            <el-option v-for="value in statusOptions" :key="value" :label="statusLabel(value)" :value="value" />
          </el-select>
        </section>

        <section v-if="loading" class="state-panel" aria-live="polite"><el-skeleton :rows="6" animated /><span>{{ t("loading") }}</span></section>
        <section v-else-if="error" class="state-panel state-error" role="alert">
          <CircleAlert /><strong>{{ denied ? t("denied") : t("failed") }}</strong><p>{{ denied ? t("deniedHint") : error }}</p><el-button @click="loadCurrent">{{ t("retry") }}</el-button>
        </section>
        <section v-else-if="items.length === 0" class="state-panel"><el-empty :description="t('empty')" /><p>{{ t("emptyHint") }}</p><el-button type="primary" :icon="Plus" @click="openCreate">{{ t("create") }}</el-button></section>

        <section v-else class="data-region">
        <el-table :data="items" stripe class="desktop-table" row-key="id" @row-dblclick="showDetails">
          <el-table-column :label="t('identity')" min-width="220">
            <template #default="{ row }"><button class="identity-link" type="button" @click="showDetails(row)"><strong>{{ recordName(row) }}</strong><span>{{ recordCode(row) }}</span></button></template>
          </el-table-column>
          <el-table-column :label="t('detail')" min-width="260"><template #default="{ row }">{{ secondary(row) }}</template></el-table-column>
          <el-table-column v-if="activeModule === 'employee'" :label="t('contact')" min-width="210"><template #default="{ row }">{{ row.phone || row.email || t('noData') }}</template></el-table-column>
          <el-table-column v-else-if="activeModule === 'product'" :label="t('validity')" min-width="180"><template #default="{ row }">{{ row.approvalNumber }}</template></el-table-column>
          <el-table-column v-else-if="activeModule === 'qualification'" :label="t('validity')" min-width="180"><template #default="{ row }">{{ formatDate(row.validFrom) }} - {{ formatDate(row.validTo) }}</template></el-table-column>
          <el-table-column v-else-if="activeModule === 'qualificationType'" :label="t('validity')" min-width="160"><template #default="{ row }">{{ row.validityDays }}d / {{ row.alertDays }}d</template></el-table-column>
          <el-table-column :label="t('status')" width="120"><template #default="{ row }"><el-tag :type="statusType(recordStatus(row))" effect="light">{{ statusLabel(recordStatus(row)) }}</el-tag></template></el-table-column>
          <el-table-column :label="t('actions')" width="230" fixed="right">
            <template #default="{ row }">
              <el-tooltip :content="t('edit')"><el-button :icon="Edit3" text circle :aria-label="t('edit')" @click="openEdit(row)" /></el-tooltip>
              <el-tooltip :content="t('detail')"><el-button :icon="Eye" text circle :aria-label="t('detail')" @click="showDetails(row)" /></el-tooltip>
              <el-tooltip v-if="activeModule === 'employee'" :content="t('upload')"><el-button :icon="FileUp" text circle :aria-label="t('upload')" @click="openAttachment(row)" /></el-tooltip>
              <el-button v-for="action in statusActions(row)" :key="action.key" text :type="['reject','revoke','disable','leave'].includes(action.key) ? 'danger' : 'primary'" @click="openStatus(row, action.key)">{{ action.label }}</el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="mobile-records">
          <article v-for="item in items" :key="item.id" class="record-card">
            <button type="button" @click="showDetails(item)"><strong>{{ recordName(item) }}</strong><span>{{ recordCode(item) }}</span></button>
            <p>{{ secondary(item) }}</p><el-tag :type="statusType(recordStatus(item))">{{ statusLabel(recordStatus(item)) }}</el-tag>
            <footer><el-button :icon="Edit3" @click="openEdit(item)">{{ t("edit") }}</el-button><el-button :icon="Eye" @click="showDetails(item)">{{ t("detail") }}</el-button></footer>
          </article>
        </div>
        </section>
      </template>

      <OAWorkspace v-else :mode="workspaceArea" :scope="{ tenantId: scope.tenantId, organizationId: scope.organizationId }" />

      <el-dialog v-if="workspaceArea === 'master'" v-model="dialogVisible" :title="dialogTitle" width="min(760px, calc(100vw - 32px))" destroy-on-close align-center>
        <el-form v-if="dialog === 'editor'" label-position="top" class="editor-grid" @submit.prevent="saveEditor">
          <el-form-item v-for="field in fields" :key="field.key" :label="label(field)" :required="field.required" :class="{ wide: field.span === 2 }">
            <el-input v-if="field.key !== 'evidenceName' && (!field.type || field.type === 'text')" v-model="form[field.key]" maxlength="255" clearable />
            <el-input v-else-if="field.type === 'textarea'" v-model="form[field.key]" type="textarea" :rows="3" maxlength="512" show-word-limit />
            <el-input-number v-else-if="field.type === 'number'" v-model="form[field.key]" :min="field.min" :max="field.max" controls-position="right" />
            <el-date-picker v-else-if="field.type === 'date'" v-model="form[field.key]" type="date" value-format="YYYY-MM-DD" />
            <el-select v-else-if="field.type === 'select'" v-model="form[field.key]" filterable clearable>
              <el-option v-for="option in fieldOptions(field)" :key="fieldOptionValue(option)" :label="fieldOptionLabel(option)" :value="fieldOptionValue(option)" />
            </el-select>
            <el-checkbox v-else-if="field.type === 'checkbox'" v-model="form[field.key]">{{ label(field) }}</el-checkbox>
            <label v-if="field.key === 'evidenceName'" class="file-picker"><FileUp /><span>{{ form.evidenceName || t("evidence") }}</span><input type="file" accept="application/pdf,image/jpeg,image/png" @change="readFile(($event.target as HTMLInputElement).files![0], 'evidence')" /></label>
          </el-form-item>
        </el-form>
        <el-form v-else-if="dialog === 'attachment'" label-position="top"><el-form-item :label="t('upload')" required><label class="file-picker"><FileUp /><span>{{ form.attachmentName || t("evidence") }}</span><input type="file" @change="readFile(($event.target as HTMLInputElement).files![0], 'attachment')" /></label></el-form-item></el-form>
        <el-form v-else label-position="top"><el-form-item :label="dialog === 'review' ? t('comment') : t('statusReason')" :required="transitionAction !== 'approve' && transitionAction !== 'enable'"><el-input v-model="form[dialog === 'review' ? 'comment' : 'reason']" type="textarea" :rows="4" maxlength="500" show-word-limit /></el-form-item><el-form-item v-if="dialog === 'review' && transitionAction === 'approve'" :label="t('reviewer')"><el-input v-model="form.reviewer" /></el-form-item></el-form>
        <template #footer><el-button @click="dialog = null">{{ t("cancel") }}</el-button><el-button type="primary" :loading="saving" @click="dialog === 'editor' ? saveEditor() : dialog === 'attachment' ? uploadAttachment() : submitTransition()">{{ t("confirm") }}</el-button></template>
      </el-dialog>

      <el-drawer v-if="workspaceArea === 'master'" v-model="detailVisible" :title="moduleLabel" size="min(520px, 92vw)" direction="rtl">
        <div v-if="selected" class="detail-sheet">
          <div class="detail-title"><span>{{ recordCode(selected) }}</span><h3>{{ recordName(selected) }}</h3><el-tag :type="statusType(recordStatus(selected))">{{ statusLabel(recordStatus(selected)) }}</el-tag></div>
          <dl><template v-for="field in fields" :key="field.key"><dt>{{ label(field) }}</dt><dd>{{ String((selected as unknown as Record<string, unknown>)[field.key] ?? t('noData')) }}</dd></template></dl>
        </div>
      </el-drawer>
    </main>
  </el-config-provider>
</template>
