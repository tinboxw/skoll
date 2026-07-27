<script setup lang="ts">
import { ChevronLeft, ChevronRight, Eye } from "lucide-vue-next";
import { computed } from "vue";

import BusinessStateBlock from "./BusinessState.vue";
import { businessMessages } from "./messages";
import type { BusinessListColumn, BusinessLocale, BusinessRecord, BusinessState } from "./types";

const props = withDefaults(defineProps<{
	items: readonly BusinessRecord[];
	columns: readonly BusinessListColumn[];
	rowKey?: string;
	state?: Extract<BusinessState, "ready" | "loading" | "empty" | "error" | "forbidden" | "conflict">;
	stateTitle?: string;
	stateDescription?: string;
	canPrevious?: boolean;
	canNext?: boolean;
	locale?: BusinessLocale;
}>(), {
	rowKey: "id",
	state: "ready",
	stateTitle: "",
	stateDescription: "",
	canPrevious: false,
	canNext: false,
	locale: "zh-CN"
});

const emit = defineEmits<{
	(event: "open", row: BusinessRecord): void;
	(event: "previous" | "next"): void;
}>();
const copy = computed(() => businessMessages(props.locale));
const visibleState = computed(() => props.state === "ready" && props.items.length === 0 ? "empty" : props.state);

function display(value: unknown): string {
	if (value === null || value === undefined || value === "") return "-";
	if (typeof value === "object") return JSON.stringify(value);
	return String(value);
}
</script>

<template>
	<section class="business-list">
		<BusinessStateBlock
			v-if="visibleState !== 'ready'"
			:state="visibleState"
			:title="stateTitle"
			:description="stateDescription"
			:locale="locale"
		>
			<template v-if="$slots.stateActions" #actions><slot name="stateActions" /></template>
		</BusinessStateBlock>
		<template v-else>
			<div class="business-list__desktop">
				<el-table :data="items" :row-key="rowKey" table-layout="fixed" stripe>
					<el-table-column
						v-for="column in columns"
						:key="column.key"
						:prop="column.key"
						:label="column.label"
						:width="column.width"
						:min-width="column.minWidth"
						:fixed="column.fixed"
						:show-overflow-tooltip="column.overflowTooltip !== false"
					>
						<template #default="{ row }: { row: BusinessRecord }">
							<slot :name="`cell-${column.key}`" :row="row" :value="row[column.key]">{{ display(row[column.key]) }}</slot>
						</template>
					</el-table-column>
					<el-table-column width="64" fixed="right">
						<template #default="{ row }: { row: BusinessRecord }">
							<slot name="actions" :row="row">
								<el-button link type="primary" :icon="Eye" :aria-label="copy.open" @click="emit('open', row)" />
							</slot>
						</template>
					</el-table-column>
				</el-table>
			</div>
			<div class="business-list__mobile">
				<article v-for="row in items" :key="String(row[rowKey])" class="business-list__record">
					<button type="button" @click="emit('open', row)">
						<strong>{{ display(row[columns[0]?.key]) }}</strong>
						<span v-if="columns[1]">{{ display(row[columns[1].key]) }}</span>
					</button>
					<dl>
						<div v-for="column in columns.slice(2)" :key="column.key">
							<dt>{{ column.label }}</dt><dd>{{ display(row[column.key]) }}</dd>
						</div>
					</dl>
					<footer v-if="$slots.actions"><slot name="actions" :row="row" /></footer>
				</article>
			</div>
		</template>
		<footer v-if="visibleState === 'ready' && (canPrevious || canNext)" class="business-list__paging">
			<el-button :icon="ChevronLeft" :disabled="!canPrevious" @click="emit('previous')">{{ copy.previous }}</el-button>
			<el-button :disabled="!canNext" @click="emit('next')">{{ copy.next }}<ChevronRight :size="16" /></el-button>
		</footer>
	</section>
</template>

<style scoped>
.business-list { display: grid; gap: 12px; min-width: 0; }
.business-list__desktop { min-width: 0; overflow: hidden; }
.business-list__mobile { display: none; }
.business-list__paging { display: flex; justify-content: flex-end; gap: 8px; }
@media (max-width: 720px) {
	.business-list__desktop { display: none; }
	.business-list__mobile { display: grid; gap: 8px; }
	.business-list__record {
		display: grid;
		gap: 10px;
		padding: 12px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm, 4px);
		background: var(--color-surface);
	}
	.business-list__record > button { display: grid; gap: 2px; padding: 0; border: 0; background: transparent; color: inherit; text-align: left; }
	.business-list__record > button span,
	.business-list__record dt { color: var(--color-text-muted); font-size: 0.78rem; }
	.business-list__record dl { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; margin: 0; }
	.business-list__record dl div { min-width: 0; }
	.business-list__record dt,
	.business-list__record dd { margin: 0; overflow-wrap: anywhere; }
	.business-list__record footer { display: flex; gap: 8px; flex-wrap: wrap; }
}
</style>
