<script setup lang="ts">
import { useI18n } from "../../i18n";

import { computed, onMounted, reactive, ref } from "vue";
import { CheckCircle2, Eye, FileText, Megaphone, Plus, RefreshCw, Send } from "lucide-vue-next";
import { ElMessage, ElMessageBox } from "element-plus";
import { DataTable, DetailDrawer, PageShell, type DataTableColumn } from "../../components/Common";
import { useButtonAccess } from "../../permissions/button";
import { confirmAnnouncementRead, createAnnouncement, listAnnouncementReadConfirmations, listAnnouncements, publishAnnouncement, type AnnouncementReadConfirmation, type PharmaAnnouncement } from "../../pharma-oa/api";
import { useUserStore } from "../../stores/user";
import { toErrorMessage } from "../../utils/common";

const { t, valueLabel } = useI18n();

type AnnouncementRow = Record<string, unknown> & PharmaAnnouncement & { audienceText: string; publishedText: string };
const access = useButtonAccess();
const userStore = useUserStore();
const loading = ref(false), saving = ref(false), publishingId = ref(""), confirming = ref(false), receiptLoading = ref(false);
const error = ref(""), receiptError = ref(""), createOpen = ref(false), detailOpen = ref(false), organizationId = ref("");
const items = ref<PharmaAnnouncement[]>([]), selected = ref<PharmaAnnouncement | null>(null), receipts = ref<AnnouncementReadConfirmation[]>([]);
const canRead = computed(() => access.can("pharma_oa.announcement.read"));
const canCreate = computed(() => access.can("pharma_oa.announcement.create"));
const canPublish = computed(() => access.can("pharma_oa.announcement.publish"));
const canConfirm = computed(() => access.can("pharma_oa.announcement.confirm"));
const canReadReceipts = computed(() => access.can("pharma_oa.announcement.receipt.read"));
const actorId = computed(() => userStore.profile?.id?.trim() || "system");
const roleId = computed(() => userStore.profile?.role?.trim() || "");
const form = reactive({ kind: "announcement" as "announcement" | "policy", title: "", content: "", organizationIds: [] as string[], roleIds: [] as string[], fileId: "", fileName: "" });
const rows = computed<AnnouncementRow[]>(() => items.value.map((item) => ({ ...item, audienceText: [...item.audience.organizationIds.map((id) => `Org: ${id}`), ...item.audience.roleIds.map((id) => `Role: ${id}`)].join(", "), publishedText: item.publishedAt ? new Date(item.publishedAt).toLocaleString() : "-" })));
const columns: DataTableColumn[] = [{ key: "kind", label: t("pharma.announcement.type"), width: 120 }, { key: "title", label: t("pharma.announcement.titleColumn"), minWidth: 220 }, { key: "audienceText", label: t("pharma.announcement.audienceColumn"), minWidth: 240 }, { key: "status", label: t("pharma.announcement.status"), width: 110 }, { key: "publishedText", label: t("pharma.announcement.published"), minWidth: 170 }];

onMounted(() => void refresh());

async function refresh() {
	if (!canRead.value) return;
	loading.value = true; error.value = "";
	try { items.value = await listAnnouncements({ organizationIds: organizationId.value ? [organizationId.value] : [], roleIds: roleId.value ? [roleId.value] : [], includeDraft: canCreate.value || canPublish.value }); }
	catch (cause) { error.value = toErrorMessage(cause); }
	finally { loading.value = false; }
}

function openCreate() {
	Object.assign(form, { kind: "announcement", title: "", content: "", organizationIds: [], roleIds: roleId.value ? [roleId.value] : [], fileId: "", fileName: "" });
	createOpen.value = true;
}

async function save() {
	if (!canCreate.value || !form.title.trim() || !form.content.trim() || form.organizationIds.length + form.roleIds.length === 0) return;
	if (form.kind === "policy" && (!form.fileId.trim() || !form.fileName.trim())) return;
	saving.value = true; error.value = "";
	try {
		const item = await createAnnouncement({ kind: form.kind, title: form.title, content: form.content, audience: { organizationIds: form.organizationIds, roleIds: form.roleIds }, documents: form.fileId && form.fileName ? [{ fileId: form.fileId, fileName: form.fileName }] : [], actorId: actorId.value });
		items.value.unshift(item); createOpen.value = false; ElMessage.success(t("pharma.announcement.draftCreated"));
	} catch (cause) { error.value = toErrorMessage(cause); }
	finally { saving.value = false; }
}

async function publish(item: PharmaAnnouncement) {
	if (!canPublish.value || item.status !== "draft") return;
	await ElMessageBox.confirm(`${t("pharma.announcement.publishAnnouncement")}: ${item.title}?`, t("pharma.announcement.publishAnnouncement"), { type: "warning", confirmButtonText: t("pharma.announcement.publish"), cancelButtonText: t("common.cancel") });
	publishingId.value = item.id; error.value = "";
	try { const updated = await publishAnnouncement(item.id, actorId.value); items.value = items.value.map((current) => current.id === updated.id ? updated : current); if (selected.value?.id === updated.id) selected.value = updated; ElMessage.success(t("pharma.announcement.published")); }
	catch (cause) { error.value = toErrorMessage(cause); }
	finally { publishingId.value = ""; }
}

async function openDetail(item: PharmaAnnouncement) {
	selected.value = item; receipts.value = []; receiptError.value = ""; detailOpen.value = true;
	if (canReadReceipts.value) {
		receiptLoading.value = true;
		try { receipts.value = await listAnnouncementReadConfirmations(item.id); }
		catch (cause) { receiptError.value = toErrorMessage(cause); }
		finally { receiptLoading.value = false; }
	}
}

async function confirmRead() {
	if (!selected.value || !canConfirm.value) return;
	confirming.value = true; receiptError.value = "";
	try { const receipt = await confirmAnnouncementRead(selected.value.id, { organizationIds: organizationId.value ? [organizationId.value] : [], roleIds: roleId.value ? [roleId.value] : [] }); if (!receipts.value.some((item) => item.userId === receipt.userId)) receipts.value.push(receipt); ElMessage.success(t("pharma.announcement.readConfirmed")); }
	catch (cause) { receiptError.value = toErrorMessage(cause); }
	finally { confirming.value = false; }
}
</script>

<template>
	<PageShell :title="t('pharma.announcement.title')" :description="t('pharma.announcement.description')" :loading="loading" :error="error" :no-permission="!canRead" :no-permission-title="t('pharma.announcement.noAnnouncementAccess')" :no-permission-description="t('pharma.announcement.thisPageRequiresPharmaOaAnnouncementReadPermission')">
		<template #actions>
			<el-input v-model="organizationId" clearable :placeholder="t('pharma.announcement.currentOrganizationId')" class="organization-filter" @keyup.enter="refresh" />
			<el-button :icon="RefreshCw" :loading="loading" @click="refresh">{{ t("pharma.announcement.refresh") }}</el-button>
			<el-button type="primary" :icon="Plus" :disabled="!canCreate" @click="openCreate">{{ t("pharma.announcement.newDraft") }}</el-button>
		</template>
		<section class="summary"><component :is="items.some((item) => item.kind === 'policy') ? FileText : Megaphone" :size="20" /><strong>{{ items.length }}</strong><span>{{ t("pharma.announcement.visibleCommunications") }}</span><small>{{ t("pharma.announcement.audience") }} {{ organizationId || "no organization" }} / {{ roleId || "no role" }}</small></section>
		<DataTable :rows="rows" :columns="columns" row-key="id" :loading="loading" :error="error" :empty-text="t('pharma.announcement.empty')">
			<template #cell-kind="{ row }"><el-tag :type="row.kind === 'policy' ? 'warning' : 'info'">{{ valueLabel(row.kind) }}</el-tag></template>
			<template #cell-status="{ row }"><el-tag :type="row.status === 'published' ? 'success' : 'info'">{{ valueLabel(row.status) }}</el-tag></template>
			<template #actions="{ row }"><el-tooltip :content="t('pharma.announcement.viewDetails')"><el-button circle :icon="Eye" :aria-label="t('pharma.announcement.viewDetails')" @click="openDetail(row as PharmaAnnouncement)" /></el-tooltip><el-tooltip v-if="row.status === 'draft'" :content="t('pharma.announcement.publish')"><el-button circle type="primary" :icon="Send" :aria-label="t('pharma.announcement.publish')" :loading="publishingId === row.id" :disabled="!canPublish" @click="publish(row as PharmaAnnouncement)" /></el-tooltip></template>
		</DataTable>

		<DetailDrawer v-model="createOpen" :title="t('pharma.announcement.draftTitle')" size="48%">
			<el-form label-position="top">
				<el-form-item :label="t('pharma.announcement.type')" required><el-segmented v-model="form.kind" :options="[{ label: valueLabel('announcement'), value: 'announcement' }, { label: valueLabel('policy'), value: 'policy' }]" /></el-form-item>
				<el-form-item :label="t('pharma.announcement.titleColumn')" required><el-input v-model="form.title" maxlength="160" show-word-limit /></el-form-item>
				<el-form-item :label="t('pharma.announcement.content')" required><el-input v-model="form.content" type="textarea" :rows="7" maxlength="5000" show-word-limit /></el-form-item>
				<div class="grid"><el-form-item :label="t('pharma.announcement.organizations')"><el-select v-model="form.organizationIds" multiple filterable allow-create default-first-option :placeholder="t('pharma.announcement.addOrganizationIds')" /></el-form-item><el-form-item :label="t('pharma.announcement.roles')"><el-select v-model="form.roleIds" multiple filterable allow-create default-first-option :placeholder="t('pharma.announcement.addRoleIds')" /></el-form-item></div>
				<div v-if="form.kind === 'policy'" class="grid"><el-form-item :label="t('pharma.announcement.documentFileId')" required><el-input v-model="form.fileId" /></el-form-item><el-form-item :label="t('pharma.announcement.documentFileName')" required><el-input v-model="form.fileName" /></el-form-item></div>
			</el-form>
			<template #footer><el-button :disabled="saving" @click="createOpen = false">{{ t("pharma.announcement.cancel") }}</el-button><el-button type="primary" :loading="saving" :disabled="!form.title.trim() || !form.content.trim() || form.organizationIds.length + form.roleIds.length === 0 || (form.kind === 'policy' && (!form.fileId.trim() || !form.fileName.trim()))" @click="save">{{ t("pharma.announcement.createDraft") }}</el-button></template>
		</DetailDrawer>

		<DetailDrawer v-model="detailOpen" :title="selected?.title || 'Announcement'" size="48%">
			<template v-if="selected"><div class="detail-meta"><el-tag>{{ valueLabel(selected.kind) }}</el-tag><el-tag :type="selected.status === 'published' ? 'success' : 'info'">{{ valueLabel(selected.status) }}</el-tag><span>{{ selected.publishedAt ? new Date(selected.publishedAt).toLocaleString() : t('pharma.announcement.notPublished') }}</span></div><p class="content">{{ selected.content }}</p><el-link v-for="document in selected.documents" :key="document.fileId" type="primary" :underline="false"><FileText :size="16" />{{ document.fileName }} ({{ document.fileId }})</el-link><el-alert v-if="receiptError" type="error" :title="receiptError" show-icon :closable="false" /><section v-if="canReadReceipts" class="receipts" v-loading="receiptLoading"><h3>{{ t("pharma.announcement.readConfirmations") }}</h3><el-empty v-if="!receiptLoading && receipts.length === 0" :description="t('pharma.announcement.emptyConfirmations')" :image-size="64" /><div v-for="receipt in receipts" :key="receipt.userId" class="receipt"><CheckCircle2 :size="16" /><span>{{ receipt.userId }}</span><time>{{ new Date(receipt.readAt).toLocaleString() }}</time></div></section></template>
			<template #footer><el-button @click="detailOpen = false">{{ t("pharma.announcement.close") }}</el-button><el-button v-if="selected?.status === 'published'" type="primary" :icon="CheckCircle2" :loading="confirming" :disabled="!canConfirm" @click="confirmRead">{{ t("pharma.announcement.confirmRead") }}</el-button></template>
		</DetailDrawer>
	</PageShell>
</template>

<style scoped>
.organization-filter{width:220px}.summary{display:flex;align-items:center;gap:10px;min-height:52px;padding:0 4px 14px;border-bottom:1px solid var(--el-border-color-lighter)}.summary strong{font-size:20px}.summary small{margin-left:auto;color:var(--el-text-color-secondary)}.grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.detail-meta{display:flex;align-items:center;gap:8px;color:var(--el-text-color-secondary)}.content{white-space:pre-wrap;line-height:1.7}.receipts{margin-top:24px;border-top:1px solid var(--el-border-color-lighter)}.receipts h3{font-size:15px}.receipt{display:grid;grid-template-columns:20px minmax(0,1fr) auto;gap:8px;align-items:center;padding:9px 0;border-bottom:1px solid var(--el-border-color-extra-light)}.receipt time{color:var(--el-text-color-secondary)}@media(max-width:760px){.organization-filter{width:100%}.grid{grid-template-columns:1fr}.summary{align-items:flex-start;flex-wrap:wrap}.summary small{width:100%;margin-left:30px}.receipt{grid-template-columns:20px minmax(0,1fr)}.receipt time{grid-column:2}}
</style>
