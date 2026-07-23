<script setup lang="ts">
import { computed } from "vue";
import { MoreHorizontal, Pin, X } from "lucide-vue-next";
import { useRoute } from "vue-router";

import { useI18n } from "../../i18n";
import { resolveSystemNavigationLabel } from "../../navigation/system-label";
import type { PinnedTab } from "../../stores/tabs";

const MAX_VISIBLE_TABS = 8;

type WorkspaceTab = PinnedTab & { kind: "pinned" | "recent" };

const props = defineProps<{
	pinnedItems: PinnedTab[];
	recentItems: PinnedTab[];
}>();

const emit = defineEmits<{
	(event: "open", path: string): void;
	(event: "unpin", path: string): void;
	(event: "removeRecent", path: string): void;
	(event: "pin", path: string): void;
}>();

const route = useRoute();
const { t } = useI18n();
const mergedItems = computed<WorkspaceTab[]>(() => {
	const localized = (item: PinnedTab): PinnedTab => ({
		...item,
		label: resolveSystemNavigationLabel(item.path, item.id, t, item.label)
	});
	const pinned = props.pinnedItems.map((item) => ({ ...localized(item), kind: "pinned" as const }));
	const pinnedPaths = new Set(props.pinnedItems.map((item) => item.path));
	const recent = props.recentItems.filter((item) => !pinnedPaths.has(item.path)).map((item) => ({ ...localized(item), kind: "recent" as const }));
	return [...pinned, ...recent];
});
const visibleItems = computed(() => mergedItems.value.slice(0, MAX_VISIBLE_TABS));
const overflowItems = computed(() => mergedItems.value.slice(MAX_VISIBLE_TABS));
const hasItems = computed(() => mergedItems.value.length > 0);

function isActive(path: string): boolean {
	return route.path === path || route.path.startsWith(path + "/");
}

function closeTab(item: WorkspaceTab): void {
	if (item.kind === "pinned") {
		emit("unpin", item.path);
		return;
	}
	emit("removeRecent", item.path);
}
</script>

<template>
	<section v-if="hasItems" class="tabs-wrap" :aria-label="t('tabs.title')">
		<span class="tabs-title">{{ t("tabs.title") }}</span>
		<div class="tabs-scroll">
			<div v-for="item in visibleItems" :key="item.path" class="tab-group" :class="{ active: isActive(item.path) }">
				<el-button text class="tab-main" @click="emit('open', item.path)">
					<el-tag size="small" effect="plain" :type="item.kind === 'pinned' ? 'success' : 'info'">
						{{ item.kind === "pinned" ? t("tabs.pinned") : t("tabs.recent") }}
					</el-tag>
					<span class="tab-label">{{ item.label }}</span>
				</el-button>
				<el-tooltip v-if="item.kind === 'recent'" :content="t('tabs.pin')">
					<el-button text circle size="small" class="tab-pin" :icon="Pin" :aria-label="`${t('tabs.pin')} ${item.label}`" @click="emit('pin', item.path)" />
				</el-tooltip>
				<el-tooltip :content="t('tabs.close')">
					<el-button text circle size="small" class="tab-close" :icon="X" :aria-label="`${t('tabs.close')} ${item.label}`" @click="closeTab(item)" />
				</el-tooltip>
			</div>

			<el-dropdown v-if="overflowItems.length > 0" @command="(path: string) => emit('open', path)">
				<el-button :icon="MoreHorizontal">{{ t("tabs.more") }} ({{ overflowItems.length }})</el-button>
				<template #dropdown>
					<el-dropdown-menu>
						<el-dropdown-item v-for="item in overflowItems" :key="item.path" :command="item.path">
							{{ item.label }} · {{ item.kind === "pinned" ? t("tabs.pinned") : t("tabs.recent") }}
						</el-dropdown-item>
					</el-dropdown-menu>
				</template>
			</el-dropdown>
		</div>
	</section>
</template>

<style scoped>
.tabs-wrap {
	display: grid;
	grid-template-columns: auto minmax(0, 1fr);
	align-items: center;
	gap: 10px;
	margin: -4px 0 10px;
	padding: 7px 9px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

.tabs-title {
	color: var(--color-text-muted);
	font-size: 0.75rem;
	font-weight: 700;
	white-space: nowrap;
}

.tabs-scroll {
	display: flex;
	gap: 7px;
	overflow-x: auto;
}

.tab-group {
	display: inline-flex;
	align-items: center;
	flex-shrink: 0;
	max-width: 300px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-sm);
	background: var(--color-surface-soft);
}

.tab-group.active {
	border-color: var(--color-primary);
	box-shadow: inset 0 0 0 1px var(--color-primary);
}

.tab-main {
	min-width: 0;
	max-width: 220px;
}

.tab-label {
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

@media (max-width: 860px) {
	.tabs-wrap {
		grid-template-columns: 1fr;
	}
}
</style>
