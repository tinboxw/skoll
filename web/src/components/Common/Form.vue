<script setup lang="ts">
const emit = defineEmits<{
	(event: "submit"): void;
}>();
</script>

<template>
	<el-form class="form" label-position="top" @submit.prevent="emit('submit')">
		<slot />
	</el-form>
</template>

<style scoped>
.form {
	display: grid;
	gap: 12px;
}
</style>
