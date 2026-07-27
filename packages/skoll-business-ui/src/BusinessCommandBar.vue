<script setup lang="ts">
import { computed } from "vue";

import type { BusinessCommand, BusinessLocale } from "./types";
import { businessMessages } from "./messages";

const props = withDefaults(defineProps<{
	commands: readonly BusinessCommand[];
	locale?: BusinessLocale;
}>(), {
	locale: "zh-CN"
});

const emit = defineEmits<{ (event: "command", command: BusinessCommand): void }>();
const copy = computed(() => businessMessages(props.locale));
</script>

<template>
	<div class="business-command-bar" role="toolbar">
		<template v-for="command in commands" :key="command.id">
			<el-popconfirm
				v-if="command.destructive"
				:title="command.confirmation || copy.states.destructive.description"
				:confirm-button-text="copy.confirm"
				:cancel-button-text="copy.cancel"
				width="280"
				@confirm="emit('command', command)"
			>
				<template #reference>
					<el-button
						:type="command.tone || 'danger'"
						:icon="command.icon"
						:disabled="command.disabled"
						:loading="command.loading"
					>{{ command.label }}</el-button>
				</template>
			</el-popconfirm>
			<el-button
				v-else
				:type="command.tone"
				:icon="command.icon"
				:disabled="command.disabled"
				:loading="command.loading"
				@click="emit('command', command)"
			>{{ command.label }}</el-button>
		</template>
	</div>
</template>

<style scoped>
.business-command-bar { display: flex; align-items: center; justify-content: flex-end; gap: 8px; flex-wrap: wrap; }
.business-command-bar :deep(.el-button + .el-button) { margin-left: 0; }
@media (max-width: 560px) {
	.business-command-bar { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); width: 100%; }
	.business-command-bar :deep(.el-button) { width: 100%; }
}
</style>
