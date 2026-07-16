<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ArrowRight, Eye, Pencil, Plus, RefreshCw, TrendingUp, XCircle } from "lucide-vue-next";
import { ElMessage, ElMessageBox } from "element-plus";
import { DataTable, DetailDrawer, PageShell, type DataTableColumn } from "../../components/Common";
import { useButtonAccess } from "../../permissions/button";
import {
	advanceSalesOpportunity,
	createSalesOpportunity,
	getSalesOpportunityStatistics,
	listCustomers,
	listProducts,
	listSalesOpportunities,
	updateSalesOpportunity,
	type PharmaCustomer,
	type PharmaProduct,
	type SalesOpportunity,
	type SalesOpportunityStage,
	type SalesOpportunityStatistics
} from "../../pharma-oa/api";
import { useUserStore } from "../../stores/user";
import { toErrorMessage } from "../../utils/common";

type OpportunityRow = Record<string, unknown> & SalesOpportunity & { customerText: string; productText: string; amountText: string; closeText: string };
const stages: SalesOpportunityStage[] = ["lead", "qualified", "proposal", "negotiation", "won", "lost"];
const forwardStages: SalesOpportunityStage[] = ["lead", "qualified", "proposal", "negotiation", "won"];
const access = useButtonAccess();
const userStore = useUserStore();
const loading = ref(false), saving = ref(false), advancingId = ref("");
const error = ref(""), keyword = ref(""), stageFilter = ref<SalesOpportunityStage | "">("");
const editOpen = ref(false), detailOpen = ref(false), editingId = ref("");
const items = ref<SalesOpportunity[]>([]), customers = ref<PharmaCustomer[]>([]), products = ref<PharmaProduct[]>([]), selected = ref<SalesOpportunity | null>(null);
const statistics = ref<SalesOpportunityStatistics>({ totalCount: 0, openCount: 0, wonCount: 0, lostCount: 0, expectedAmountCents: 0, stages: [] });
const actorId = computed(() => userStore.profile?.id?.trim() || "system");
const canRead = computed(() => access.can("pharma_oa.sales_opportunity.read"));
const canCreate = computed(() => access.can("pharma_oa.sales_opportunity.create"));
const canUpdate = computed(() => access.can("pharma_oa.sales_opportunity.update"));
const canAdvance = computed(() => access.can("pharma_oa.sales_opportunity.advance"));
const form = reactive<{ title: string; customerId: string; productIds: string[]; expectedAmount: number; estimatedCloseDate: Date | null }>({ title: "", customerId: "", productIds: [], expectedAmount: 0, estimatedCloseDate: null });
const rows = computed<OpportunityRow[]>(() => items.value.map((item) => ({ ...item, customerText: `${item.customerCode} · ${item.customerName}`, productText: item.products.map((product) => product.productName).join(", "), amountText: formatMoney(item.expectedAmountCents), closeText: formatDate(item.estimatedCloseDate) })));
const columns: DataTableColumn[] = [
	{ key: "title", label: "Opportunity", minWidth: 190 },
	{ key: "customerText", label: "Customer", minWidth: 210 },
	{ key: "productText", label: "Products", minWidth: 190 },
	{ key: "amountText", label: "Expected", width: 130 },
	{ key: "closeText", label: "Close date", width: 130 },
	{ key: "stage", label: "Stage", width: 125 }
];

onMounted(() => void refresh());

async function refresh() {
	if (!canRead.value) return;
	loading.value = true;
	error.value = "";
	try {
		const [opportunities, stats, customerItems, productItems] = await Promise.all([
			listSalesOpportunities({ keyword: keyword.value, stage: stageFilter.value }), getSalesOpportunityStatistics(),
			listCustomers({ status: "active", scope: { ownerId: actorId.value } }), listProducts({ status: "active" })
		]);
		items.value = opportunities;
		statistics.value = stats;
		customers.value = customerItems;
		products.value = productItems;
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		loading.value = false;
	}
}

function openCreate() {
	editingId.value = "";
	Object.assign(form, { title: "", customerId: customers.value[0]?.id || "", productIds: [], expectedAmount: 0, estimatedCloseDate: new Date(Date.now() + 30 * 86400000) });
	editOpen.value = true;
}

function openEdit(item: SalesOpportunity) {
	editingId.value = item.id;
	Object.assign(form, { title: item.title, customerId: item.customerId, productIds: item.products.map((product) => product.productId), expectedAmount: item.expectedAmountCents / 100, estimatedCloseDate: new Date(item.estimatedCloseDate) });
	editOpen.value = true;
}

async function save() {
	if (!form.title.trim() || !form.customerId || form.productIds.length === 0 || form.expectedAmount <= 0 || !form.estimatedCloseDate) return;
	saving.value = true;
	error.value = "";
	try {
		const body = { title: form.title.trim(), customerId: form.customerId, productIds: form.productIds, expectedAmountCents: Math.round(form.expectedAmount * 100), estimatedCloseDate: form.estimatedCloseDate.toISOString() };
		const item = editingId.value ? await updateSalesOpportunity(editingId.value, body) : await createSalesOpportunity(body);
		replaceItem(item, !editingId.value);
		editOpen.value = false;
		await refreshStatistics();
		ElMessage.success(editingId.value ? "Opportunity updated" : "Opportunity created");
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		saving.value = false;
	}
}

async function advance(item: SalesOpportunity) {
	const target = nextStage(item.stage);
	if (!target || !canAdvance.value) return;
	try {
		await ElMessageBox.confirm(`Advance ${item.title} to ${stageLabel(target)}?`, "Advance opportunity", { type: "warning", confirmButtonText: "Advance" });
	} catch {
		return;
	}
	await applyStage(item, target, "Stage advanced");
}

async function lose(item: SalesOpportunity) {
	if (!canAdvance.value || isTerminal(item.stage)) return;
	let result;
	try {
		result = await ElMessageBox.prompt("Record why this opportunity was lost.", `Close ${item.title} as lost`, { type: "warning", confirmButtonText: "Close as lost", inputValidator: (value) => Boolean(value.trim()) || "Reason is required" });
	} catch {
		return;
	}
	await applyStage(item, "lost", result.value.trim());
}

async function applyStage(item: SalesOpportunity, stage: SalesOpportunityStage, note: string) {
	advancingId.value = item.id;
	error.value = "";
	try {
		replaceItem(await advanceSalesOpportunity(item.id, stage, note));
		await refreshStatistics();
		ElMessage.success(stage === "lost" ? "Opportunity closed as lost" : `Advanced to ${stageLabel(stage)}`);
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		advancingId.value = "";
	}
}

async function refreshStatistics() { statistics.value = await getSalesOpportunityStatistics(); }
function replaceItem(item: SalesOpportunity, prepend = false) { items.value = prepend ? [item, ...items.value] : items.value.map((current) => current.id === item.id ? item : current); if (selected.value?.id === item.id) selected.value = item; }
function openDetail(item: SalesOpportunity) { selected.value = item; detailOpen.value = true; }
function nextStage(stage: SalesOpportunityStage): SalesOpportunityStage | null { const index = forwardStages.indexOf(stage); return index >= 0 && index < forwardStages.length - 1 ? forwardStages[index + 1] : null; }
function isTerminal(stage: SalesOpportunityStage) { return stage === "won" || stage === "lost"; }
function stageLabel(stage?: SalesOpportunityStage | null) { return stage ? stage.replace(/^./, (value) => value.toUpperCase()) : "Unknown"; }
function stageType(stage: SalesOpportunityStage): "success" | "danger" | "warning" | "info" { return stage === "won" ? "success" : stage === "lost" ? "danger" : stage === "negotiation" ? "warning" : "info"; }
function formatMoney(cents: number) { return new Intl.NumberFormat(undefined, { style: "currency", currency: "USD" }).format(cents / 100); }
function formatDate(value: string) { return new Date(value).toLocaleDateString([], { dateStyle: "medium" }); }
function formatDateTime(value: string) { return new Date(value).toLocaleString([], { dateStyle: "medium", timeStyle: "short" }); }
</script>

<template>
	<PageShell title="Sales Opportunities" :loading="loading" :error="error" :forbidden="!canRead" forbidden-title="No opportunity access" forbidden-description="This page requires pharma_oa.sales_opportunity.read permission.">
		<template #actions>
			<el-input v-model="keyword" clearable placeholder="Search opportunity or customer" class="filter-control" @keyup.enter="refresh" />
			<el-select v-model="stageFilter" clearable placeholder="All stages" class="filter-control" @change="refresh"><el-option v-for="stage in stages" :key="stage" :label="stageLabel(stage)" :value="stage" /></el-select>
			<el-tooltip content="Refresh"><el-button circle :icon="RefreshCw" aria-label="Refresh opportunities" :loading="loading" @click="refresh" /></el-tooltip>
			<el-button type="primary" :icon="Plus" :disabled="!canCreate || customers.length === 0 || products.length === 0" @click="openCreate">New opportunity</el-button>
		</template>

		<section class="funnel-band" aria-label="Sales funnel statistics">
			<div class="funnel-total"><span>Pipeline</span><strong>{{ formatMoney(statistics.expectedAmountCents) }}</strong><small>{{ statistics.openCount }} open · {{ statistics.wonCount }} won</small></div>
			<div v-for="bucket in statistics.stages" :key="bucket.stage" class="funnel-stage"><span>{{ stageLabel(bucket.stage) }}</span><strong>{{ bucket.count }}</strong><small>{{ formatMoney(bucket.expectedAmountCents) }}</small></div>
		</section>

		<DataTable :rows="rows" :columns="columns" row-key="id" :loading="loading" :error="error" empty-text="No sales opportunities match the current filters.">
			<template #cell-title="{ row }"><strong>{{ row.title }}</strong></template>
			<template #cell-stage="{ row }"><el-tag :type="stageType(row.stage as SalesOpportunityStage)">{{ stageLabel(row.stage as SalesOpportunityStage) }}</el-tag></template>
			<template #actions="{ row }">
				<el-tooltip content="View details"><el-button circle :icon="Eye" aria-label="View opportunity" @click="openDetail(row as SalesOpportunity)" /></el-tooltip>
				<el-tooltip v-if="!isTerminal(row.stage as SalesOpportunityStage)" content="Edit"><el-button circle :icon="Pencil" aria-label="Edit opportunity" :disabled="!canUpdate" @click="openEdit(row as SalesOpportunity)" /></el-tooltip>
				<el-tooltip v-if="nextStage(row.stage as SalesOpportunityStage)" :content="`Advance to ${stageLabel(nextStage(row.stage as SalesOpportunityStage)!)}`"><el-button circle type="primary" :icon="ArrowRight" aria-label="Advance opportunity" :loading="advancingId === row.id" :disabled="!canAdvance" @click="advance(row as SalesOpportunity)" /></el-tooltip>
				<el-tooltip v-if="!isTerminal(row.stage as SalesOpportunityStage)" content="Close as lost"><el-button circle type="danger" :icon="XCircle" aria-label="Close opportunity as lost" :loading="advancingId === row.id" :disabled="!canAdvance" @click="lose(row as SalesOpportunity)" /></el-tooltip>
			</template>
		</DataTable>

		<DetailDrawer v-model="editOpen" :title="editingId ? 'Edit opportunity' : 'New sales opportunity'" size="52%">
			<el-form label-position="top"><el-form-item label="Title" required><el-input v-model="form.title" maxlength="120" /></el-form-item><div class="form-grid"><el-form-item label="Customer" required><el-select v-model="form.customerId" filterable :disabled="Boolean(editingId)"><el-option v-for="customer in customers" :key="customer.id" :label="`${customer.code} · ${customer.name}`" :value="customer.id" /></el-select></el-form-item><el-form-item label="Expected amount" required><el-input-number v-model="form.expectedAmount" :min="0" :precision="2" :step="100" controls-position="right" /></el-form-item></div><div class="form-grid"><el-form-item label="Products" required><el-select v-model="form.productIds" multiple filterable collapse-tags><el-option v-for="product in products" :key="product.id" :label="`${product.code} · ${product.name}`" :value="product.id" /></el-select></el-form-item><el-form-item label="Estimated close" required><el-date-picker v-model="form.estimatedCloseDate" type="date" /></el-form-item></div></el-form>
			<template #footer><el-button :disabled="saving" @click="editOpen = false">Close</el-button><el-button type="primary" :loading="saving" :disabled="!form.title.trim() || !form.customerId || form.productIds.length === 0 || form.expectedAmount <= 0 || !form.estimatedCloseDate" @click="save">Save opportunity</el-button></template>
		</DetailDrawer>

		<DetailDrawer v-model="detailOpen" :title="selected?.title || 'Sales opportunity'" size="50%"><template v-if="selected"><div class="detail-status"><el-tag :type="stageType(selected.stage)">{{ stageLabel(selected.stage) }}</el-tag><strong>{{ formatMoney(selected.expectedAmountCents) }}</strong></div><dl class="detail-grid"><div><dt>Customer</dt><dd>{{ selected.customerCode }} · {{ selected.customerName }}</dd></div><div><dt>Owner</dt><dd>{{ selected.ownerId }}</dd></div><div><dt>Products</dt><dd>{{ selected.products.map((product) => product.productName).join(', ') }}</dd></div><div><dt>Estimated close</dt><dd>{{ formatDate(selected.estimatedCloseDate) }}</dd></div><div v-if="selected.lostReason" class="wide"><dt>Lost reason</dt><dd>{{ selected.lostReason }}</dd></div></dl><section class="timeline"><h3><TrendingUp :size="18" /> Stage history</h3><el-timeline><el-timeline-item v-for="change in [...selected.stageHistory].reverse()" :key="`${change.to}-${change.changedAt}`" :timestamp="formatDateTime(change.changedAt)" placement="top"><strong>{{ stageLabel(change.to) }}</strong><p>{{ change.note || `Changed by ${change.changedBy}` }}</p></el-timeline-item></el-timeline></section></template><template #footer><el-button @click="detailOpen = false">Close</el-button><el-button v-if="selected && nextStage(selected.stage)" type="primary" :icon="ArrowRight" :disabled="!canAdvance" @click="advance(selected)">Advance</el-button></template></DetailDrawer>
	</PageShell>
</template>

<style scoped>
.filter-control{width:210px}.funnel-band{display:grid;grid-template-columns:minmax(190px,1.5fr) repeat(6,minmax(90px,1fr));gap:0;margin-bottom:16px;border-block:1px solid var(--el-border-color-lighter)}.funnel-band>div{display:grid;gap:3px;padding:13px 16px;border-right:1px solid var(--el-border-color-lighter);min-width:0}.funnel-band>div:last-child{border-right:0}.funnel-band span,.funnel-band small{color:var(--el-text-color-secondary);font-size:12px}.funnel-band strong{font-size:19px;overflow-wrap:anywhere}.funnel-total strong{color:var(--el-color-primary)}.form-grid,.detail-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.detail-status{display:flex;align-items:center;justify-content:space-between;margin-bottom:18px}.detail-grid{margin:0}.detail-grid .wide{grid-column:1/-1}.detail-grid dt{color:var(--el-text-color-secondary);font-size:12px}.detail-grid dd{margin:5px 0 0;overflow-wrap:anywhere}.timeline{margin-top:24px;border-top:1px solid var(--el-border-color-lighter)}.timeline h3{display:flex;align-items:center;gap:8px;font-size:15px}.timeline p{margin:5px 0;color:var(--el-text-color-secondary)}@media(max-width:1050px){.funnel-band{grid-template-columns:repeat(3,minmax(0,1fr))}.funnel-band>div{border-bottom:1px solid var(--el-border-color-lighter)}}@media(max-width:760px){.filter-control{width:100%}.funnel-band{grid-template-columns:repeat(2,minmax(0,1fr))}.funnel-band strong{font-size:17px}.form-grid,.detail-grid{grid-template-columns:1fr}.detail-grid .wide{grid-column:auto}}
</style>
