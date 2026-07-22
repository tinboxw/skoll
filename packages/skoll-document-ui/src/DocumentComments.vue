<script setup lang="ts">
import { MessageSquare, Send } from "lucide-vue-next";
import { computed, ref } from "vue";

import { formatDocumentDate } from "./format";
import { documentKitMessages } from "./messages";
import type { DocumentComment, DocumentKitMessages, DocumentLocale } from "./types";

const props = withDefaults(defineProps<{
	comments?: DocumentComment[];
	loading?: boolean;
	busy?: boolean;
	canComment?: boolean;
	locale?: DocumentLocale;
	messages?: Partial<DocumentKitMessages>;
}>(), {
	comments: () => [],
	loading: false,
	busy: false,
	canComment: false,
	locale: "zh-CN",
	messages: () => ({})
});

const emit = defineEmits<{
	(e: "comment", value: string): void;
}>();

const text = computed(() => documentKitMessages(props.locale, props.messages));
const body = ref("");

function submit(): void {
	const value = body.value.trim();
	if (!value) return;
	emit("comment", value);
	body.value = "";
}
</script>

<template>
	<section class="document-comments" :aria-label="text.comments" v-loading="loading">
		<h3><MessageSquare :size="17" />{{ text.comments }}</h3>
		<el-empty v-if="!loading && comments.length === 0" :description="text.noComments" :image-size="72" />
		<ul v-else>
			<li v-for="comment in comments" :key="comment.id">
				<header><strong>{{ comment.author.name || comment.author.id }}</strong><time>{{ formatDocumentDate(comment.createdAt, locale) }}</time></header>
				<p>{{ comment.body }}</p>
			</li>
		</ul>
		<div v-if="canComment" class="document-comments__composer">
			<el-input v-model="body" type="textarea" :rows="3" maxlength="8000" show-word-limit :disabled="busy" />
			<el-button type="primary" :loading="busy" :disabled="!body.trim()" @click="submit"><Send :size="16" />{{ text.addComment }}</el-button>
		</div>
	</section>
</template>

<style scoped>
.document-comments,
.document-comments ul,
.document-comments__composer {
	display: grid;
	gap: 10px;
}

.document-comments h3,
.document-comments li header {
	display: flex;
	align-items: center;
	gap: 8px;
}

.document-comments h3 {
	margin: 0;
	font-size: 1rem;
}

.document-comments ul {
	margin: 0;
	padding: 0;
	list-style: none;
}

.document-comments li {
	padding: 10px 12px;
	border-left: 3px solid var(--color-border-strong);
	background: var(--color-surface-soft);
}

.document-comments li header {
	justify-content: space-between;
}

.document-comments time {
	color: var(--color-text-muted);
	font-size: 0.8rem;
}

.document-comments li p {
	margin: 6px 0 0;
	white-space: pre-wrap;
	word-break: break-word;
}

.document-comments__composer :deep(.el-button) {
	justify-self: end;
}
</style>
