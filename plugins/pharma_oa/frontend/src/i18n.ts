import { computed, ref } from "vue";
import type { Locale } from "./types";

const messages = {
  "zh-CN": {
    brand: "医药 OA", workspace: "主数据工作台", scope: "业务范围", tenant: "租户", organization: "组织", apply: "应用", search: "搜索当前模块", all: "全部状态", active: "启用中", disabled: "已停用", on_leave: "休假", left: "已离职", draft: "草稿", pending: "待审核", approved: "已通过", rejected: "已驳回", revoked: "已撤销", expired: "已到期", refresh: "刷新", create: "新建", edit: "编辑", actions: "操作", save: "保存", cancel: "取消", confirm: "确认", retry: "重试", loading: "正在加载业务数据", empty: "暂无业务数据", emptyHint: "新建首条记录，开始维护当前模块。", denied: "缺少当前模块访问权限", deniedHint: "请联系管理员配置对应的医药 OA 权限。", failed: "数据加载失败", saved: "数据已保存", updated: "状态已更新", employee: "员工管理", customer: "客户管理", supplier: "供应商管理", product: "药品管理", category: "药品分类", unit: "计量单位", manufacturer: "生产企业", qualification: "资质台账", qualificationType: "资质类型", total: "记录总数", enabled: "当前有效", attention: "需要关注", compliance: "合规覆盖", code: "编码", name: "名称", status: "状态", identity: "业务标识", detail: "业务详情", contact: "联系方式", validity: "有效期限", evidence: "证据文件", assignment: "任职信息", noData: "未填写", statusReason: "状态变更原因", leave: "办理离职", upload: "上传附件", submit: "提交审核", approve: "审核通过", reject: "驳回", revoke: "撤销", scan: "扫描到期资质", reviewer: "审核人", comment: "审核意见", expiryCenter: "效期中心", expiring: "即将到期", formRequired: "请完整填写必填项", dangerConfirm: "该操作会立即影响业务准入，是否继续？", locale: "语言", close: "关闭"
  },
  "en-US": {
    brand: "Pharma OA", workspace: "Master Data Workspace", scope: "Business scope", tenant: "Tenant", organization: "Organization", apply: "Apply", search: "Search this module", all: "All statuses", active: "Active", disabled: "Disabled", on_leave: "On leave", left: "Departed", draft: "Draft", pending: "Pending", approved: "Approved", rejected: "Rejected", revoked: "Revoked", expired: "Expired", refresh: "Refresh", create: "New", edit: "Edit", actions: "Actions", save: "Save", cancel: "Cancel", confirm: "Confirm", retry: "Retry", loading: "Loading business data", empty: "No business records", emptyHint: "Create the first record for this module.", denied: "Access required", deniedHint: "Ask an administrator to grant the required Pharma OA permission.", failed: "Unable to load data", saved: "Record saved", updated: "Status updated", employee: "Employees", customer: "Customers", supplier: "Suppliers", product: "Products", category: "Categories", unit: "Units", manufacturer: "Manufacturers", qualification: "Qualification Ledger", qualificationType: "Qualification Types", total: "Total records", enabled: "Currently valid", attention: "Needs attention", compliance: "Compliance coverage", code: "Code", name: "Name", status: "Status", identity: "Business identity", detail: "Business detail", contact: "Contact", validity: "Validity", evidence: "Evidence", assignment: "Assignment", noData: "Not provided", statusReason: "Status change reason", leave: "Process departure", upload: "Upload attachment", submit: "Submit", approve: "Approve", reject: "Reject", revoke: "Revoke", scan: "Scan expirations", reviewer: "Reviewer", comment: "Review comment", expiryCenter: "Expiry center", expiring: "Expiring soon", formRequired: "Complete all required fields", dangerConfirm: "This action immediately changes business eligibility. Continue?", locale: "Language", close: "Close"
  }
} as const;

export type MessageKey = keyof typeof messages["zh-CN"];
export const language = ref<Locale>(window.__SKOLL_HOST__?.locale === "en-US" ? "en-US" : "zh-CN");
export const elementLanguage = computed(() => language.value);
export function t(key: MessageKey): string { return messages[language.value][key]; }
export function setLocale(value: string): void {
  language.value = value === "en-US" ? "en-US" : "zh-CN";
  document.documentElement.lang = language.value;
}

window.addEventListener("skoll:locale", (event) => setLocale((event as CustomEvent<{ locale?: string }>).detail?.locale ?? "zh-CN"));
window.addEventListener("skoll:host-ready", () => setLocale(window.__SKOLL_HOST__?.locale ?? "zh-CN"));
