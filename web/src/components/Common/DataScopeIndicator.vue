<script setup lang="ts">
import { computed } from "vue";
import { ShieldCheck } from "lucide-vue-next";

import { useI18n } from "../../i18n";
import type { AuthorizedDataScopeDecision } from "../../permissions/data-scope";

const props = withDefaults(defineProps<{
	decision?: AuthorizedDataScopeDecision | null;
	loading?: boolean;
	error?: string;
}>(), {
	decision: null,
	loading: false,
	error: ""
});

const emit = defineEmits<{ retry: [] }>();
const { t } = useI18n();
const scopeLabel = computed(() => props.decision ? t(`dataScope.${props.decision.scope}`) : "");
const scopeDescription = computed(() => props.decision ? t(`dataScope.${props.decision.scope}.description`) : "");
const targetIDs = computed(() => props.decision?.scope === "self" ? props.decision.userIds : props.decision?.departmentIds ?? []);
</script>

<template>
	<section class="scope-indicator" aria-live="polite" :aria-busy="loading ? 'true' : 'false'">
		<el-skeleton v-if="loading" :rows="1" animated />
		<div v-else-if="error" class="scope-error">
			<el-alert type="error" :title="t('dataScope.loadFailed')" :description="error" show-icon :closable="false" />
			<el-button link type="primary" @click="emit('retry')">{{ t("common.retry") }}</el-button>
		</div>
		<template v-else-if="decision">
			<ShieldCheck :size="20" aria-hidden="true" />
			<div class="scope-copy">
				<span>{{ t("dataScope.effective") }}</span>
				<strong>{{ scopeLabel }}</strong>
				<small>{{ scopeDescription }}</small>
			</div>
			<div v-if="targetIDs.length > 0" class="scope-targets" :aria-label="t('dataScope.authorizedTargets')">
				<el-tag v-for="id in targetIDs" :key="id" type="success" effect="plain">{{ id }}</el-tag>
			</div>
		</template>
	</section>
</template>

<style scoped>
.scope-indicator {
	display: grid;
	grid-template-columns: auto minmax(0, 1fr) auto;
	align-items: center;
	gap: 12px;
	min-height: 72px;
	padding: 12px 14px;
	border: 1px solid var(--el-color-success-light-5);
	border-radius: 8px;
	background: var(--el-color-success-light-9);
}

.scope-copy {
	display: grid;
	min-width: 0;
}

.scope-copy span,
.scope-copy small {
	color: var(--color-text-secondary);
	overflow-wrap: anywhere;
}

.scope-copy strong {
	font-size: 1rem;
}

.scope-targets {
	display: flex;
	justify-content: flex-end;
	gap: 6px;
	flex-wrap: wrap;
}

.scope-error {
	display: flex;
	grid-column: 1 / -1;
	align-items: center;
	gap: 8px;
}

.scope-error .el-alert {
	min-width: 0;
	flex: 1;
}

@media (max-width: 720px) {
	.scope-indicator {
		grid-template-columns: auto minmax(0, 1fr);
		align-items: flex-start;
	}

	.scope-targets {
		grid-column: 1 / -1;
		justify-content: flex-start;
	}

	.scope-targets :deep(.el-tag) {
		max-width: 100%;
		overflow-wrap: anywhere;
		white-space: normal;
	}

	.scope-error {
		align-items: stretch;
		flex-direction: column;
	}
}
</style>
