<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";

import { useI18n } from "../../i18n";
import type { PinnedTab } from "../../stores/tabs";

const MAX_VISIBLE_TABS = 8;

type WorkspaceTab = PinnedTab & {
	kind: "pinned" | "recent";
};

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
	const pinned = props.pinnedItems.map((item) => ({ ...item, kind: "pinned" as const }));
	const pinnedPaths = new Set(props.pinnedItems.map((item) => item.path));
	const recent = props.recentItems
		.filter((item) => !pinnedPaths.has(item.path))
		.map((item) => ({ ...item, kind: "recent" as const }));
	return [...pinned, ...recent];
});

const visibleItems = computed(() => mergedItems.value.slice(0, MAX_VISIBLE_TABS));
const overflowItems = computed(() => mergedItems.value.slice(MAX_VISIBLE_TABS));
const hasItems = computed(() => mergedItems.value.length > 0);

function isActive(path: string): boolean {
	const current = route.path;
	return current === path || current.startsWith(path + "/");
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
		<div class="tabs-title">{{ t("tabs.title") }}</div>
		<div class="tabs-scroll">
			<button
				v-for="item in visibleItems"
				:key="item.path"
				type="button"
				class="tab"
				:class="{ active: isActive(item.path) }"
				@click="emit('open', item.path)"
			>
				<span class="dot" aria-hidden="true" />
				<span class="kind" :class="item.kind">{{ item.kind === "pinned" ? t("tabs.pinned") : t("tabs.recent") }}</span>
				<span class="label">{{ item.label }}</span>
				<span
					v-if="item.kind === 'recent'"
					class="pin"
					@click.stop="emit('pin', item.path)"
				>
					{{ t("tabs.pin") }}
				</span>
				<span class="close" @click.stop="closeTab(item)">×</span>
			</button>
			<details v-if="overflowItems.length > 0" class="overflow-menu">
				<summary>{{ t("tabs.more") }} ({{ overflowItems.length }})</summary>
				<div class="overflow-list">
					<button
						v-for="item in overflowItems"
						:key="`more-${item.path}`"
						type="button"
						class="overflow-item"
						@click="emit('open', item.path)"
					>
						<span class="overflow-label">{{ item.label }}</span>
						<span class="overflow-kind">{{ item.kind === "pinned" ? t("tabs.pinned") : t("tabs.recent") }}</span>
					</button>
				</div>
			</details>
		</div>
	</section>
</template>

<style scoped>
.tabs-wrap {
	display: grid;
	grid-template-columns: auto 1fr;
	align-items: center;
	gap: 10px;
	margin: -4px 0 10px;
	padding: 8px 10px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-lg);
	background: linear-gradient(160deg, var(--color-surface) 0%, var(--color-surface-soft) 100%);
}

.tabs-title {
	font-size: 12px;
	font-weight: 700;
	letter-spacing: 0.08em;
	text-transform: uppercase;
	color: var(--color-text-muted);
	white-space: nowrap;
}

.tabs-scroll {
	display: flex;
	gap: 8px;
	overflow-x: auto;
	padding-bottom: 2px;
}

.tab {
	display: flex;
	align-items: center;
	gap: 8px;
	padding: 7px 10px;
	border-radius: 10px;
	border: 1px solid var(--color-border);
	background: var(--color-surface);
	color: var(--color-text-muted);
	cursor: pointer;
	white-space: nowrap;
	max-width: 260px;
	flex-shrink: 0;
	transition: border-color 0.15s ease, box-shadow 0.15s ease, transform 0.15s ease;
}

.tab:hover {
	border-color: var(--color-border-strong);
	transform: translateY(-1px);
}

.label {
	overflow: hidden;
	text-overflow: ellipsis;
}

.kind {
	font-size: 10px;
	font-weight: 700;
	padding: 2px 6px;
	border-radius: 999px;
	background: var(--color-surface-soft);
	color: var(--color-text-muted);
	text-transform: uppercase;
	letter-spacing: 0.04em;
}

.kind.pinned {
	background: var(--color-tag-bg);
	color: var(--color-tag-text);
}

.tab.active {
	border-color: var(--color-primary);
	background: #ffffff;
	color: var(--color-tag-text);
	box-shadow: inset 0 0 0 1px var(--color-primary);
}

.dot {
	width: 8px;
	height: 8px;
	border-radius: 999px;
	background: var(--color-border-strong);
}

.tab.active .dot {
	background: var(--color-primary);
}

.close {
	display: inline-flex;
	align-items: center;
	justify-content: center;
	width: 18px;
	height: 18px;
	border-radius: 6px;
	font-size: 13px;
	font-weight: 700;
	line-height: 1;
	background: var(--color-surface);
	border: 1px solid var(--color-border-strong);
}

.pin {
	border: 1px solid var(--color-border-strong);
	background: var(--color-surface-soft);
	color: var(--color-tag-text);
	border-radius: 6px;
	padding: 2px 6px;
	font-size: 11px;
	font-weight: 600;
}

.overflow-menu {
	position: relative;
	flex-shrink: 0;
}

.overflow-menu summary {
	list-style: none;
	cursor: pointer;
	padding: 7px 10px;
	border-radius: 10px;
	border: 1px solid var(--color-border);
	background: var(--color-surface);
	font-size: 12px;
	font-weight: 600;
	color: var(--color-text-muted);
	white-space: nowrap;
}

.overflow-menu summary::-webkit-details-marker {
	display: none;
}

.overflow-list {
	position: absolute;
	right: 0;
	margin-top: 8px;
	padding: 8px;
	min-width: 260px;
	max-width: 320px;
	display: grid;
	gap: 6px;
	border-radius: var(--radius-md);
	border: 1px solid var(--color-border);
	background: var(--color-surface);
	box-shadow: 0 12px 24px -16px var(--color-shadow);
	z-index: 8;
}

.overflow-item {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 10px;
	border: 1px solid var(--color-border);
	background: var(--color-surface-soft);
	border-radius: 8px;
	padding: 6px 8px;
	cursor: pointer;
}

.overflow-label {
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.overflow-kind {
	font-size: 11px;
	color: var(--color-text-muted);
}

@media (max-width: 860px) {
	.tabs-wrap {
		grid-template-columns: 1fr;
		gap: 6px;
	}
}
</style>
