<script setup lang="ts">
import { useI18n } from "../../i18n";

import { computed, onMounted, reactive, ref, watch } from "vue";
import { PackageCheck, Plus, RefreshCw } from "lucide-vue-next";
import { ElMessage } from "element-plus";
import { DataTable, DetailDrawer, PageShell, type DataTableColumn } from "../../components/Common";
import { useButtonAccess } from "../../permissions/button";
import { createPurchaseInbound, listPurchaseInbounds, listPurchaseOrders, type PurchaseInbound, type PurchaseOrder } from "../../pharma-oa/api";
import { useUserStore } from "../../stores/user";
import { toErrorMessage } from "../../utils/common";

const { t, valueLabel } = useI18n();

type InboundRow = Record<string, unknown> & PurchaseInbound & { position: string; batchSummary: string };
const access = useButtonAccess(); const userStore = useUserStore();
const loading=ref(false),saving=ref(false),error=ref(""),drawerOpen=ref(false);
const inbounds=ref<PurchaseInbound[]>([]),orders=ref<PurchaseOrder[]>([]);
const canRead=computed(()=>access.can("pharma_oa.inbound.read")); const canCreate=computed(()=>access.can("pharma_oa.inbound.create"));
const actorId=computed(()=>userStore.profile?.id?.trim()||"system");
const today=()=>new Date().toISOString().slice(0,10); const nextYear=()=>{const d=new Date();d.setFullYear(d.getFullYear()+1);return d.toISOString().slice(0,10)};
const form=reactive({number:"",purchaseOrderId:"",warehouseId:"",areaId:"",locationId:"",batchNo:"",productionDate:today(),expiresAt:nextYear(),attachmentFileId:"",attachmentFileName:""});
const selectedOrder=computed(()=>orders.value.find(item=>item.id===form.purchaseOrderId));
const rows=computed<InboundRow[]>(()=>inbounds.value.map(item=>({...item,position:`${item.warehouseId} / ${item.areaId} / ${item.locationId}`,batchSummary:item.lines.map(line=>`${line.batchNo} (${line.quantity})`).join(", ")})));
const columns:DataTableColumn[]=[{key:"number",label:t("pharma.purchaseInbound.inbound"),minWidth:140},{key:"purchaseOrderId",label:t("pharma.purchaseInbound.purchaseOrder"),minWidth:180},{key:"position",label:t("pharma.purchaseInbound.position"),minWidth:220},{key:"batchSummary",label:t("pharma.purchaseInbound.batches"),minWidth:180},{key:"status",label:t("pharma.purchaseInbound.status"),width:110},{key:"receivedAt",label:t("pharma.purchaseInbound.received"),minWidth:170}];
onMounted(()=>void refresh()); watch(()=>form.purchaseOrderId,()=>{if(!form.number&&selectedOrder.value)form.number=`IN-${selectedOrder.value.number.replace(/^PO-/,"")}`});
async function refresh(){error.value="";if(!canRead.value)return;loading.value=true;try{[inbounds.value,orders.value]=await Promise.all([listPurchaseInbounds(),listPurchaseOrders()])}catch(e){error.value=toErrorMessage(e)}finally{loading.value=false}}
function openCreate(){Object.assign(form,{number:"",purchaseOrderId:orders.value[0]?.id||"",warehouseId:"",areaId:"",locationId:"",batchNo:"",productionDate:today(),expiresAt:nextYear(),attachmentFileId:"",attachmentFileName:""});drawerOpen.value=true}
async function save(){if(!canCreate.value||!selectedOrder.value){error.value=t("pharma.purchaseInbound.selectAnApprovedPurchaseOrder");return}saving.value=true;error.value="";try{const order=selectedOrder.value;const item=await createPurchaseInbound({number:form.number,purchaseOrderId:order.id,warehouseId:form.warehouseId,areaId:form.areaId,locationId:form.locationId,actorId:actorId.value,lines:order.lines.map(line=>({productId:line.productId,quantity:Number(line.quantity),batchNo:form.batchNo,productionDate:`${form.productionDate}T00:00:00Z`,expiresAt:`${form.expiresAt}T00:00:00Z`})),attachments:form.attachmentFileId&&form.attachmentFileName?[{fileId:form.attachmentFileId,fileName:form.attachmentFileName,size:0}]:[]});inbounds.value.unshift(item);drawerOpen.value=false;ElMessage.success(t("pharma.purchaseInbound.purchaseInboundCompleted"))}catch(e){error.value=toErrorMessage(e)}finally{saving.value=false}}
</script>

<template>
	<PageShell :title="t('pharma.purchaseInbound.purchaseInbound')" :description="t('pharma.purchaseInbound.receiveApprovedPurchaseOrdersIntoBatchControlledInventory')" :loading="loading" :error="error" :no-permission="!canRead" :no-permission-title="t('pharma.purchaseInbound.noInboundAccess')" :no-permission-description="t('pharma.purchaseInbound.purchaseInboundRequiresPharmaOaInboundReadPermission')">
		<template #actions><el-button :icon="RefreshCw" :loading="loading" @click="refresh">{{ t("pharma.purchaseInbound.refresh") }}</el-button><el-button type="primary" :icon="Plus" :disabled="!canCreate||orders.length===0" @click="openCreate">{{ t("pharma.purchaseInbound.newInbound") }}</el-button></template>
		<section class="summary"><PackageCheck :size="20"/><div><strong>{{ inbounds.length }}</strong><span> {{ t("pharma.purchaseInbound.completedInbound") }}{{ inbounds.length===1?"":"s" }}</span></div><small>{{ orders.length }} {{ t("pharma.purchaseInbound.approvedPurchaseOrder") }}{{ orders.length===1?"":"s" }} {{ t("pharma.purchaseInbound.available") }}</small></section>
		<DataTable :rows="rows" :columns="columns" row-key="id" :loading="loading" :error="error" :empty-title="t('pharma.purchaseInbound.noPurchaseInbounds')" :empty-description="t('pharma.purchaseInbound.approveAPurchaseRequestThenReceiveItsOrderIntoInventory')"><template #cell-status="{ row }"><el-tag type="success">{{ valueLabel(row.status) }}</el-tag></template></DataTable>
		<DetailDrawer v-model="drawerOpen" :title="t('pharma.purchaseInbound.newPurchaseInbound')" size="48%"><el-form label-position="top">
			<el-form-item :label="t('pharma.purchaseInbound.purchaseOrder')" required><el-select v-model="form.purchaseOrderId" filterable><el-option v-for="order in orders" :key="order.id" :label="`${order.number} / ${order.supplierId}`" :value="order.id"/></el-select></el-form-item>
			<div class="grid"><el-form-item :label="t('pharma.purchaseInbound.inboundNumber')" required><el-input v-model="form.number"/></el-form-item><el-form-item :label="t('pharma.purchaseInbound.batchNumber')" required><el-input v-model="form.batchNo"/></el-form-item></div>
			<div class="grid"><el-form-item :label="t('pharma.purchaseInbound.warehouseId')" required><el-input v-model="form.warehouseId"/></el-form-item><el-form-item :label="t('pharma.purchaseInbound.areaId')" required><el-input v-model="form.areaId"/></el-form-item><el-form-item :label="t('pharma.purchaseInbound.locationId')" required><el-input v-model="form.locationId"/></el-form-item></div>
			<div class="grid"><el-form-item :label="t('pharma.purchaseInbound.productionDate')" required><el-date-picker v-model="form.productionDate" type="date" value-format="YYYY-MM-DD"/></el-form-item><el-form-item :label="t('pharma.purchaseInbound.expiryDate')" required><el-date-picker v-model="form.expiresAt" type="date" value-format="YYYY-MM-DD"/></el-form-item></div>
			<div class="grid"><el-form-item :label="t('pharma.purchaseInbound.attachmentFileId')"><el-input v-model="form.attachmentFileId"/></el-form-item><el-form-item :label="t('pharma.purchaseInbound.attachmentFileName')"><el-input v-model="form.attachmentFileName"/></el-form-item></div>
		</el-form><template #footer><el-button @click="drawerOpen=false">{{ t("pharma.purchaseInbound.cancel") }}</el-button><el-button type="primary" :loading="saving" @click="save">{{ t("pharma.purchaseInbound.completeInbound") }}</el-button></template></DetailDrawer>
	</PageShell>
</template>
<style scoped>.summary{display:flex;align-items:center;gap:10px;padding:12px;border:1px solid var(--color-border);border-radius:8px;background:var(--color-surface)}.summary small{margin-left:auto;color:var(--color-text-secondary)}.grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}@media(max-width:720px){.grid{grid-template-columns:1fr}.summary{align-items:flex-start;flex-wrap:wrap}.summary small{width:100%;margin-left:30px}}</style>
