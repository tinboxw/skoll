import { computed, ref } from "vue";
import type { Locale } from "./types";

const oaMessages = {
  "zh-CN": {
    businessWorkspace: "业务工作台",
    requestCenter: "申请中心",
    approvalInbox: "待我审批",
    masterData: "主数据",
    oaRequestTitle: "OA 申请与审批",
    oaRequestDescription: "统一处理请假、报销、采购、合同和自定义申请",
    oaInboxDescription: "聚焦待处理事项，连续完成审批与转交",
    newRequest: "新建申请",
    requestType: "申请类型",
    leaveRequest: "请假申请",
    expenseRequest: "报销申请",
    procurementRequest: "采购申请",
    contractRequest: "合同申请",
    customRequest: "自定义申请",
    requestTitle: "申请标题",
    requestDescription: "申请说明",
    approverId: "审批人账号",
    approverName: "审批人姓名",
    leaveType: "请假类型",
    annualLeave: "年假",
    sickLeave: "病假",
    personalLeave: "事假",
    startDate: "开始日期",
    endDate: "结束日期",
    amount: "金额",
    expenseCategory: "费用类别",
    travelExpense: "差旅费",
    officeExpense: "办公费",
    hospitalityExpense: "业务招待费",
    procurementPurpose: "采购用途",
    counterparty: "合同相对方",
    effectiveDate: "生效日期",
    customFormKey: "表单标识",
    customContent: "补充内容",
    saveDraft: "保存草稿",
    submitRequest: "提交申请",
    withdrawRequest: "撤回申请",
    cancelRequest: "取消申请",
    approveRequest: "通过",
    rejectRequest: "驳回",
    delegateRequest: "转交",
    requestDetail: "申请详情",
    workflowTimeline: "审批轨迹",
    collaboration: "协作记录",
    currentApprover: "当前审批人",
    createdTime: "创建时间",
    submittedTime: "提交时间",
    reminderTime: "提醒时间",
    addComment: "发表评论",
    commentPlaceholder: "补充业务信息或审批说明",
    sendComment: "发送",
    addAttachment: "添加附件",
    scheduleReminder: "设置提醒",
    actionComment: "处理意见",
    rejectionReason: "驳回原因",
    withdrawalReason: "撤回原因",
    cancellationReason: "取消原因",
    delegateTargetId: "转交人账号",
    delegateTargetName: "转交人姓名",
    pendingCount: "待处理",
    draftCount: "草稿",
    finishedCount: "已完成",
    allRequestTypes: "全部类型",
    allRequestStatuses: "全部状态",
    searchRequests: "搜索标题或说明",
    noRequests: "暂无申请",
    noRequestsHint: "新建一条申请，或调整筛选条件。",
    noInbox: "没有待审批事项",
    noInboxHint: "当前业务范围内没有需要处理的申请。",
    requestSaved: "申请草稿已保存",
    requestSubmitted: "申请已提交",
    actionCompleted: "处理已完成",
    commentAdded: "评论已添加",
    attachmentAdded: "附件已添加",
    reminderScheduled: "提醒已设置",
    confirmSubmit: "提交后将进入审批流程，是否继续？",
    confirmWithdraw: "确认撤回当前申请？",
    confirmCancel: "确认取消当前申请？",
    confirmApprove: "确认通过当前申请？",
    fileLimit: "附件不能超过 5 MB",
    basicInformation: "基本信息",
    requestContent: "申请内容",
    attachments: "附件",
    comments: "评论",
    noTimeline: "尚无审批轨迹",
    noComments: "尚无评论",
    noAttachments: "尚无附件",
    requiredFields: "请完整填写申请类型、标题、审批人和业务必填项",
    withdrawn: "已撤回",
    canceled: "已取消",
    running: "审批中",
    delegated: "已委派",
    start: "发起申请",
    withdraw: "撤回",
    delegate: "委派",
    substitute: "代办",
    escalate: "升级",
    copy: "抄送",
    openDetail: "查看详情"
    ,purchaseWorkspace: "采购与入库"
    ,purchaseTitle: "采购执行工作台"
    ,purchaseDescription: "从合规采购申请、审批到订单分批入库连续处理"
    ,purchaseRequests: "采购申请"
    ,purchaseOrders: "采购订单"
    ,inboundRecords: "入库记录"
    ,newPurchaseRequest: "新建采购申请"
    ,supplier: "供应商"
    ,purchaseReason: "采购事由"
    ,approver: "审批人"
    ,productLines: "采购明细"
    ,product: "药品"
    ,quantity: "数量"
    ,unitPrice: "含税单价"
    ,lineAmount: "金额"
    ,addLine: "添加明细"
    ,removeLine: "删除明细"
    ,purchaseNumber: "单据编号"
    ,purchaseTotal: "采购金额"
    ,purchaseProgress: "入库进度"
    ,pendingApproval: "待审批"
    ,openOrders: "待入库"
    ,partialOrders: "部分入库"
    ,completedOrders: "已完成"
    ,receiveGoods: "办理入库"
    ,warehouse: "仓库"
    ,warehouseArea: "库区"
    ,warehouseLocation: "库位"
    ,batchNumber: "批号"
    ,productionDate: "生产日期"
    ,expiryDate: "有效期至"
    ,receivedQuantity: "本次入库"
    ,remainingQuantity: "待入库"
    ,inboundEvidence: "入库凭证"
    ,submitInbound: "确认入库"
    ,decisionComment: "审批意见"
    ,purchaseCreated: "采购申请已创建"
    ,purchaseApproved: "采购申请已通过并生成订单"
    ,purchaseRejected: "采购申请已驳回"
    ,inboundCreated: "入库单已生成"
    ,qualificationBlocked: "资质校验未通过，请先修复供应商、药品或生产企业资质"
    ,staleConflict: "数据已被其他人更新，请刷新后重试"
    ,purchaseRequired: "请完整填写供应商、审批人、采购事由和有效采购明细"
    ,inboundRequired: "请完整填写库位、批号、日期和至少一条入库数量"
    ,noPurchaseRequests: "暂无采购申请"
    ,noPurchaseOrders: "暂无采购订单"
    ,noInboundRecords: "暂无入库记录"
    ,searchPurchase: "搜索单号、供应商或事由"
    ,orderDetail: "采购订单详情"
    ,requestDetailTitle: "采购申请详情"
    ,inboundHistory: "入库历史"
    ,allPurchaseStatuses: "全部采购状态"
    ,currencyCNY: "人民币"
    ,orderedQuantity: "订购数量"
    ,receivedTotal: "已入库"
    ,viewOrder: "查看订单"
    ,open: "待入库"
    ,partial: "部分入库"
    ,received: "已入库"
    ,completed: "已完成"
  },
  "en-US": {
    businessWorkspace: "Business Workspace",
    requestCenter: "Request Center",
    approvalInbox: "Approval Inbox",
    masterData: "Master Data",
    oaRequestTitle: "OA Requests and Approvals",
    oaRequestDescription: "Handle leave, expense, procurement, contract, and custom requests",
    oaInboxDescription: "Focus on pending work and process approvals efficiently",
    newRequest: "New Request",
    requestType: "Request Type",
    leaveRequest: "Leave Request",
    expenseRequest: "Expense Request",
    procurementRequest: "Procurement Request",
    contractRequest: "Contract Request",
    customRequest: "Custom Request",
    requestTitle: "Title",
    requestDescription: "Description",
    approverId: "Approver ID",
    approverName: "Approver Name",
    leaveType: "Leave Type",
    annualLeave: "Annual Leave",
    sickLeave: "Sick Leave",
    personalLeave: "Personal Leave",
    startDate: "Start Date",
    endDate: "End Date",
    amount: "Amount",
    expenseCategory: "Expense Category",
    travelExpense: "Travel",
    officeExpense: "Office",
    hospitalityExpense: "Hospitality",
    procurementPurpose: "Procurement Purpose",
    counterparty: "Counterparty",
    effectiveDate: "Effective Date",
    customFormKey: "Form Key",
    customContent: "Additional Content",
    saveDraft: "Save Draft",
    submitRequest: "Submit",
    withdrawRequest: "Withdraw",
    cancelRequest: "Cancel Request",
    approveRequest: "Approve",
    rejectRequest: "Reject",
    delegateRequest: "Delegate",
    requestDetail: "Request Detail",
    workflowTimeline: "Approval Timeline",
    collaboration: "Collaboration",
    currentApprover: "Current Approver",
    createdTime: "Created",
    submittedTime: "Submitted",
    reminderTime: "Reminder",
    addComment: "Add Comment",
    commentPlaceholder: "Add business context or an approval note",
    sendComment: "Send",
    addAttachment: "Add Attachment",
    scheduleReminder: "Schedule Reminder",
    actionComment: "Comment",
    rejectionReason: "Rejection Reason",
    withdrawalReason: "Withdrawal Reason",
    cancellationReason: "Cancellation Reason",
    delegateTargetId: "Delegate ID",
    delegateTargetName: "Delegate Name",
    pendingCount: "Pending",
    draftCount: "Drafts",
    finishedCount: "Completed",
    allRequestTypes: "All Types",
    allRequestStatuses: "All Statuses",
    searchRequests: "Search title or description",
    noRequests: "No Requests",
    noRequestsHint: "Create a request or change the filters.",
    noInbox: "Inbox Is Clear",
    noInboxHint: "There are no pending requests in this business scope.",
    requestSaved: "Draft saved",
    requestSubmitted: "Request submitted",
    actionCompleted: "Action completed",
    commentAdded: "Comment added",
    attachmentAdded: "Attachment added",
    reminderScheduled: "Reminder scheduled",
    confirmSubmit: "Submit this request for approval?",
    confirmWithdraw: "Withdraw this request?",
    confirmCancel: "Cancel this request?",
    confirmApprove: "Approve this request?",
    fileLimit: "Attachments cannot exceed 5 MB",
    basicInformation: "Basic Information",
    requestContent: "Request Content",
    attachments: "Attachments",
    comments: "Comments",
    noTimeline: "No approval activity yet",
    noComments: "No comments yet",
    noAttachments: "No attachments yet",
    requiredFields: "Complete the request type, title, approver, and required business fields",
    withdrawn: "Withdrawn",
    canceled: "Canceled",
    running: "Running",
    delegated: "Delegated",
    start: "Started",
    withdraw: "Withdrawn",
    delegate: "Delegate",
    substitute: "Substitute",
    escalate: "Escalate",
    copy: "Copied",
    openDetail: "View details"
    ,purchaseWorkspace: "Purchasing & Receiving"
    ,purchaseTitle: "Purchase Operations"
    ,purchaseDescription: "Process compliant requests, approvals, orders, and partial receiving"
    ,purchaseRequests: "Purchase Requests"
    ,purchaseOrders: "Purchase Orders"
    ,inboundRecords: "Inbound Records"
    ,newPurchaseRequest: "New Purchase Request"
    ,supplier: "Supplier"
    ,purchaseReason: "Purchase Reason"
    ,approver: "Approver"
    ,productLines: "Purchase Lines"
    ,product: "Product"
    ,quantity: "Quantity"
    ,unitPrice: "Unit Price"
    ,lineAmount: "Amount"
    ,addLine: "Add Line"
    ,removeLine: "Remove Line"
    ,purchaseNumber: "Document Number"
    ,purchaseTotal: "Purchase Total"
    ,purchaseProgress: "Receiving Progress"
    ,pendingApproval: "Pending Approval"
    ,openOrders: "Open Orders"
    ,partialOrders: "Partially Received"
    ,completedOrders: "Completed"
    ,receiveGoods: "Receive"
    ,warehouse: "Warehouse"
    ,warehouseArea: "Area"
    ,warehouseLocation: "Location"
    ,batchNumber: "Batch Number"
    ,productionDate: "Production Date"
    ,expiryDate: "Expiry Date"
    ,receivedQuantity: "Receive Now"
    ,remainingQuantity: "Remaining"
    ,inboundEvidence: "Receiving Evidence"
    ,submitInbound: "Confirm Receipt"
    ,decisionComment: "Decision Comment"
    ,purchaseCreated: "Purchase request created"
    ,purchaseApproved: "Request approved and order created"
    ,purchaseRejected: "Purchase request rejected"
    ,inboundCreated: "Inbound receipt created"
    ,qualificationBlocked: "Qualification check failed. Repair supplier, product, or manufacturer qualifications first"
    ,staleConflict: "This record was updated by another user. Refresh and retry"
    ,purchaseRequired: "Complete supplier, approver, reason, and valid purchase lines"
    ,inboundRequired: "Complete location, batch dates, and at least one receiving quantity"
    ,noPurchaseRequests: "No purchase requests"
    ,noPurchaseOrders: "No purchase orders"
    ,noInboundRecords: "No inbound records"
    ,searchPurchase: "Search number, supplier, or reason"
    ,orderDetail: "Purchase Order Detail"
    ,requestDetailTitle: "Purchase Request Detail"
    ,inboundHistory: "Inbound History"
    ,allPurchaseStatuses: "All Purchase Statuses"
    ,currencyCNY: "CNY"
    ,orderedQuantity: "Ordered"
    ,receivedTotal: "Received"
    ,viewOrder: "View Order"
    ,open: "Open"
    ,partial: "Partial"
    ,received: "Received"
    ,completed: "Completed"
  }
} as const;

const messages = {
  "zh-CN": {
    ...oaMessages["zh-CN"],
    brand: "医药 OA", workspace: "主数据工作台", scope: "业务范围", tenant: "租户", organization: "组织", apply: "应用", search: "搜索当前模块", all: "全部状态", active: "启用中", disabled: "已停用", on_leave: "休假", left: "已离职", draft: "草稿", pending: "待审核", approved: "已通过", rejected: "已驳回", revoked: "已撤销", expired: "已到期", refresh: "刷新", create: "新建", edit: "编辑", actions: "操作", save: "保存", cancel: "取消", confirm: "确认", retry: "重试", loading: "正在加载业务数据", empty: "暂无业务数据", emptyHint: "新建首条记录，开始维护当前模块。", denied: "缺少当前模块访问权限", deniedHint: "请联系管理员配置对应的医药 OA 权限。", failed: "数据加载失败", saved: "数据已保存", updated: "状态已更新", employee: "员工管理", customer: "客户管理", supplier: "供应商管理", product: "药品管理", category: "药品分类", unit: "计量单位", manufacturer: "生产企业", qualification: "资质台账", qualificationType: "资质类型", total: "记录总数", enabled: "当前有效", attention: "需要关注", compliance: "合规覆盖", code: "编码", name: "名称", status: "状态", identity: "业务标识", detail: "业务详情", contact: "联系方式", validity: "有效期限", evidence: "证据文件", assignment: "任职信息", noData: "未填写", statusReason: "状态变更原因", leave: "办理离职", upload: "上传附件", submit: "提交审核", approve: "审核通过", reject: "驳回", revoke: "撤销", scan: "扫描到期资质", reviewer: "审核人", comment: "审核意见", expiryCenter: "效期中心", expiring: "即将到期", formRequired: "请完整填写必填项", dangerConfirm: "该操作会立即影响业务准入，是否继续？", locale: "语言", close: "关闭"
  },
  "en-US": {
    ...oaMessages["en-US"],
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
