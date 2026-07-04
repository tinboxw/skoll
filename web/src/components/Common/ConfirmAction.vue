<script setup lang="ts">
import { ElMessageBox } from "element-plus";

const props = withDefaults(defineProps<{
	label: string;
	message: string;
	title?: string;
	type?: "primary" | "success" | "warning" | "danger" | "info";
	disabled?: boolean;
	loading?: boolean;
	plain?: boolean;
	size?: "small" | "default" | "large";
}>(), {
	title: "Confirm action",
	type: "danger",
	disabled: false,
	loading: false,
	plain: false,
	size: "default"
});

const emit = defineEmits<{
	(e: "confirm"): void;
}>();

async function confirmAction(): Promise<void> {
	await ElMessageBox.confirm(props.message, props.title, {
		type: props.type === "danger" ? "warning" : props.type,
		confirmButtonText: props.label,
		cancelButtonText: "Cancel",
		distinguishCancelAndClose: true
	});
	emit("confirm");
}
</script>

<template>
	<el-button
		:type="type"
		:disabled="disabled"
		:loading="loading"
		:plain="plain"
		:size="size"
		@click="confirmAction"
	>
		<slot>{{ label }}</slot>
	</el-button>
</template>
