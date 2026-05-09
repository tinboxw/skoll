<script setup lang="ts">
const emit = defineEmits<{
	(event: "submit"): void;
}>();
</script>

<template>
	<form class="form" @submit.prevent="emit('submit')">
		<slot />
	</form>
</template>

<style scoped>
.form {
	display: grid;
	gap: 12px;
}
</style>

