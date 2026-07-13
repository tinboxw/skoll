<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { PackageMinus, Plus, RefreshCw, ShoppingCart } from "lucide-vue-next";
import { ElMessage } from "element-plus";
import { DataTable, DetailDrawer, PageShell, type DataTableColumn } from "../../components/Common";
import { useButtonAccess } from "../../permissions/button";
import { createSalesOrder, createSalesOutbound, listCustomers, listSalesOrders, listSalesOutbounds, type PharmaCustomer, type SalesOrder, type SalesOutbound } from "../../pharma-oa/api";
import { useUserStore } from "../../stores/user";
import { toErrorMessage } from "../../utils/common";

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
const orderColumns: DataTableColumn[] = [{ key: "number", label: "Sales order", minWidth: 140 }, { key: "customerId", label: "Customer", minWidth: 170 }, { key: "lineSummary", label: "Products", minWidth: 180 }, { key: "totalAmount", label: "Total", width: 120 }, { key: "status", label: "Status", width: 100 }, { key: "createdAt", label: "Created", minWidth: 170 }];
const outboundColumns: DataTableColumn[] = [{ key: "number", label: "Outbound", minWidth: 140 }, { key: "salesOrderId", label: "Sales order", minWidth: 170 }, { key: "position", label: "Position", minWidth: 220 }, { key: "batchSummary", label: "Batch movement", minWidth: 180 }, { key: "status", label: "Status", width: 110 }, { key: "shippedAt", label: "Shipped", minWidth: 170 }];

onMounted(() => void refresh());

async function refresh() {
	error.value = "";
	loading.value = true;
	try {
		const requests: Promise<unknown>[] = [];
		if (canReadOrders.value) requests.push(listSalesOrders().then((items) => { orders.value = items; }));
		if (canReadOutbounds.value) requests.push(listSalesOutbounds().then((items) => { outbounds.value = items; }));
		if (canCreateOrder.value) requests.push(listCustomers({ status: "active", scope: { includeAll: true } }).then((items) => { customers.value = items; }));
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
			ElMessage.success("Sales order created");
		} else {
			const order = selectedOrder.value;
			if (!order) throw new Error("Select a sales order");
			const productId = form.productId || order.lines[0]?.productId || "";
			const item = await createSalesOutbound({ number: form.number, salesOrderId: order.id, warehouseId: form.warehouseId, areaId: form.areaId, locationId: form.locationId, actorId: actorId.value, lines: [{ productId, quantity: Number(form.quantity), batchId: form.batchId }] });
			outbounds.value.unshift(item);
			ElMessage.success("Sales outbound completed");
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
	<PageShell title="Sales & Outbound" description="Validate customer qualifications and ship batch-controlled inventory." :loading="loading" :error="error" :no-permission="!canRead" no-permission-title="No sales access" no-permission-description="The selected view requires its sales read permission.">
		<template #actions>
			<el-button :icon="RefreshCw" :loading="loading" @click="refresh">Refresh</el-button>
			<el-button type="primary" :icon="Plus" :disabled="!canCreate || (mode === 'orders' && customers.length === 0) || (mode === 'outbounds' && orders.length === 0)" @click="openCreate">{{ mode === "orders" ? "New order" : "New outbound" }}</el-button>
		</template>
		<el-tabs v-model="mode" class="view-tabs" @tab-change="refresh">
			<el-tab-pane label="Sales orders" name="orders" />
			<el-tab-pane label="Sales outbounds" name="outbounds" />
		</el-tabs>
		<section class="summary">
			<component :is="mode === 'orders' ? ShoppingCart : PackageMinus" :size="20" />
			<div><strong>{{ mode === "orders" ? orders.length : outbounds.length }}</strong><span>{{ mode === "orders" ? " open sales orders" : " completed outbounds" }}</span></div>
			<small>Customer qualification is checked again before every shipment.</small>
		</section>
		<DataTable v-if="mode === 'orders'" :rows="orderRows" :columns="orderColumns" row-key="id" :loading="loading" :error="error" empty-title="No sales orders" empty-description="Create an order for an active, qualified customer."><template #cell-status="{ row }"><el-tag>{{ row.status }}</el-tag></template></DataTable>
		<DataTable v-else :rows="outboundRows" :columns="outboundColumns" row-key="id" :loading="loading" :error="error" empty-title="No sales outbounds" empty-description="Select an order and ship an available inventory batch."><template #cell-status="{ row }"><el-tag type="success">{{ row.status }}</el-tag></template></DataTable>

		<DetailDrawer v-model="drawerOpen" :title="mode === 'orders' ? 'New sales order' : 'New sales outbound'" size="48%">
			<el-form label-position="top">
				<template v-if="mode === 'orders'">
					<el-form-item label="Customer" required><el-select v-model="form.customerId" filterable><el-option v-for="customer in customers" :key="customer.id" :label="`${customer.code} / ${customer.name}`" :value="customer.id" /></el-select></el-form-item>
					<div class="grid"><el-form-item label="Order number" required><el-input v-model="form.number" /></el-form-item><el-form-item label="Product ID" required><el-input v-model="form.productId" /></el-form-item></div>
					<div class="grid"><el-form-item label="Quantity" required><el-input-number v-model="form.quantity" :min="1" /></el-form-item><el-form-item label="Unit price" required><el-input-number v-model="form.unitPrice" :min="0" :precision="2" /></el-form-item></div>
				</template>
				<template v-else>
					<el-form-item label="Sales order" required><el-select v-model="form.salesOrderId" filterable><el-option v-for="order in orders" :key="order.id" :label="`${order.number} / ${order.customerId}`" :value="order.id" /></el-select></el-form-item>
					<div class="grid"><el-form-item label="Outbound number" required><el-input v-model="form.number" /></el-form-item><el-form-item label="Product ID" required><el-input v-model="form.productId" :placeholder="selectedOrder?.lines[0]?.productId" /></el-form-item></div>
					<div class="grid"><el-form-item label="Warehouse ID" required><el-input v-model="form.warehouseId" /></el-form-item><el-form-item label="Area ID" required><el-input v-model="form.areaId" /></el-form-item><el-form-item label="Location ID" required><el-input v-model="form.locationId" /></el-form-item></div>
					<div class="grid"><el-form-item label="Batch ID" required><el-input v-model="form.batchId" /></el-form-item><el-form-item label="Quantity" required><el-input-number v-model="form.quantity" :min="1" /></el-form-item></div>
				</template>
			</el-form>
			<template #footer><el-button :disabled="saving" @click="drawerOpen = false">Cancel</el-button><el-button type="primary" :loading="saving" :disabled="!canCreate" @click="save">{{ mode === "orders" ? "Create order" : "Complete outbound" }}</el-button></template>
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
