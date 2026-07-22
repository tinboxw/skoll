<script setup lang="ts">
import { computed, ref } from "vue";

import {
	DocumentDetail,
	DocumentForm,
	DocumentList,
	DocumentPrintView,
	createDocumentDraft,
	type DocumentLocale,
	type DocumentRecord,
	type DocumentSchema,
	type WorkflowInstance
} from "../../../packages/skoll-document-ui/src";

const params = new URLSearchParams(location.search);
const view = params.get("view") || "list";
const state = params.get("state") || "ready";
const locale = (params.get("locale") === "en-US" ? "en-US" : "zh-CN") as DocumentLocale;
const schema: DocumentSchema = {
	key: "purchase_order", name: locale === "en-US" ? "Purchase order" : "采购订单", version: 1, initialState: "draft",
	header: [
		{ key: "subject", label: locale === "en-US" ? "Subject" : "主题", type: "string", required: true },
		{ key: "budget", label: locale === "en-US" ? "Budget" : "预算", type: "money", required: true, sensitive: true }
	],
	lines: [{
		key: "items", name: locale === "en-US" ? "Items" : "采购明细", minItems: 1, maxItems: 20,
		fields: [
			{ key: "product", label: locale === "en-US" ? "Product" : "产品", type: "string", required: true },
			{ key: "quantity", label: locale === "en-US" ? "Quantity" : "数量", type: "quantity", required: true }
		]
	}],
	states: [
		{ key: "draft", name: locale === "en-US" ? "Draft" : "草稿", terminal: false },
		{ key: "pending", name: locale === "en-US" ? "Pending" : "审批中", terminal: false },
		{ key: "approved", name: locale === "en-US" ? "Approved" : "已通过", terminal: true }
	],
	actions: [{ key: "submit", name: locale === "en-US" ? "Submit" : "提交", from: ["draft"], to: "pending", requiresComment: false }]
};
const draft = ref(createDocumentDraft(schema, { id: "po-1", number: "PO-20260722-001", title: locale === "en-US" ? "Cold-chain medicine purchase" : "冷链药品采购" }));
draft.value.header.subject = { type: "string", value: locale === "en-US" ? "Quarterly replenishment" : "季度补货" };
draft.value.header.budget = { type: "money", value: "128000.50", currency: "CNY" };
draft.value.lines.items[0].values.product = { type: "string", value: locale === "en-US" ? "Insulin injection" : "胰岛素注射液" };
draft.value.lines.items[0].values.quantity = { type: "quantity", value: "120", unit: "box" };
const record: DocumentRecord = {
	...draft.value, state: "pending", version: 2,
	metadata: { createdAt: "2026-07-22T08:00:00Z", updatedAt: "2026-07-22T09:30:00Z", createdBy: "Zhang Wei", updatedBy: "Li Ming" }
};
const workflow: WorkflowInstance = {
	id: "workflow-1", definitionId: "purchase-approval", definitionKey: "purchase_order", businessType: "purchase_order", businessId: record.id,
	title: record.title, status: "running", starter: { id: "user-1", name: "Zhang Wei" }, currentNode: locale === "en-US" ? "Finance review" : "财务复核",
	tasks: [{ id: "task-1", instanceId: "workflow-1", nodeId: "finance", assignee: { id: "finance-1", name: "Chen Yu" }, status: "pending", createdAt: "2026-07-22T09:00:00Z" }],
	timeline: [], createdAt: "2026-07-22T08:00:00Z", updatedAt: "2026-07-22T09:30:00Z"
};
const page = computed(() => ({
	items: state === "empty" ? [] : [
		{ id: record.id, type: record.type, number: record.number, title: record.title, state: record.state, version: record.version, createdAt: record.metadata.createdAt, updatedAt: record.metadata.updatedAt, createdBy: record.metadata.createdBy, updatedBy: record.metadata.updatedBy },
		{ id: "po-2", type: record.type, number: "PO-20260722-002", title: locale === "en-US" ? "Clinic supplies" : "诊所耗材采购", state: "approved", version: 3, createdAt: record.metadata.createdAt, updatedAt: record.metadata.updatedAt, createdBy: "Wang Min", updatedBy: "Wang Min" }
	],
	hasMore: state === "ready",
	nextCursor: state === "ready" ? "next" : undefined
}));
</script>

<template>
	<main class="fixture-shell">
		<DocumentList
			v-if="view === 'list'"
			:schema="schema"
			:page="page"
			:loading="state === 'loading'"
			:error="state === 'error' ? (locale === 'en-US' ? 'Request failed' : '请求失败') : ''"
			:forbidden="state === 'forbidden'"
			:can-create="true"
			:can-export="true"
			:locale="locale"
		/>
		<DocumentForm v-else-if="view === 'form'" v-model="draft" :schema="schema" :locale="locale" />
		<DocumentDetail
			v-else-if="view === 'detail'"
			:schema="schema"
			:document="record"
			:workflow="workflow"
			:available-actions="['approve', 'reject', 'delegate']"
			:delegate-options="[{ id: 'user-2', name: 'Liu Fang' }]"
			:attachments="[{ id: 'attachment-1', documentId: record.id, file: { id: 'file-1', name: 'quality-certificate.pdf', size: 125600, mime: 'application/pdf' }, addedBy: { id: 'user-1', name: 'Zhang Wei' }, addedAt: '2026-07-22T08:30:00Z', addedSequence: 2 }]"
			:comments="[{ id: 'comment-1', documentId: record.id, body: locale === 'en-US' ? 'Qualification checked.' : '资质已核验。', author: { id: 'user-2', name: 'Liu Fang' }, createdAt: '2026-07-22T09:10:00Z', sequence: 3 }]"
			:timeline="[{ id: 'event-1', sequence: 1, documentId: record.id, kind: 'action', action: 'submit', actor: { id: 'user-1', name: 'Zhang Wei' }, occurredAt: '2026-07-22T08:00:00Z' }]"
			:can-add-attachment="true"
			:can-remove-attachment="true"
			:can-comment="true"
			:can-print="true"
			:can-export="true"
			:locale="locale"
		/>
		<DocumentPrintView
			v-else
			:payload="{ schema, document: record, workflow, redactedFields: ['header.budget'], generatedAt: '2026-07-22T10:00:00Z' }"
			:locale="locale"
		/>
	</main>
</template>

<style scoped>
.fixture-shell {
	width: min(100%, 1320px);
	min-height: 100vh;
	margin: 0 auto;
	padding: var(--content-padding);
}
</style>
