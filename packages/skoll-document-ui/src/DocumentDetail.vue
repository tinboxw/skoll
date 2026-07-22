<script setup lang="ts">
import { Download, Printer } from "lucide-vue-next";
import { computed } from "vue";

import DocumentApprovalPanel, { type DocumentApprovalAction } from "./DocumentApprovalPanel.vue";
import DocumentAttachments from "./DocumentAttachments.vue";
import DocumentComments from "./DocumentComments.vue";
import DocumentTimeline from "./DocumentTimeline.vue";
import { documentStateTone, formatDocumentDate, formatDocumentValue } from "./format";
import { documentKitMessages } from "./messages";
import type {
	DocumentActionRequest,
	DocumentAttachment,
	DocumentComment,
	DocumentKitMessages,
	DocumentLocale,
	DocumentRecord,
	DocumentSchema,
	DocumentTimelineEvent,
	WorkflowActor,
	WorkflowInstance
} from "./types";

const props = withDefaults(defineProps<{
	schema: DocumentSchema;
	document: DocumentRecord;
	workflow: WorkflowInstance;
	attachments?: DocumentAttachment[];
	comments?: DocumentComment[];
	timeline?: DocumentTimelineEvent[];
	redactedFields?: string[];
	availableActions?: DocumentApprovalAction[];
	delegateOptions?: WorkflowActor[];
	busy?: boolean;
	canAddAttachment?: boolean;
	canRemoveAttachment?: boolean;
	canComment?: boolean;
	canPrint?: boolean;
	canExport?: boolean;
	locale?: DocumentLocale;
	messages?: Partial<DocumentKitMessages>;
}>(), {
	attachments: () => [],
	comments: () => [],
	timeline: () => [],
	redactedFields: () => [],
	availableActions: () => [],
	delegateOptions: () => [],
	busy: false,
	canAddAttachment: false,
	canRemoveAttachment: false,
	canComment: false,
	canPrint: false,
	canExport: false,
	locale: "zh-CN",
	messages: () => ({})
});

const emit = defineEmits<{
	(e: "action", value: DocumentActionRequest): void;
	(e: "addAttachment" | "print" | "export"): void;
	(e: "downloadAttachment" | "removeAttachment", value: DocumentAttachment): void;
	(e: "comment", value: string): void;
}>();

const text = computed(() => documentKitMessages(props.locale, props.messages));
const stateName = computed(() => props.schema.states.find((state) => state.key === props.document.state)?.name || props.document.state);
const redacted = computed(() => new Set(props.redactedFields));
</script>

<template>
	<article class="document-detail" :aria-label="document.title">
		<header class="document-detail__header">
			<div class="document-detail__identity">
				<div class="document-detail__eyebrow">
					<span>{{ document.number }}</span>
					<el-tag :type="documentStateTone(document.state)">{{ stateName }}</el-tag>
				</div>
				<h2>{{ document.title }}</h2>
				<p>{{ schema.name }} · {{ text.version }} {{ document.version }}</p>
			</div>
			<div class="document-detail__commands">
				<el-button v-if="canExport" @click="emit('export')"><Download :size="16" />{{ text.export }}</el-button>
				<el-button v-if="canPrint" type="primary" @click="emit('print')"><Printer :size="16" />{{ text.print }}</el-button>
			</div>
		</header>

		<el-alert v-if="redactedFields.length" :title="text.sensitiveHidden" type="warning" show-icon :closable="false" />

		<section class="document-detail__metadata" :aria-label="text.documentInfo">
			<dl>
				<div><dt>{{ text.createdAt }}</dt><dd>{{ formatDocumentDate(document.metadata.createdAt, locale) }}</dd></div>
				<div><dt>{{ text.createdBy }}</dt><dd>{{ document.metadata.createdBy }}</dd></div>
				<div><dt>{{ text.updatedAt }}</dt><dd>{{ formatDocumentDate(document.metadata.updatedAt, locale) }}</dd></div>
				<div><dt>{{ text.updatedBy }}</dt><dd>{{ document.metadata.updatedBy }}</dd></div>
			</dl>
		</section>

		<section class="document-detail__fields">
			<div v-for="field in schema.header" :key="field.key" class="document-detail__field">
				<dt>{{ field.label }}</dt>
				<dd v-if="redacted.has(`header.${field.key}`)" class="document-detail__redacted">{{ text.redacted }}</dd>
				<dd v-else :class="{ 'document-detail__code': field.type === 'json' }">{{ formatDocumentValue(field, document.header[field.key], text) }}</dd>
			</div>
		</section>

		<section v-for="lineSchema in schema.lines" :key="lineSchema.key" class="document-detail__line-section">
			<h3>{{ lineSchema.name }}</h3>
			<div class="document-detail__table-wrap">
				<table>
					<thead><tr><th v-for="field in lineSchema.fields" :key="field.key">{{ field.label }}</th></tr></thead>
					<tbody>
						<tr v-for="line in document.lines[lineSchema.key] || []" :key="line.id">
							<td v-for="field in lineSchema.fields" :key="field.key">
								<span v-if="redacted.has(`lines.${lineSchema.key}.${field.key}`)" class="document-detail__redacted">{{ text.redacted }}</span>
								<span v-else>{{ formatDocumentValue(field, line.values[field.key], text) }}</span>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
		</section>

		<el-tabs class="document-detail__tabs">
			<el-tab-pane :label="text.workflow">
				<DocumentApprovalPanel
					:workflow="workflow"
					:available-actions="availableActions"
					:delegate-options="delegateOptions"
					:busy="busy"
					:locale="locale"
					:messages="messages"
					@action="emit('action', $event)"
				/>
			</el-tab-pane>
			<el-tab-pane :label="`${text.attachments} (${attachments.length})`">
				<DocumentAttachments
					:attachments="attachments"
					:loading="busy"
					:can-add="canAddAttachment"
					:can-remove="canRemoveAttachment"
					:locale="locale"
					:messages="messages"
					@add="emit('addAttachment')"
					@download="emit('downloadAttachment', $event)"
					@remove="emit('removeAttachment', $event)"
				/>
			</el-tab-pane>
			<el-tab-pane :label="`${text.comments} (${comments.length})`">
				<DocumentComments
					:comments="comments"
					:busy="busy"
					:can-comment="canComment"
					:locale="locale"
					:messages="messages"
					@comment="emit('comment', $event)"
				/>
			</el-tab-pane>
			<el-tab-pane :label="text.timeline">
				<DocumentTimeline :events="timeline" :loading="busy" :locale="locale" :messages="messages" />
			</el-tab-pane>
		</el-tabs>
	</article>
</template>

<style scoped>
.document-detail {
	display: grid;
	gap: var(--layout-gap);
	min-width: 0;
}

.document-detail__header,
.document-detail__eyebrow,
.document-detail__commands {
	display: flex;
	align-items: center;
	gap: 10px;
}

.document-detail__header {
	justify-content: space-between;
	padding-bottom: 14px;
	border-bottom: 1px solid var(--color-border);
}

.document-detail__identity h2,
.document-detail__identity p {
	margin: 0;
}

.document-detail__identity h2 {
	margin-top: 5px;
	font-size: 1.45rem;
}

.document-detail__identity p,
.document-detail__eyebrow {
	color: var(--color-text-muted);
	font-size: 0.86rem;
}

.document-detail__metadata,
.document-detail__fields,
.document-detail__line-section,
.document-detail__tabs {
	min-width: 0;
	padding: var(--content-padding);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.document-detail__metadata dl,
.document-detail__fields {
	display: grid;
	grid-template-columns: repeat(4, minmax(0, 1fr));
	gap: 14px;
	margin: 0;
}

.document-detail__metadata dl div,
.document-detail__field {
	display: grid;
	gap: 3px;
}

.document-detail dt {
	color: var(--color-text-muted);
	font-size: 0.78rem;
}

.document-detail dd {
	min-width: 0;
	margin: 0;
	word-break: break-word;
	white-space: pre-wrap;
}

.document-detail__redacted {
	color: var(--color-warning-text);
	font-style: italic;
}

.document-detail__code {
	font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
	font-size: 0.82rem;
}

.document-detail__line-section h3 {
	margin: 0 0 12px;
	font-size: 1rem;
}

.document-detail__table-wrap {
	width: 100%;
	max-width: 100%;
	min-width: 0;
	overflow-x: auto;
}

.document-detail table {
	width: 100%;
	min-width: 640px;
	border-collapse: collapse;
}

.document-detail th,
.document-detail td {
	padding: 9px 10px;
	border-bottom: 1px solid var(--color-border);
	text-align: left;
	vertical-align: top;
}

.document-detail th {
	color: var(--color-text-muted);
	font-size: 0.8rem;
	font-weight: 600;
}

.document-detail tbody tr:last-child td {
	border-bottom: 0;
}

@media (max-width: 880px) {
	.document-detail__metadata dl,
	.document-detail__fields {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}
}

@media (max-width: 560px) {
	.document-detail__header {
		align-items: flex-start;
		flex-direction: column;
	}

	.document-detail__commands {
		width: 100%;
	}

	.document-detail__commands :deep(.el-button) {
		flex: 1;
	}

	.document-detail__metadata dl,
	.document-detail__fields {
		grid-template-columns: 1fr;
	}
}

@media print {
	.document-detail__commands,
	.document-detail__tabs {
		display: none;
	}

	.document-detail__metadata,
	.document-detail__fields,
	.document-detail__line-section {
		break-inside: avoid;
		border-color: var(--color-border-strong);
	}
}
</style>
