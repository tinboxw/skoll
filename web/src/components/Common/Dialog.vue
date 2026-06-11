<script setup lang="ts">
defineProps<{
	title: string;
	open: boolean;
	width?: string;
}>();

const emit = defineEmits<{
	(event: "close"): void;
}>();
</script>

<template>
	<el-dialog
		:model-value="open"
		:title="title"
		:width="width || '520px'"
		destroy-on-close
		@close="emit('close')"
	>
		<slot />
	</el-dialog>
</template>
