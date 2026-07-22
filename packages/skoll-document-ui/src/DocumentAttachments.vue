<script setup lang="ts">
import { Download, Paperclip, Plus, Trash2 } from "lucide-vue-next";
import { computed } from "vue";

import { formatDocumentDate, formatFileSize } from "./format";
import { documentKitMessages } from "./messages";
import type { DocumentAttachment, DocumentKitMessages, DocumentLocale } from "./types";

const props = withDefaults(defineProps<{
	attachments?: DocumentAttachment[];
	loading?: boolean;
	canAdd?: boolean;
	canRemove?: boolean;
	locale?: DocumentLocale;
	messages?: Partial<DocumentKitMessages>;
}>(), {
	attachments: () => [],
	loading: false,
	canAdd: false,
	canRemove: false,
	locale: "zh-CN",
	messages: () => ({})
});

const emit = defineEmits<{
	(e: "add"): void;
	(e: "download" | "remove", value: DocumentAttachment): void;
}>();

const text = computed(() => documentKitMessages(props.locale, props.messages));
</script>

<template>
	<section class="document-attachments" :aria-label="text.attachments" v-loading="loading">
		<header>
			<h3><Paperclip :size="17" />{{ text.attachments }}</h3>
			<el-button v-if="canAdd" @click="emit('add')"><Plus :size="16" />{{ text.addAttachment }}</el-button>
		</header>
		<el-empty v-if="!loading && attachments.length === 0" :description="text.noAttachments" :image-size="72" />
		<ul v-else class="document-attachments__list">
			<li v-for="attachment in attachments" :key="attachment.id" :class="{ 'is-removed': attachment.removedAt }">
				<div class="document-attachments__file">
					<strong>{{ attachment.file.name }}</strong>
					<span>{{ formatFileSize(attachment.file.size) }} · {{ attachment.addedBy.name || attachment.addedBy.id }} · {{ formatDocumentDate(attachment.addedAt, locale) }}</span>
				</div>
				<el-tag v-if="attachment.removedAt" type="info">{{ text.fileRemoved }}</el-tag>
				<div v-else class="document-attachments__actions">
				<el-tooltip :content="text.downloadAttachment">
					<el-button link :aria-label="text.downloadAttachment" @click="emit('download', attachment)"><Download :size="17" /></el-button>
				</el-tooltip>
				<el-tooltip v-if="canRemove" :content="text.removeAttachment">
					<el-button link type="danger" :aria-label="text.removeAttachment" @click="emit('remove', attachment)"><Trash2 :size="17" /></el-button>
					</el-tooltip>
				</div>
			</li>
		</ul>
	</section>
</template>

<style scoped>
.document-attachments,
.document-attachments__list {
	display: grid;
	gap: 10px;
}

.document-attachments header,
.document-attachments li,
.document-attachments h3,
.document-attachments__actions {
	display: flex;
	align-items: center;
	gap: 8px;
}

.document-attachments header,
.document-attachments li {
	justify-content: space-between;
}

.document-attachments h3 {
	margin: 0;
	font-size: 1rem;
}

.document-attachments__list {
	margin: 0;
	padding: 0;
	list-style: none;
}

.document-attachments li {
	padding: 10px 12px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-sm);
	background: var(--color-surface-soft);
}

.document-attachments li.is-removed {
	opacity: 0.68;
}

.document-attachments__file {
	display: grid;
	gap: 2px;
	min-width: 0;
}

.document-attachments__file strong,
.document-attachments__file span {
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.document-attachments__file span {
	color: var(--color-text-muted);
	font-size: 0.8rem;
}
</style>
