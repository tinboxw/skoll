<script setup lang="ts">
import { useI18n } from "../../i18n";

import { computed, onMounted, reactive, ref } from "vue";
import { useRoute } from "vue-router";
import { Banknote, Ban, BellRing, Eye, FilePlus2, FileText, RefreshCw, RotateCcw, Upload } from "lucide-vue-next";
import { ElMessage, ElMessageBox, type UploadFile } from "element-plus";
import { DataTable, DetailDrawer, PageShell, type DataTableColumn } from "../../components/Common";
import { uploadFile } from "../../files/api";
import { useButtonAccess } from "../../permissions/button";
import {
	createInvoiceRecord, createPaymentPlan, listInvoiceRecords, listPaymentPlans, listPaymentReminderJobs, listSalesOrders, recordPayment,
	retryPaymentReminderScan, runPaymentReminderScan, voidInvoiceRecord,
	type FinancialAttachment, type InvoiceRecord, type InvoiceRecordStatus, type PaymentPlan, type PaymentPlanStatus, type PaymentReminderJob, type SalesOrder
} from "../../pharma-oa/api";
import { useUserStore } from "../../stores/user";
import { toErrorMessage } from "../../utils/common";

const { t, valueLabel } = useI18n();

type FinanceTab = "plans" | "invoices" | "jobs";
type PlanRow = Record<string, unknown> & PaymentPlan & { amountText: string; paidText: string; dueText: string };
type InvoiceRow = Record<string, unknown> & InvoiceRecord & { amountText: string; issuedText: string };
type JobRow = Record<string, unknown> & PaymentReminderJob & { createdText: string; resultText: string };

const access = useButtonAccess();
const userStore = useUserStore();
const route = useRoute();
const loading = ref(false), error = ref(""), activeTab = ref<FinanceTab>("plans"), keyword = ref("");
const statusFilter = ref<PaymentPlanStatus | InvoiceRecordStatus | "">("");
const saving = ref(""), actionId = ref(""), detailOpen = ref(false), planOpen = ref(false), receiptOpen = ref(false), invoiceOpen = ref(false), scanOpen = ref(false);
const plans = ref<PaymentPlan[]>([]), invoices = ref<InvoiceRecord[]>([]), jobs = ref<PaymentReminderJob[]>([]), orders = ref<SalesOrder[]>([]);
const selectedPlan = ref<PaymentPlan | null>(null), selectedInvoice = ref<InvoiceRecord | null>(null), selectedJob = ref<PaymentReminderJob | null>(null);
const planFile = ref<File | null>(null), receiptFile = ref<File | null>(null), invoiceFile = ref<File | null>(null);
const actorId = computed(() => userStore.profile?.id?.trim() || "system");
const canRead = computed(() => access.can("pharma_oa.payment_invoice.read"));
const canCreatePlan = computed(() => access.can("pharma_oa.payment_plan.create"));
const canReceive = computed(() => access.can("pharma_oa.payment_plan.receive"));
const canCreateInvoice = computed(() => access.can("pharma_oa.invoice_record.create"));
const canVoidInvoice = computed(() => access.can("pharma_oa.invoice_record.void"));
const canRun = computed(() => access.can("pharma_oa.payment_reminder.run"));
const ownedOrders = computed(() => orders.value.filter((item) => item.createdBy === actorId.value));
const planForm = reactive({ salesOrderId: "", amount: 0, dueAt: null as Date | null, note: "" });
const receiptForm = reactive({ amount: 0, paidAt: new Date() as Date | null, reference: "" });
const invoiceForm = reactive({ number: "", salesOrderId: "", amount: 0, issuedAt: new Date() as Date | null, note: "" });
const scanForm = reactive({ recipientId: "" });
const planRows = computed<PlanRow[]>(() => plans.value.map((item) => ({ ...item, amountText: money(item.amountCents), paidText: money(item.paidAmountCents), dueText: date(item.dueAt) })));
const invoiceRows = computed<InvoiceRow[]>(() => invoices.value.map((item) => ({ ...item, amountText: money(item.amountCents), issuedText: date(item.issuedAt) })));
const jobRows = computed<JobRow[]>(() => jobs.value.map((item) => ({ ...item, createdText: date(item.createdAt), resultText: `${item.matchedCount} / ${item.createdCount}` })));
const planColumns: DataTableColumn[] = [{ key: "salesOrderNumber", label: t("pharma.paymentInvoice.salesOrder"), minWidth: 150 }, { key: "amountText", label: t("pharma.paymentInvoice.planned"), width: 130 }, { key: "paidText", label: t("pharma.paymentInvoice.received"), width: 130 }, { key: "dueText", label: t("pharma.paymentInvoice.due"), minWidth: 170 }, { key: "status", label: t("pharma.paymentInvoice.status"), width: 115 }];
const invoiceColumns: DataTableColumn[] = [{ key: "number", label: t("pharma.paymentInvoice.invoice"), minWidth: 150 }, { key: "salesOrderNumber", label: t("pharma.paymentInvoice.salesOrder"), minWidth: 150 }, { key: "amountText", label: t("pharma.paymentInvoice.amount"), width: 130 }, { key: "issuedText", label: t("pharma.paymentInvoice.issued"), minWidth: 170 }, { key: "status", label: t("pharma.paymentInvoice.status"), width: 110 }];
const jobColumns: DataTableColumn[] = [{ key: "createdText", label: t("pharma.paymentInvoice.created"), minWidth: 170 }, { key: "recipientId", label: t("pharma.paymentInvoice.recipient"), minWidth: 150 }, { key: "resultText", label: t("pharma.paymentInvoice.matchedNew"), width: 145 }, { key: "retryCount", label: t("pharma.paymentInvoice.retries"), width: 90 }, { key: "status", label: t("pharma.paymentInvoice.status"), width: 115 }];
const outstandingCents = computed(() => plans.value.reduce((sum, item) => sum + Math.max(0, item.amountCents - item.paidAmountCents), 0));
const overdueCount = computed(() => plans.value.filter((item) => item.status === "overdue").length);
const issuedCents = computed(() => invoices.value.filter((item) => item.status === "issued").reduce((sum, item) => sum + item.amountCents, 0));

onMounted(() => void refresh());

async function refresh() {
	if (!canRead.value) return;
	loading.value = true;
	error.value = "";
	try {
		const [planItems, invoiceItems, jobItems, orderItems] = await Promise.all([
			listPaymentPlans({ keyword: keyword.value, status: activeTab.value === "plans" ? statusFilter.value as PaymentPlanStatus | "" : "" }),
			listInvoiceRecords({ keyword: keyword.value, status: activeTab.value === "invoices" ? statusFilter.value as InvoiceRecordStatus | "" : "" }),
			listPaymentReminderJobs(), listSalesOrders()
		]);
		plans.value = planItems; invoices.value = invoiceItems; jobs.value = jobItems; orders.value = orderItems;
		const target = String(route.query.planId || "").trim();
		if (target) { const item = planItems.find((value) => value.id === target); if (item) openPlanDetail(item); }
	} catch (cause) { error.value = toErrorMessage(cause); } finally { loading.value = false; }
}

function changeTab() { statusFilter.value = ""; void refresh(); }
function openPlanCreate() { Object.assign(planForm, { salesOrderId: ownedOrders.value[0]?.id || "", amount: 0, dueAt: new Date(Date.now() + 7 * 86400000), note: "" }); planFile.value = null; planOpen.value = true; }
function openInvoiceCreate() { Object.assign(invoiceForm, { number: `INV-${Date.now()}`, salesOrderId: ownedOrders.value[0]?.id || "", amount: 0, issuedAt: new Date(), note: "" }); invoiceFile.value = null; invoiceOpen.value = true; }
function openReceipt(item: PaymentPlan) { selectedPlan.value = item; Object.assign(receiptForm, { amount: (item.amountCents - item.paidAmountCents) / 100, paidAt: new Date(), reference: "" }); receiptFile.value = null; receiptOpen.value = true; }
function onPlanFile(file: UploadFile) { planFile.value = file.raw || null; }
function onReceiptFile(file: UploadFile) { receiptFile.value = file.raw || null; }
function onInvoiceFile(file: UploadFile) { invoiceFile.value = file.raw || null; }

async function financialAttachment(file: File | null, resource: string): Promise<FinancialAttachment[]> {
	if (!file) return [];
	const uploaded = await uploadFile({ file, name: file.name, visibility: "private", ownerType: "user", ownerId: actorId.value, sourceModule: "pharma_oa", sourcePluginId: "pharma_oa", metadata: { resource } });
	return [{ fileId: uploaded.id, fileName: uploaded.name, size: uploaded.size }];
}

async function savePlan() {
	if (!planForm.salesOrderId || planForm.amount <= 0 || !planForm.dueAt || !canCreatePlan.value) return;
	saving.value = "plan"; error.value = "";
	try {
		const item = await createPaymentPlan({ salesOrderId: planForm.salesOrderId, amountCents: cents(planForm.amount), dueAt: planForm.dueAt.toISOString(), note: planForm.note.trim(), attachments: await financialAttachment(planFile.value, "payment_plan") });
		plans.value = [item, ...plans.value]; planOpen.value = false; ElMessage.success(t("pharma.paymentInvoice.paymentPlanCreated"));
	} catch (cause) { error.value = toErrorMessage(cause); } finally { saving.value = ""; }
}

async function saveReceipt() {
	if (!selectedPlan.value || receiptForm.amount <= 0 || !receiptForm.paidAt || !canReceive.value) return;
	saving.value = "receipt"; error.value = "";
	try {
		const item = await recordPayment(selectedPlan.value.id, { amountCents: cents(receiptForm.amount), paidAt: receiptForm.paidAt.toISOString(), reference: receiptForm.reference.trim(), attachments: await financialAttachment(receiptFile.value, "payment_receipt") });
		replacePlan(item); receiptOpen.value = false; ElMessage.success(t("pharma.paymentInvoice.paymentRecorded"));
	} catch (cause) { error.value = toErrorMessage(cause); } finally { saving.value = ""; }
}

async function saveInvoice() {
	if (!invoiceForm.number.trim() || !invoiceForm.salesOrderId || invoiceForm.amount <= 0 || !invoiceForm.issuedAt || !canCreateInvoice.value) return;
	saving.value = "invoice"; error.value = "";
	try {
		const item = await createInvoiceRecord({ number: invoiceForm.number.trim(), salesOrderId: invoiceForm.salesOrderId, amountCents: cents(invoiceForm.amount), issuedAt: invoiceForm.issuedAt.toISOString(), note: invoiceForm.note.trim(), attachments: await financialAttachment(invoiceFile.value, "invoice_record") });
		invoices.value = [item, ...invoices.value]; invoiceOpen.value = false; ElMessage.success(t("pharma.paymentInvoice.invoiceRecorded"));
	} catch (cause) { error.value = toErrorMessage(cause); } finally { saving.value = ""; }
}

async function voidInvoice(item: InvoiceRecord) {
	if (!canVoidInvoice.value || item.status !== "issued") return;
	let result: { value: string };
	try { result = await ElMessageBox.prompt(t("pharma.paymentInvoice.voidReason"), `${t("pharma.paymentInvoice.voidInvoice")}: ${item.number}`, { type: "warning", confirmButtonText: t("pharma.paymentInvoice.voidInvoice"), cancelButtonText: t("common.cancel"), inputValidator: (value) => Boolean(value.trim()) || t("pharma.common.reasonRequired") }); } catch { return; }
	actionId.value = item.id; error.value = "";
	try { replaceInvoice(await voidInvoiceRecord(item.id, result.value)); ElMessage.success(t("pharma.paymentInvoice.invoiceVoided")); } catch (cause) { error.value = toErrorMessage(cause); } finally { actionId.value = ""; }
}

async function runScan() {
	if (!scanForm.recipientId.trim() || !canRun.value) return;
	saving.value = "scan"; error.value = "";
	try { const job = await runPaymentReminderScan(scanForm.recipientId.trim()); jobs.value = [job, ...jobs.value]; await refresh(); scanOpen.value = false; activeTab.value = "jobs"; ElMessage.success(t("pharma.paymentInvoice.reminderScanCompleted")); } catch (cause) { error.value = toErrorMessage(cause); await refresh(); } finally { saving.value = ""; }
}

async function retryJob(item: PaymentReminderJob) {
	if (!canRun.value || item.status !== "failed") return;
	actionId.value = item.id; error.value = "";
	try { replaceJob(await retryPaymentReminderScan(item.id)); ElMessage.success(t("pharma.paymentInvoice.reminderJobRetried")); } catch (cause) { error.value = toErrorMessage(cause); } finally { actionId.value = ""; }
}

function replacePlan(item: PaymentPlan) { plans.value = plans.value.map((value) => value.id === item.id ? item : value); selectedPlan.value = item; }
function replaceInvoice(item: InvoiceRecord) { invoices.value = invoices.value.map((value) => value.id === item.id ? item : value); selectedInvoice.value = item; }
function replaceJob(item: PaymentReminderJob) { jobs.value = jobs.value.map((value) => value.id === item.id ? item : value); selectedJob.value = item; }
function openPlanDetail(item: PaymentPlan) { selectedPlan.value = item; selectedInvoice.value = null; selectedJob.value = null; detailOpen.value = true; }
function openInvoiceDetail(item: InvoiceRecord) { selectedInvoice.value = item; selectedPlan.value = null; selectedJob.value = null; detailOpen.value = true; }
function openJobDetail(item: PaymentReminderJob) { selectedJob.value = item; selectedPlan.value = null; selectedInvoice.value = null; detailOpen.value = true; }
function cents(value: number) { return Math.round(value * 100); }
function money(value: number) { return new Intl.NumberFormat(undefined, { style: "currency", currency: "CNY" }).format(value / 100); }
function date(value: string) { return new Date(value).toLocaleString([], { dateStyle: "medium", timeStyle: "short" }); }
function statusType(value: string): "success" | "warning" | "danger" | "info" { return value === "paid" || value === "succeeded" || value === "issued" ? "success" : value === "overdue" || value === "failed" || value === "voided" ? "danger" : value === "partial" || value === "running" ? "warning" : "info"; }
</script>

<template>
	<PageShell :title="t('pharma.paymentInvoice.paymentsInvoices')" :loading="loading" :error="error" :forbidden="!canRead" :forbidden-title="t('pharma.paymentInvoice.noFinanceAccess')" :forbidden-description="t('pharma.paymentInvoice.thisPageRequiresPharmaOaPaymentInvoiceReadPermission')">
		<template #actions>
			<el-input v-model="keyword" clearable :placeholder="t('pharma.paymentInvoice.searchOrderOrInvoice')" class="filter" @keyup.enter="refresh" />
			<el-select v-if="activeTab !== 'jobs'" v-model="statusFilter" clearable :placeholder="t('pharma.paymentInvoice.allStatuses')" class="filter" @change="refresh">
				<template v-if="activeTab === 'plans'"><el-option :label="t('pharma.paymentInvoice.pending')" value="pending" /><el-option :label="t('pharma.paymentInvoice.partial')" value="partial" /><el-option :label="t('pharma.paymentInvoice.overdue')" value="overdue" /><el-option :label="t('pharma.paymentInvoice.paid')" value="paid" /></template>
				<template v-else><el-option :label="t('pharma.paymentInvoice.issued')" value="issued" /><el-option :label="t('pharma.paymentInvoice.voided')" value="voided" /></template>
			</el-select>
			<el-tooltip :content="t('pharma.paymentInvoice.refresh')"><el-button circle :icon="RefreshCw" :loading="loading" :aria-label="t('pharma.paymentInvoice.refreshFinanceRecords')" @click="refresh" /></el-tooltip>
			<el-button v-if="activeTab === 'plans'" type="primary" :icon="Banknote" :disabled="!canCreatePlan || ownedOrders.length === 0" @click="openPlanCreate">{{ t("pharma.paymentInvoice.newPlan") }}</el-button>
			<el-button v-if="activeTab === 'invoices'" type="primary" :icon="FilePlus2" :disabled="!canCreateInvoice || ownedOrders.length === 0" @click="openInvoiceCreate">{{ t("pharma.paymentInvoice.newInvoice") }}</el-button>
			<el-button v-if="activeTab === 'jobs'" type="primary" :icon="BellRing" :disabled="!canRun" @click="scanOpen = true">{{ t("pharma.paymentInvoice.runScan") }}</el-button>
		</template>
		<section class="summary"><div><span>{{ t("pharma.paymentInvoice.outstanding") }}</span><strong>{{ money(outstandingCents) }}</strong></div><div><span>{{ t("pharma.paymentInvoice.overdue") }}</span><strong>{{ overdueCount }}</strong></div><div><span>{{ t("pharma.paymentInvoice.issued") }}</span><strong>{{ money(issuedCents) }}</strong></div></section>
		<el-tabs v-model="activeTab" @tab-change="changeTab">
			<el-tab-pane :label="t('pharma.paymentInvoice.paymentPlans')" name="plans">
				<DataTable :rows="planRows" :columns="planColumns" row-key="id" :loading="loading" :error="error" :empty-text="t('pharma.paymentInvoice.noPaymentPlansMatchTheCurrentFilters')">
					<template #cell-status="{ row }"><el-tag :type="statusType(String(row.status))">{{ valueLabel(row.status) }}</el-tag></template>
					<template #actions="{ row }"><el-tooltip :content="t('pharma.paymentInvoice.view')"><el-button circle :icon="Eye" :aria-label="t('pharma.paymentInvoice.viewPaymentPlan')" @click="openPlanDetail(row as PaymentPlan)" /></el-tooltip><el-tooltip v-if="row.status !== 'paid'" :content="t('pharma.paymentInvoice.recordPayment')"><el-button circle type="success" :icon="Banknote" :disabled="!canReceive" :aria-label="t('pharma.paymentInvoice.recordPayment')" @click="openReceipt(row as PaymentPlan)" /></el-tooltip></template>
				</DataTable>
			</el-tab-pane>
			<el-tab-pane :label="t('pharma.paymentInvoice.invoices')" name="invoices">
				<DataTable :rows="invoiceRows" :columns="invoiceColumns" row-key="id" :loading="loading" :error="error" :empty-text="t('pharma.paymentInvoice.noInvoicesMatchTheCurrentFilters')">
					<template #cell-status="{ row }"><el-tag :type="statusType(String(row.status))">{{ valueLabel(row.status) }}</el-tag></template>
					<template #actions="{ row }"><el-tooltip :content="t('pharma.paymentInvoice.view')"><el-button circle :icon="Eye" :aria-label="t('pharma.paymentInvoice.viewInvoice')" @click="openInvoiceDetail(row as InvoiceRecord)" /></el-tooltip><el-tooltip v-if="row.status === 'issued'" :content="t('pharma.paymentInvoice.void')"><el-button circle type="danger" :icon="Ban" :loading="actionId === row.id" :disabled="!canVoidInvoice" :aria-label="t('pharma.paymentInvoice.voidInvoice')" @click="voidInvoice(row as InvoiceRecord)" /></el-tooltip></template>
				</DataTable>
			</el-tab-pane>
			<el-tab-pane :label="t('pharma.paymentInvoice.reminderJobs')" name="jobs">
				<DataTable :rows="jobRows" :columns="jobColumns" row-key="id" :loading="loading" :error="error" :empty-text="t('pharma.paymentInvoice.noReminderJobsHaveRun')">
					<template #cell-status="{ row }"><el-tag :type="statusType(String(row.status))">{{ valueLabel(row.status) }}</el-tag></template>
					<template #actions="{ row }"><el-tooltip :content="t('pharma.paymentInvoice.logs')"><el-button circle :icon="Eye" :aria-label="t('pharma.paymentInvoice.viewReminderJob')" @click="openJobDetail(row as PaymentReminderJob)" /></el-tooltip><el-tooltip v-if="row.status === 'failed'" :content="t('pharma.paymentInvoice.retry')"><el-button circle type="warning" :icon="RotateCcw" :loading="actionId === row.id" :disabled="!canRun" :aria-label="t('pharma.paymentInvoice.retryReminderJob')" @click="retryJob(row as PaymentReminderJob)" /></el-tooltip></template>
				</DataTable>
			</el-tab-pane>
		</el-tabs>

		<DetailDrawer v-model="planOpen" :title="t('pharma.paymentInvoice.newPaymentPlan')" size="48%"><el-form label-position="top"><el-form-item :label="t('pharma.paymentInvoice.salesOrder')" required><el-select v-model="planForm.salesOrderId" filterable><el-option v-for="order in ownedOrders" :key="order.id" :label="`${order.number} · ${money(Math.round(order.totalAmount * 100))}`" :value="order.id" /></el-select></el-form-item><div class="grid"><el-form-item :label="t('pharma.paymentInvoice.amount')" required><el-input-number v-model="planForm.amount" :min="0.01" :precision="2" :step="100" /></el-form-item><el-form-item :label="t('pharma.paymentInvoice.due')" required><el-date-picker v-model="planForm.dueAt" type="datetime" /></el-form-item></div><el-form-item :label="t('pharma.paymentInvoice.note')"><el-input v-model="planForm.note" type="textarea" :rows="3" maxlength="500" /></el-form-item><el-form-item :label="t('pharma.paymentInvoice.attachment')"><el-upload :auto-upload="false" :limit="1" :on-change="onPlanFile" accept=".pdf,.doc,.docx,.png,.jpg,.jpeg"><el-button :icon="Upload">{{ t("pharma.paymentInvoice.selectFile") }}</el-button></el-upload></el-form-item></el-form><template #footer><el-button :disabled="saving === 'plan'" @click="planOpen = false">{{ t("pharma.paymentInvoice.close") }}</el-button><el-button type="primary" :loading="saving === 'plan'" :disabled="!planForm.salesOrderId || planForm.amount <= 0 || !planForm.dueAt" @click="savePlan">{{ t("pharma.paymentInvoice.savePlan") }}</el-button></template></DetailDrawer>
		<DetailDrawer v-model="receiptOpen" :title="selectedPlan ? `${t('pharma.paymentInvoice.recordPayment')} / ${selectedPlan.salesOrderNumber}` : t('pharma.paymentInvoice.recordPayment')" size="48%"><el-form label-position="top"><div class="grid"><el-form-item :label="t('pharma.paymentInvoice.amount')" required><el-input-number v-model="receiptForm.amount" :min="0.01" :max="selectedPlan ? (selectedPlan.amountCents - selectedPlan.paidAmountCents) / 100 : undefined" :precision="2" /></el-form-item><el-form-item :label="t('pharma.paymentInvoice.paidAt')" required><el-date-picker v-model="receiptForm.paidAt" type="datetime" /></el-form-item></div><el-form-item :label="t('pharma.paymentInvoice.reference')"><el-input v-model="receiptForm.reference" maxlength="120" /></el-form-item><el-form-item :label="t('pharma.paymentInvoice.receipt')"><el-upload :auto-upload="false" :limit="1" :on-change="onReceiptFile" accept=".pdf,.png,.jpg,.jpeg"><el-button :icon="Upload">{{ t("pharma.paymentInvoice.selectFile") }}</el-button></el-upload></el-form-item></el-form><template #footer><el-button :disabled="saving === 'receipt'" @click="receiptOpen = false">{{ t("pharma.paymentInvoice.close") }}</el-button><el-button type="success" :loading="saving === 'receipt'" :disabled="receiptForm.amount <= 0 || !receiptForm.paidAt" @click="saveReceipt">{{ t("pharma.paymentInvoice.recordPayment") }}</el-button></template></DetailDrawer>
		<DetailDrawer v-model="invoiceOpen" :title="t('pharma.paymentInvoice.newInvoice')" size="48%"><el-form label-position="top"><div class="grid"><el-form-item :label="t('pharma.paymentInvoice.invoiceNumber')" required><el-input v-model="invoiceForm.number" maxlength="80" /></el-form-item><el-form-item :label="t('pharma.paymentInvoice.salesOrder')" required><el-select v-model="invoiceForm.salesOrderId" filterable><el-option v-for="order in ownedOrders" :key="order.id" :label="order.number" :value="order.id" /></el-select></el-form-item><el-form-item :label="t('pharma.paymentInvoice.amount')" required><el-input-number v-model="invoiceForm.amount" :min="0.01" :precision="2" /></el-form-item><el-form-item :label="t('pharma.paymentInvoice.issuedAt')" required><el-date-picker v-model="invoiceForm.issuedAt" type="datetime" /></el-form-item></div><el-form-item :label="t('pharma.paymentInvoice.note')"><el-input v-model="invoiceForm.note" type="textarea" :rows="3" maxlength="500" /></el-form-item><el-form-item :label="t('pharma.paymentInvoice.invoiceFile')"><el-upload :auto-upload="false" :limit="1" :on-change="onInvoiceFile" accept=".pdf,.png,.jpg,.jpeg"><el-button :icon="Upload">{{ t("pharma.paymentInvoice.selectFile") }}</el-button></el-upload></el-form-item></el-form><template #footer><el-button :disabled="saving === 'invoice'" @click="invoiceOpen = false">{{ t("pharma.paymentInvoice.close") }}</el-button><el-button type="primary" :loading="saving === 'invoice'" :disabled="!invoiceForm.number.trim() || !invoiceForm.salesOrderId || invoiceForm.amount <= 0 || !invoiceForm.issuedAt" @click="saveInvoice">{{ t("pharma.paymentInvoice.saveInvoice") }}</el-button></template></DetailDrawer>
		<DetailDrawer v-model="scanOpen" :title="t('pharma.paymentInvoice.runOverdueScan')" size="42%"><el-form label-position="top"><el-form-item :label="t('pharma.paymentInvoice.reminderRecipient')" required><el-input v-model="scanForm.recipientId" maxlength="120" /></el-form-item></el-form><template #footer><el-button :disabled="saving === 'scan'" @click="scanOpen = false">{{ t("pharma.paymentInvoice.close") }}</el-button><el-button type="primary" :icon="BellRing" :loading="saving === 'scan'" :disabled="!scanForm.recipientId.trim()" @click="runScan">{{ t("pharma.paymentInvoice.runScan") }}</el-button></template></DetailDrawer>

		<DetailDrawer v-model="detailOpen" :title="t('pharma.paymentInvoice.financeRecord')" size="52%">
			<template v-if="selectedPlan"><div class="detail-head"><strong>{{ selectedPlan.salesOrderNumber }}</strong><el-tag :type="statusType(selectedPlan.status)">{{ valueLabel(selectedPlan.status) }}</el-tag></div><dl class="details"><div><dt>{{ t("pharma.paymentInvoice.plan") }}</dt><dd>{{ money(selectedPlan.amountCents) }}</dd></div><div><dt>{{ t("pharma.paymentInvoice.received") }}</dt><dd>{{ money(selectedPlan.paidAmountCents) }}</dd></div><div><dt>{{ t("pharma.paymentInvoice.due") }}</dt><dd>{{ date(selectedPlan.dueAt) }}</dd></div><div><dt>{{ t("pharma.paymentInvoice.owner") }}</dt><dd>{{ selectedPlan.ownerId }}</dd></div><div class="wide"><dt>{{ t("pharma.paymentInvoice.note") }}</dt><dd>{{ selectedPlan.note || t('pharma.paymentInvoice.none') }}</dd></div></dl><h3>{{ t("pharma.paymentInvoice.receipts") }}</h3><el-timeline v-if="selectedPlan.receipts.length"><el-timeline-item v-for="item in selectedPlan.receipts" :key="item.id" :timestamp="date(item.paidAt)"><strong>{{ money(item.amountCents) }}</strong><p>{{ item.reference || t('pharma.paymentInvoice.noReference') }} · {{ item.recordedBy }}</p></el-timeline-item></el-timeline><el-empty v-else :description="t('pharma.paymentInvoice.noPaymentsRecorded')" :image-size="72" /></template>
			<template v-else-if="selectedInvoice"><div class="detail-head"><strong>{{ selectedInvoice.number }}</strong><el-tag :type="statusType(selectedInvoice.status)">{{ valueLabel(selectedInvoice.status) }}</el-tag></div><dl class="details"><div><dt>{{ t("pharma.paymentInvoice.salesOrder") }}</dt><dd>{{ selectedInvoice.salesOrderNumber }}</dd></div><div><dt>{{ t("pharma.paymentInvoice.amount") }}</dt><dd>{{ money(selectedInvoice.amountCents) }}</dd></div><div><dt>{{ t("pharma.paymentInvoice.issued") }}</dt><dd>{{ date(selectedInvoice.issuedAt) }}</dd></div><div><dt>{{ t("pharma.paymentInvoice.createdBy") }}</dt><dd>{{ selectedInvoice.createdBy }}</dd></div><div v-if="selectedInvoice.voidReason" class="wide"><dt>{{ t("pharma.paymentInvoice.voidReason") }}</dt><dd>{{ selectedInvoice.voidReason }}</dd></div></dl><section v-if="selectedInvoice.attachments.length" class="files"><div v-for="file in selectedInvoice.attachments" :key="file.fileId"><FileText :size="18" /><span>{{ file.fileName }}</span></div></section></template>
			<template v-else-if="selectedJob"><div class="detail-head"><strong>{{ selectedJob.id }}</strong><el-tag :type="statusType(selectedJob.status)">{{ valueLabel(selectedJob.status) }}</el-tag></div><dl class="details"><div><dt>{{ t("pharma.paymentInvoice.recipient") }}</dt><dd>{{ selectedJob.recipientId }}</dd></div><div><dt>{{ t("pharma.paymentInvoice.retries") }}</dt><dd>{{ selectedJob.retryCount }}</dd></div><div><dt>{{ t("pharma.paymentInvoice.matched") }}</dt><dd>{{ selectedJob.matchedCount }}</dd></div><div><dt>{{ t("pharma.paymentInvoice.newReminders") }}</dt><dd>{{ selectedJob.createdCount }}</dd></div><div v-if="selectedJob.error" class="wide"><dt>{{ t("pharma.paymentInvoice.error") }}</dt><dd>{{ selectedJob.error }}</dd></div></dl><el-timeline><el-timeline-item v-for="(log, index) in selectedJob.logs" :key="index" :timestamp="date(log.createdAt)" :type="log.level === 'error' ? 'danger' : 'primary'">{{ log.message }}</el-timeline-item></el-timeline></template>
			<template #footer><el-button @click="detailOpen = false">{{ t("pharma.paymentInvoice.close") }}</el-button></template>
		</DetailDrawer>
	</PageShell>
</template>

<style scoped>
.filter{width:190px}.summary{display:flex;gap:42px;padding:4px 0 14px;border-bottom:1px solid var(--el-border-color-lighter)}.summary div{display:grid;gap:3px}.summary span,.details dt{color:var(--el-text-color-secondary);font-size:12px}.summary strong{font-size:21px}.grid,.details{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.detail-head{display:flex;align-items:center;justify-content:space-between;gap:12px;margin-bottom:18px}.details{margin:0}.details dd{margin:4px 0 0;overflow-wrap:anywhere}.details .wide{grid-column:1/-1}.files{margin-top:20px;border-top:1px solid var(--el-border-color-lighter)}.files div{display:flex;align-items:center;gap:8px;padding:12px 0}.el-timeline p{margin:4px 0;color:var(--el-text-color-secondary)}@media(max-width:760px){.filter{width:100%}.summary{gap:16px;justify-content:space-between}.summary strong{font-size:17px}.grid,.details{grid-template-columns:1fr}.details .wide{grid-column:auto}}
</style>
