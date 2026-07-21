<script setup lang="ts">
import { computed, h, useSlots } from "vue";

import { localizeKnownError, useI18n } from "../../i18n";
import StateBlock from "./StateBlock.vue";

export type DataTableColumn<T extends Record<string, unknown> = Record<string, unknown>> = {
	key: keyof T & string;
	label: string;
	width?: string | number;
	minWidth?: string | number;
	align?: "left" | "center" | "right";
	fixed?: boolean | "left" | "right";
	formatter?: (row: T) => string | number;
};

const props = withDefaults(defineProps<{
	rows: Record<string, unknown>[];
	columns: DataTableColumn[];
	rowKey?: string;
	loading?: boolean;
	error?: string;
	forbidden?: boolean;
	emptyText?: string;
	height?: string | number;
	stripe?: boolean;
	border?: boolean;
	ariaLabel?: string;
	virtualized?: boolean;
}>(), {
	rowKey: "id",
	loading: false,
	error: "",
	forbidden: false,
	emptyText: "",
	height: undefined,
	stripe: true,
	border: false,
	ariaLabel: "",
	virtualized: false
});

const hasRows = computed(() => props.rows.length > 0);
const slots = useSlots();
const { t } = useI18n();
const localizedError = computed(() => localizeKnownError(props.error));
const tableLabel = computed(() => props.ariaLabel || props.emptyText || t("state.noData"));
const virtualHeight = computed(() => typeof props.height === "number" ? props.height : Number.parseInt(String(props.height || "520"), 10) || 520);
const virtualColumns = computed(() => {
	const columns = props.columns.map((column) => ({
		key: column.key,
		dataKey: column.key,
		title: column.label,
		width: numericColumnWidth(column),
		align: column.align,
		fixed: column.fixed,
		cellRenderer: ({ rowData }: { rowData: Record<string, unknown> }) => {
			const slot = slots[`cell-${column.key}`];
			return slot
				? h("div", { class: "data-table__virtual-cell" }, slot({ row: rowData, value: rowData[column.key] }))
				: h("span", String(cellText(rowData, column)));
		}
	}));
	if (slots.actions) {
		columns.push({
			key: "__actions",
			dataKey: "__actions",
			title: t("common.actions"),
			width: 132,
			align: "right",
			fixed: "right",
			cellRenderer: ({ rowData }: { rowData: Record<string, unknown> }) => h("div", { class: "data-table__actions" }, slots.actions?.({ row: rowData }))
		});
	}
	return columns;
});

function cellText(row: Record<string, unknown>, column: DataTableColumn): string | number {
	if (column.formatter) {
		return column.formatter(row);
	}
	const value = row[column.key];
	if (value === null || value === undefined || value === "") {
		return "-";
	}
	if (typeof value === "string" || typeof value === "number") {
		return value;
	}
	return String(value);
}

function numericColumnWidth(column: DataTableColumn): number {
	const value = column.width ?? column.minWidth ?? 160;
	if (typeof value === "number") return value;
	const parsed = Number.parseInt(value, 10);
	return Number.isFinite(parsed) ? parsed : 160;
}
</script>

<template>
	<section class="data-table">
		<StateBlock v-if="forbidden" type="forbidden" :title="t('state.noPermission')" :description="t('state.noDataPermission')" />
		<StateBlock v-else-if="error" type="error" :title="t('state.requestFailed')" :description="localizedError" />
		<div v-else-if="virtualized && hasRows" v-loading="loading" class="data-table__virtual" :style="{ height: `${virtualHeight}px` }">
			<el-auto-resizer>
				<template #default="{ height: availableHeight, width: availableWidth }">
					<el-table-v2
						:columns="virtualColumns"
						:data="rows"
						:width="availableWidth"
						:height="availableHeight"
						:row-key="rowKey"
						fixed
						:aria-label="tableLabel"
					/>
				</template>
			</el-auto-resizer>
		</div>
		<el-table
			v-else
			:data="rows"
			:row-key="rowKey"
			:loading="loading"
			:stripe="stripe"
				:border="border"
				:height="height"
				:aria-label="tableLabel"
				class="data-table__el"
		>
			<template #empty>
				<StateBlock type="empty" :description="emptyText || t('state.noData')" />
			</template>
			<el-table-column
				v-for="column in columns"
				:key="column.key"
				:prop="column.key"
				:label="column.label"
				:width="column.width"
				:min-width="column.minWidth"
				:align="column.align"
				:fixed="column.fixed"
				show-overflow-tooltip
			>
				<template #default="{ row }">
					<slot :name="`cell-${column.key}`" :row="row" :value="row[column.key]">
						{{ cellText(row, column) }}
					</slot>
				</template>
			</el-table-column>
			<el-table-column v-if="$slots.actions" fixed="right" :label="t('common.actions')" width="132" align="right">
				<template #default="{ row }">
					<div class="data-table__actions">
						<slot name="actions" :row="row" />
					</div>
				</template>
			</el-table-column>
		</el-table>
		<div v-if="hasRows && $slots.pagination" class="data-table__pagination">
			<slot name="pagination" />
		</div>
	</section>
</template>

<style scoped>
.data-table {
	display: grid;
	gap: 10px;
	min-width: 0;
	max-width: 100%;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
	overflow-x: auto;
	overflow-y: hidden;
}

.data-table__el {
	width: 100%;
}

.data-table__virtual {
	width: 100%;
	min-width: 0;
}

:deep(.data-table__virtual-cell) {
	display: flex;
	align-items: center;
	min-width: 0;
	height: 100%;
	overflow: hidden;
}

.data-table__actions {
	display: inline-flex;
	align-items: center;
	justify-content: flex-end;
	gap: 6px;
	flex-wrap: wrap;
}

.data-table__pagination {
	display: flex;
	justify-content: flex-end;
	padding: 0 12px 12px;
}

:deep(.state-block) {
	border: 0;
	border-radius: 0;
}
</style>
