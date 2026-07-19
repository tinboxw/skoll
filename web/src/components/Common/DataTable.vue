<script setup lang="ts">
import { computed } from "vue";

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
}>(), {
	rowKey: "id",
	loading: false,
	error: "",
	forbidden: false,
	emptyText: "",
	height: undefined,
	stripe: true,
	border: false
});

const hasRows = computed(() => props.rows.length > 0);
const { t } = useI18n();
const localizedError = computed(() => localizeKnownError(props.error));

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
</script>

<template>
	<section class="data-table">
		<StateBlock v-if="forbidden" type="forbidden" :title="t('state.noPermission')" :description="t('state.noDataPermission')" />
		<StateBlock v-else-if="error" type="error" :title="t('state.requestFailed')" :description="localizedError" />
		<el-table
			v-else
			:data="rows"
			:row-key="rowKey"
			:loading="loading"
			:stripe="stripe"
			:border="border"
			:height="height"
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
			<el-table-column v-if="$slots.actions" fixed="right" label="" width="132" align="right">
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
