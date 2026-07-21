<script setup lang="ts">
import { useI18n } from "../../i18n";

import { computed, onMounted, reactive, ref } from "vue";
import { PackageMinus, Plus, RefreshCw, ShoppingCart } from "lucide-vue-next";
import { ElMessage } from "element-plus";
import { DataTable, DetailDrawer, PageShell, type DataTableColumn } from "../../components/Common";
import { useButtonAccess } from "../../permissions/button";
import { createSalesOrder, createSalesOutbound, listCustomers, listSalesOrders, listSalesOutbounds, type PharmaCustomer, type SalesOrder, type SalesOutbound } from "../../pharma-oa/api";
import { useUserStore } from "../../stores/user";
import { toErrorMessage } from "../../utils/common";

const { t, valueLabel } = useI18n();

type Mode = "orders" | "outbounds";
const access = useButtonAccess();
const userStore = useUserStore();
const mode = ref<Mode>("orders");
const loading = ref(false);
const saving = ref(false);
const error = ref("");
const drawerOpen = ref(false);
const customers = ref<PharmaCustomer[]>([]);
const orders = ref<SalesOrder[]>([]);
const outbounds = ref<SalesOutbound[]>([]);
const canReadOrders = computed(() => access.can("pharma_oa.sales.order.read"));
const canCreateOrder = computed(() => access.can("pharma_oa.sales.order.create"));
const canReadOutbounds = computed(() => access.can("pharma_oa.sales.outbound.read"));
const canCreateOutbound = computed(() => access.can("pharma_oa.sales.outbound.create"));
const canRead = computed(() => mode.value === "orders" ? canReadOrders.value : canReadOutbounds.value);
const canCreate = computed(() => mode.value === "orders" ? canCreateOrder.value : canCreateOutbound.value);
const actorId = computed(() => userStore.profile?.id?.trim() || "system");
const form = reactive({ number: "", customerId: "", productId: "", quantity: 1, unitPrice: 0, salesOrderId: "", warehouseId: "", areaId: "", locationId: "", batchId: "" });
const selectedOrder = computed(() => orders.value.find((item) => item.id === form.salesOrderId));
const orderRows = computed(() => orders.value.map((item) => ({ ...item, lineSummary: item.lines.map((line) => `${line.productId} x ${line.quantity}`).join(", ") })));
const outboundRows = computed(() => outbounds.value.map((item) => ({ ...item, position: `${item.warehouseId} / ${item.areaId} / ${item.locationId}`, batchSummary: item.lines.map((line) => `${line.batchId} (-${line.quantity})`).join(", ") })));
const orderColumns: DataTableColumn[] = [{ key: "number", label: t("pharma.sales.salesOrder"), minWidth: 140 }, { key: "customerId", label: t("pharma.sales.customer"), minWidth: 170 }, { key: "lineSummary", label: t("pharma.sales.products"), minWidth: 180 }, { key: "totalAmount", label: t("pharma.sales.total"), width: 120 }, { key: "status", label: t("pharma.sales.status"), width: 100 }, { key: "createdAt", label: t("pharma.sales.created"), minWidth: 170 }];
const outboundColumns: DataTableColumn[] = [{ key: "number", label: t("pharma.sales.outbound"), minWidth: 140 }, { key: "salesOrderId", label: t("pharma.sales.salesOrder"), minWidth: 170 }, { key: "position", label: t("pharma.sales.position"), minWidth: 220 }, { key: "batchSummary", label: t("pharma.sales.batchMovement"), minWidth: 180 }, { key: "status", label: t("pharma.sales.status"), width: 110 }, { key: "shippedAt", label: t("pharma.sales.shipped"), minWidth: 170 }];

onMounted(() => void refresh());

async function refresh() {
	error.value = "";
	loading.value = true;
	try {
		const requests: Promise<unknown>[] = [];
		if (canReadOrders.value) requests.push(listSalesOrders().then((items) => { orders.value = items; }));
		if (canReadOutbounds.value) requests.push(listSalesOutbounds().then((items) => { outbounds.value = items; }));
		if (canCreateOrder.value) requests.push(listCustomers({ status: "active", limit: 200 }).then((page) => { customers.value = page.items; }));
		await Promise.all(requests);
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		loading.value = false;
	}
}

function openCreate() {
	Object.assign(form, { number: "", customerId: customers.value[0]?.id || "", productId: "", quantity: 1, unitPrice: 0, salesOrderId: orders.value[0]?.id || "", warehouseId: "", areaId: "", locationId: "", batchId: "" });
	drawerOpen.value = true;
}

async function save() {
	if (!canCreate.value) return;
	error.value = "";
	saving.value = true;
	try {
		if (mode.value === "orders") {
			const item = await createSalesOrder({ number: form.number, customerId: form.customerId, actorId: actorId.value, lines: [{ productId: form.productId, quantity: Number(form.quantity), unitPrice: Number(form.unitPrice) }] });
			orders.value.unshift(item);
			ElMessage.success(t("pharma.sales.salesOrderCreated"));
		} else {
			const order = selectedOrder.value;
			if (!order) throw new Error("Select a sales order");
			const productId = form.productId || order.lines[0]?.productId || "";
			const item = await createSalesOutbound({ number: form.number, salesOrderId: order.id, warehouseId: form.warehouseId, areaId: form.areaId, locationId: form.locationId, actorId: actorId.value, lines: [{ productId, quantity: Number(form.quantity), batchId: form.batchId }] });
			outbounds.value.unshift(item);
			ElMessage.success(t("pharma.sales.salesOutboundCompleted"));
		}
		drawerOpen.value = false;
	} catch (cause) {
		error.value = toErrorMessage(cause);
	} finally {
		saving.value = false;
	}
}
</script>

<template>
	<PageShell :title="t('pharma.sales.salesOutbound')" :description="t('pharma.sales.validateCustomerQualificationsAndShipBatchControlledInventory')" :loading="loading" :error="error" :no-permission="!canRead" :no-permission-title="t('pharma.sales.noSalesAccess')" :no-permission-description="t('pharma.sales.theSelectedViewRequiresItsSalesReadPermission')">
		<template #actions>
			<el-button :icon="RefreshCw" :loading="loading" @click="refresh">{{ t("pharma.sales.refresh") }}</el-button>
			<el-button type="primary" :icon="Plus" :disabled="!canCreate || (mode === 'orders' && customers.length === 0) || (mode === 'outbounds' && orders.length === 0)" @click="openCreate">{{ mode === "orders" ? t("pharma.sales.newOrder") : t("pharma.sales.newOutbound") }}</el-button>
		</template>
		<el-tabs v-model="mode" class="view-tabs" @tab-change="refresh">
			<el-tab-pane :label="t('pharma.sales.salesOrders')" name="orders" />
			<el-tab-pane :label="t('pharma.sales.salesOutbounds')" name="outbounds" />
		</el-tabs>
		<section class="summary">
			<component :is="mode === 'orders' ? ShoppingCart : PackageMinus" :size="20" />
			<div><strong>{{ mode === "orders" ? orders.length : outbounds.length }}</strong><span>{{ mode === "orders" ? t("pharma.sales.openSalesOrders") : t("pharma.sales.completedOutbounds") }}</span></div>
			<small>{{ t("pharma.sales.customerQualificationIsCheckedAgainBeforeEveryShipment") }}</small>
		</section>
		<DataTable v-if="mode === 'orders'" :rows="orderRows" :columns="orderColumns" row-key="id" :loading="loading" :error="error" :empty-title="t('pharma.sales.noSalesOrders')" :empty-description="t('pharma.sales.createAnOrderForAnActiveQualifiedCustomer')"><template #cell-status="{ row }"><el-tag>{{ valueLabel(row.status) }}</el-tag></template></DataTable>
		<DataTable v-else :rows="outboundRows" :columns="outboundColumns" row-key="id" :loading="loading" :error="error" :empty-title="t('pharma.sales.noSalesOutbounds')" :empty-description="t('pharma.sales.selectAnOrderAndShipAnAvailableInventoryBatch')"><template #cell-status="{ row }"><el-tag type="success">{{ valueLabel(row.status) }}</el-tag></template></DataTable>

		<DetailDrawer v-model="drawerOpen" :title="mode === 'orders' ? t('pharma.sales.newSalesOrder') : t('pharma.sales.newSalesOutbound')" size="48%">
			<el-form label-position="top">
				<template v-if="mode === 'orders'">
					<el-form-item :label="t('pharma.sales.customer')" required><el-select v-model="form.customerId" filterable><el-option v-for="customer in customers" :key="customer.id" :label="`${customer.code} / ${customer.name}`" :value="customer.id" /></el-select></el-form-item>
					<div class="grid"><el-form-item :label="t('pharma.sales.orderNumber')" required><el-input v-model="form.number" /></el-form-item><el-form-item :label="t('pharma.sales.productId')" required><el-input v-model="form.productId" /></el-form-item></div>
					<div class="grid"><el-form-item :label="t('pharma.sales.quantity')" required><el-input-number v-model="form.quantity" :min="1" /></el-form-item><el-form-item :label="t('pharma.sales.unitPrice')" required><el-input-number v-model="form.unitPrice" :min="0" :precision="2" /></el-form-item></div>
				</template>
				<template v-else>
					<el-form-item :label="t('pharma.sales.salesOrder')" required><el-select v-model="form.salesOrderId" filterable><el-option v-for="order in orders" :key="order.id" :label="`${order.number} / ${order.customerId}`" :value="order.id" /></el-select></el-form-item>
					<div class="grid"><el-form-item :label="t('pharma.sales.outboundNumber')" required><el-input v-model="form.number" /></el-form-item><el-form-item :label="t('pharma.sales.productId')" required><el-input v-model="form.productId" :placeholder="selectedOrder?.lines[0]?.productId" /></el-form-item></div>
					<div class="grid"><el-form-item :label="t('pharma.sales.warehouseId')" required><el-input v-model="form.warehouseId" /></el-form-item><el-form-item :label="t('pharma.sales.areaId')" required><el-input v-model="form.areaId" /></el-form-item><el-form-item :label="t('pharma.sales.locationId')" required><el-input v-model="form.locationId" /></el-form-item></div>
					<div class="grid"><el-form-item :label="t('pharma.sales.batchId')" required><el-input v-model="form.batchId" /></el-form-item><el-form-item :label="t('pharma.sales.quantity')" required><el-input-number v-model="form.quantity" :min="1" /></el-form-item></div>
				</template>
			</el-form>
			<template #footer><el-button :disabled="saving" @click="drawerOpen = false">{{ t("pharma.sales.cancel") }}</el-button><el-button type="primary" :loading="saving" :disabled="!canCreate" @click="save">{{ mode === "orders" ? t("pharma.sales.createOrder") : t("pharma.sales.completeOutbound") }}</el-button></template>
		</DetailDrawer>
	</PageShell>
</template>

<style scoped>
.view-tabs { margin-bottom: 12px; }
.summary { display: flex; align-items: center; gap: 12px; min-height: 56px; padding: 0 4px 14px; border-bottom: 1px solid var(--el-border-color-lighter); }
.summary strong { font-size: 20px; }
.summary small { margin-left: auto; color: var(--el-text-color-secondary); }
.grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.grid:has(> :nth-child(3)) { grid-template-columns: repeat(3, minmax(0, 1fr)); }
@media (max-width: 760px) { .grid, .grid:has(> :nth-child(3)) { grid-template-columns: 1fr; } .summary { align-items: flex-start; flex-wrap: wrap; } .summary small { width: 100%; margin-left: 32px; } }
</style>
