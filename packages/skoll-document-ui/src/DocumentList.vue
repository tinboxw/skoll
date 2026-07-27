<script setup lang="ts">
import { ChevronLeft, ChevronRight, Download, Eye, Plus, RefreshCw, Search } from "lucide-vue-next";
import { computed } from "vue";

import { documentStateTone, formatDocumentDate } from "./format";
import { documentKitMessages } from "./messages";
import type { DocumentKitMessages, DocumentLocale, DocumentSchema, DocumentSearchPage, DocumentSummary } from "./types";

const props = withDefaults(defineProps<{
	schema: DocumentSchema;
	page: DocumentSearchPage;
	query?: string;
	stateFilter?: string;
	loading?: boolean;
	error?: string;
	forbidden?: boolean;
	canCreate?: boolean;
	canExport?: boolean;
	canGoPrevious?: boolean;
	locale?: DocumentLocale;
	messages?: Partial<DocumentKitMessages>;
}>(), {
	query: "",
	stateFilter: "",
	loading: false,
	error: "",
	forbidden: false,
	canCreate: false,
	canExport: false,
	canGoPrevious: false,
	locale: "zh-CN",
	messages: () => ({})
});

const emit = defineEmits<{
	(e: "update:query", value: string): void;
	(e: "update:stateFilter", value: string): void;
	(e: "search" | "refresh" | "create" | "previous" | "next" | "export"): void;
	(e: "open", value: DocumentSummary): void;
}>();

const text = computed(() => documentKitMessages(props.locale, props.messages));
</script>

<template>
	<section class="document-list" :aria-label="schema.name">
		<el-form class="document-list__filters" inline @submit.prevent="emit('search')">
			<el-form-item :label="text.search">
				<el-input
					:model-value="query"
					:placeholder="text.searchPlaceholder"
					clearable
					@update:model-value="(value: string) => emit('update:query', value)"
				>
					<template #prefix><Search :size="16" aria-hidden="true" /></template>
				</el-input>
			</el-form-item>
			<el-form-item :label="text.state">
				<el-select
					:model-value="stateFilter"
					clearable
					@update:model-value="(value: string) => emit('update:stateFilter', value || '')"
				>
					<el-option value="" :label="text.allStates" />
					<el-option v-for="state in schema.states" :key="state.key" :value="state.key" :label="state.name" />
				</el-select>
			</el-form-item>
			<div class="document-list__commands">
				<el-button native-type="submit" type="primary"><Search :size="16" />{{ text.search }}</el-button>
				<el-tooltip :content="text.refresh">
					<el-button :aria-label="text.refresh" :loading="loading" @click="emit('refresh')"><RefreshCw :size="16" /></el-button>
				</el-tooltip>
				<el-button v-if="canExport" @click="emit('export')"><Download :size="16" />{{ text.export }}</el-button>
				<el-button v-if="canCreate" type="primary" @click="emit('create')"><Plus :size="16" />{{ text.create }}</el-button>
			</div>
		</el-form>

		<el-skeleton v-if="loading" :rows="6" animated />
		<el-alert v-else-if="forbidden" class="state-block--forbidden" :title="text.forbidden" type="warning" show-icon :closable="false" />
		<el-alert v-else-if="error" class="state-block--error" :title="text.requestFailed" :description="error" type="error" show-icon :closable="false" />
		<el-empty v-else-if="page.items.length === 0" :description="text.noData" />
		<div v-else class="document-list__table">
			<el-table :data="page.items" row-key="id" table-layout="fixed" :aria-label="schema.name">
				<el-table-column prop="number" :label="text.number" min-width="150" fixed="left" />
				<el-table-column prop="title" :label="text.title" min-width="220" show-overflow-tooltip />
				<el-table-column prop="state" :label="text.state" width="120">
					<template #default="{ row }: { row: DocumentSummary }">
						<el-tag :type="documentStateTone(row.state)" effect="light">
							{{ schema.states.find((state) => state.key === row.state)?.name || row.state }}
						</el-tag>
					</template>
				</el-table-column>
				<el-table-column prop="updatedAt" :label="text.updatedAt" min-width="170">
					<template #default="{ row }: { row: DocumentSummary }">{{ formatDocumentDate(row.updatedAt, locale) }}</template>
				</el-table-column>
				<el-table-column prop="createdBy" :label="text.createdBy" min-width="140" show-overflow-tooltip />
				<el-table-column :label="text.actions" width="72" fixed="right">
					<template #default="{ row }: { row: DocumentSummary }">
						<el-tooltip :content="text.open">
							<el-button link type="primary" :aria-label="text.open" @click="emit('open', row)"><Eye :size="17" /></el-button>
						</el-tooltip>
					</template>
				</el-table-column>
			</el-table>
		</div>

		<footer v-if="!loading && !forbidden && !error && (page.items.length || canGoPrevious || page.hasMore)" class="document-list__paging">
			<el-button :disabled="!canGoPrevious" @click="emit('previous')"><ChevronLeft :size="16" />{{ text.previous }}</el-button>
			<el-button :disabled="!page.hasMore" @click="emit('next')">{{ text.next }}<ChevronRight :size="16" /></el-button>
		</footer>
	</section>
</template>

<style scoped>
.document-list {
	display: grid;
	gap: var(--layout-gap);
	min-width: 0;
}

.document-list__filters {
	display: grid;
	grid-template-columns: minmax(220px, 1fr) minmax(180px, 0.45fr) auto;
	gap: 10px;
	align-items: end;
}

.document-list__filters :deep(.el-form-item) {
	min-width: 0;
	margin: 0;
}

.document-list__filters :deep(.el-form-item__content),
.document-list__filters :deep(.el-select) {
	width: 100%;
}

.document-list__commands,
.document-list__paging {
	display: flex;
	gap: 8px;
	align-items: center;
	flex-wrap: wrap;
}

.document-list__table {
	width: 100%;
	max-width: 100%;
	min-width: 0;
	overflow: hidden;
}

.document-list__paging {
	justify-content: flex-end;
}

@media (max-width: 760px) {
	.document-list__filters {
		grid-template-columns: 1fr;
	}

	.document-list__commands {
		width: 100%;
	}

	.document-list__commands :deep(.el-button) {
		flex: 1;
	}
}
</style>
