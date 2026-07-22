<script setup lang="ts">
import { computed } from "vue";

import { formatDocumentDate } from "./format";
import { documentKitMessages } from "./messages";
import type { DocumentKitMessages, DocumentLocale, DocumentTimelineEvent } from "./types";

const props = withDefaults(defineProps<{
	events?: DocumentTimelineEvent[];
	loading?: boolean;
	locale?: DocumentLocale;
	messages?: Partial<DocumentKitMessages>;
}>(), {
	events: () => [],
	loading: false,
	locale: "zh-CN",
	messages: () => ({})
});

const text = computed(() => documentKitMessages(props.locale, props.messages));

function eventLabel(event: DocumentTimelineEvent): string {
	if (event.kind === "action") return event.action || text.value.workflow;
	if (event.kind === "attachment_added") return `${text.value.attachments} +`;
	if (event.kind === "attachment_removed") return `${text.value.attachments} -`;
	return text.value.comments;
}
</script>

<template>
	<section class="document-timeline" :aria-label="text.timeline" v-loading="loading">
		<h3>{{ text.timeline }}</h3>
		<el-empty v-if="!loading && events.length === 0" :description="text.noTimeline" :image-size="72" />
		<el-timeline v-else>
			<el-timeline-item
				v-for="event in events"
				:key="event.id"
				:timestamp="formatDocumentDate(event.occurredAt, locale)"
				:color="event.kind === 'action' ? 'var(--color-primary)' : 'var(--color-info)'"
			>
				<div class="document-timeline__event">
					<strong>{{ eventLabel(event) }}</strong>
					<span>{{ event.actor.name || event.actor.id }}</span>
				</div>
			</el-timeline-item>
		</el-timeline>
	</section>
</template>

<style scoped>
.document-timeline h3 {
	margin: 0 0 14px;
	font-size: 1rem;
}

.document-timeline__event {
	display: grid;
	gap: 2px;
}

.document-timeline__event span {
	color: var(--color-text-muted);
	font-size: 0.82rem;
}
</style>
