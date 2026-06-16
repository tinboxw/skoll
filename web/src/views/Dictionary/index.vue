<script setup lang="ts">
import { computed, onMounted, ref } from "vue";

import { confirmAction } from "../../composables/useConfirmAction";
import { useI18n } from "../../i18n";
import { type ApiResponse, apiGet, apiPut } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";

type DictionaryItem = {
	label: string;
	value: string;
	status: "enabled" | "disabled";
	order: number;
};

type DictionaryType = {
	type: string;
	name: string;
	description?: string;
	status: "enabled" | "disabled";
	order: number;
	items: DictionaryItem[];
};

type DictionaryPayload = {
	items?: DictionaryType[];
	customized?: boolean;
};

const { t } = useI18n();

const loading = ref(false);
const saving = ref(false);
const error = ref("");
const success = ref("");
const search = ref("");
const customized = ref(false);
const dictionaries = ref<DictionaryType[]>([]);
const selectedType = ref("");

const selectedDictionary = computed(() => dictionaries.value.find((item) => item.type === selectedType.value) ?? null);
const filteredDictionaries = computed(() => {
	const keyword = search.value.trim().toLowerCase();
	if (!keyword) {
		return dictionaries.value;
	}
	return dictionaries.value.filter((item) => {
		return item.type.toLowerCase().includes(keyword) || item.name.toLowerCase().includes(keyword);
	});
});

function normalizeStatus(value: unknown): "enabled" | "disabled" {
	return value === "disabled" ? "disabled" : "enabled";
}

function normalizeDictionaries(items: DictionaryType[]): DictionaryType[] {
	return items
		.map((item) => ({
			type: String(item.type || "").trim(),
			name: String(item.name || "").trim(),
			description: String(item.description || "").trim(),
			status: normalizeStatus(item.status),
			order: Number.isFinite(item.order) ? Number(item.order) : 0,
			items: Array.isArray(item.items)
				? item.items.map((child) => ({
					label: String(child.label || "").trim(),
					value: String(child.value || "").trim(),
					status: normalizeStatus(child.status),
					order: Number.isFinite(child.order) ? Number(child.order) : 0
				})).filter((child) => child.label !== "" && child.value !== "")
				: []
		}))
		.filter((item) => item.type !== "" && item.name !== "")
		.sort((a, b) => a.order - b.order || a.type.localeCompare(b.type));
}

async function loadDictionaries(): Promise<void> {
	loading.value = true;
	error.value = "";
	try {
		const payload = await apiGet<ApiResponse<DictionaryPayload>>("/v1/system/dictionaries");
		dictionaries.value = normalizeDictionaries(payload.data?.items ?? []);
		customized.value = Boolean(payload.data?.customized);
		if (!selectedType.value || !dictionaries.value.some((item) => item.type === selectedType.value)) {
			selectedType.value = dictionaries.value[0]?.type ?? "";
		}
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		loading.value = false;
	}
}

async function saveDictionaries(): Promise<void> {
	saving.value = true;
	error.value = "";
	success.value = "";
	try {
		const payload = await apiPut<ApiResponse<DictionaryPayload>>("/v1/system/dictionaries", {
			items: normalizeDictionaries(dictionaries.value)
		});
		dictionaries.value = normalizeDictionaries(payload.data?.items ?? []);
		customized.value = Boolean(payload.data?.customized);
		success.value = t("dictionary.saveDone");
		if (!selectedType.value || !dictionaries.value.some((item) => item.type === selectedType.value)) {
			selectedType.value = dictionaries.value[0]?.type ?? "";
		}
	} catch (e) {
		error.value = toErrorMessage(e);
	} finally {
		saving.value = false;
	}
}

function addDictionaryType(): void {
	const nextIndex = dictionaries.value.length + 1;
	const type = `custom.type.${nextIndex}`;
	dictionaries.value.push({
		type,
		name: `Custom Type ${nextIndex}`,
		description: "",
		status: "enabled",
		order: nextIndex * 10,
		items: []
	});
	selectedType.value = type;
}

function selectDictionary(row: DictionaryType | null): void {
	if (row?.type) {
		selectedType.value = row.type;
	}
}

async function removeDictionaryType(type: string): Promise<void> {
	const confirmed = await confirmAction({
		title: t("common.confirm"),
		message: t("dictionary.removeTypeConfirm"),
		confirmText: t("common.delete"),
		cancelText: t("common.cancel"),
		danger: true
	});
	if (!confirmed) {
		return;
	}
	dictionaries.value = dictionaries.value.filter((item) => item.type !== type);
	selectedType.value = dictionaries.value[0]?.type ?? "";
}

function addDictionaryItem(): void {
	if (!selectedDictionary.value) {
		return;
	}
	const nextIndex = selectedDictionary.value.items.length + 1;
	selectedDictionary.value.items.push({
		label: `Item ${nextIndex}`,
		value: `item_${nextIndex}`,
		status: "enabled",
		order: nextIndex * 10
	});
}

async function removeDictionaryItem(index: number): Promise<void> {
	if (!selectedDictionary.value) {
		return;
	}
	const confirmed = await confirmAction({
		title: t("common.confirm"),
		message: t("dictionary.removeItemConfirm"),
		confirmText: t("common.delete"),
		cancelText: t("common.cancel"),
		danger: true
	});
	if (!confirmed) {
		return;
	}
	selectedDictionary.value.items.splice(index, 1);
}

onMounted(() => {
	void loadDictionaries();
});
</script>

<template>
	<section class="dictionary-page">
		<header class="page-header">
			<div>
				<h2>{{ t("page.dictionaries") }}</h2>
				<p>{{ t("dictionary.desc") }}</p>
			</div>
			<div class="header-actions">
				<el-tag :type="customized ? 'success' : 'info'" effect="plain">
					{{ customized ? t("menu.editor.customized") : t("menu.editor.defaultSource") }}
				</el-tag>
				<el-button :loading="loading" :disabled="saving" @click="loadDictionaries">{{ t("common.refresh") }}</el-button>
				<el-button type="primary" :loading="saving" :disabled="loading" @click="saveDictionaries">{{ t("common.save") }}</el-button>
			</div>
		</header>

		<el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
		<el-alert v-if="success" :title="success" type="success" show-icon :closable="false" />

		<div class="workbench">
			<section class="panel type-panel">
				<div class="section-header">
					<h3>{{ t("dictionary.typeList") }}</h3>
					<el-button type="primary" plain size="small" @click="addDictionaryType">{{ t("dictionary.addType") }}</el-button>
				</div>
				<el-input v-model="search" clearable :placeholder="t('settings.searchPlaceholder')" />
				<el-table
					v-loading="loading"
					:data="filteredDictionaries"
					border
					row-key="type"
					highlight-current-row
					:empty-text="t('common.empty')"
					@current-change="selectDictionary"
				>
					<el-table-column prop="type" :label="t('dictionary.type')" min-width="150" show-overflow-tooltip />
					<el-table-column prop="name" :label="t('dictionary.name')" min-width="140" show-overflow-tooltip />
					<el-table-column :label="t('table.status')" width="100">
						<template #default="{ row }">
							<el-tag :type="row.status === 'enabled' ? 'success' : 'info'" effect="plain">
								{{ row.status === "enabled" ? t("dictionary.enabled") : t("dictionary.disabled") }}
							</el-tag>
						</template>
					</el-table-column>
					<el-table-column :label="t('table.actions')" width="100" fixed="right">
						<template #default="{ row }">
							<el-button link type="danger" :disabled="saving" @click.stop="removeDictionaryType(row.type)">{{ t("common.delete") }}</el-button>
						</template>
					</el-table-column>
				</el-table>
			</section>

			<section class="panel detail-panel">
				<el-empty v-if="!selectedDictionary" :description="t('common.empty')" />
				<template v-else>
					<div class="section-header">
						<h3>{{ t("dictionary.itemList") }}</h3>
						<el-button type="primary" plain size="small" @click="addDictionaryItem">{{ t("dictionary.addItem") }}</el-button>
					</div>
					<el-form label-position="top" class="type-form">
						<el-form-item :label="t('dictionary.type')" required>
							<el-input v-model="selectedDictionary.type" :placeholder="t('dictionary.typePlaceholder')" />
						</el-form-item>
						<el-form-item :label="t('dictionary.name')" required>
							<el-input v-model="selectedDictionary.name" />
						</el-form-item>
						<el-form-item :label="t('table.status')">
							<el-select v-model="selectedDictionary.status">
								<el-option :label="t('dictionary.enabled')" value="enabled" />
								<el-option :label="t('dictionary.disabled')" value="disabled" />
							</el-select>
						</el-form-item>
						<el-form-item :label="t('menu.editor.order')">
							<el-input-number v-model="selectedDictionary.order" :step="10" />
						</el-form-item>
						<el-form-item class="full-row" :label="t('dictionary.description')">
							<el-input v-model="selectedDictionary.description" type="textarea" :rows="2" />
						</el-form-item>
					</el-form>

					<el-table :data="selectedDictionary.items" border row-key="value" :empty-text="t('common.empty')">
						<el-table-column :label="t('dictionary.itemLabel')" min-width="160">
							<template #default="{ row }">
								<el-input v-model="row.label" />
							</template>
						</el-table-column>
						<el-table-column :label="t('dictionary.itemValue')" min-width="160">
							<template #default="{ row }">
								<el-input v-model="row.value" />
							</template>
						</el-table-column>
						<el-table-column :label="t('table.status')" width="130">
							<template #default="{ row }">
								<el-select v-model="row.status">
									<el-option :label="t('dictionary.enabled')" value="enabled" />
									<el-option :label="t('dictionary.disabled')" value="disabled" />
								</el-select>
							</template>
						</el-table-column>
						<el-table-column :label="t('menu.editor.order')" width="120">
							<template #default="{ row }">
								<el-input-number v-model="row.order" :step="10" controls-position="right" />
							</template>
						</el-table-column>
						<el-table-column :label="t('table.actions')" width="90" fixed="right">
							<template #default="{ $index }">
								<el-button link type="danger" @click="removeDictionaryItem($index)">{{ t("common.delete") }}</el-button>
							</template>
						</el-table-column>
					</el-table>
				</template>
			</section>
		</div>
	</section>
</template>

<style scoped>
.dictionary-page {
	display: grid;
	gap: 14px;
}

.page-header,
.section-header {
	display: flex;
	align-items: flex-start;
	justify-content: space-between;
	gap: 16px;
}

.page-header h2,
.section-header h3 {
	margin: 0;
}

.page-header h2 {
	font-size: 1.35rem;
}

.page-header p {
	margin: 6px 0 0;
	color: var(--color-text-muted);
}

.header-actions {
	display: flex;
	align-items: center;
	gap: 8px;
}

.workbench {
	display: grid;
	grid-template-columns: minmax(360px, 0.9fr) minmax(0, 1.4fr);
	gap: 14px;
}

.panel {
	display: grid;
	align-content: start;
	gap: 12px;
	padding: 16px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.type-form {
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: 4px 12px;
}

.full-row {
	grid-column: 1 / -1;
}

@media (max-width: 1080px) {
	.workbench,
	.type-form,
	.page-header,
	.section-header,
	.header-actions {
		display: grid;
		grid-template-columns: 1fr;
	}

	.full-row {
		grid-column: auto;
	}
}
</style>
