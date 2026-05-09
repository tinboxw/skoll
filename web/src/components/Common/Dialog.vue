<script setup lang="ts">
defineProps<{
	title: string;
	open: boolean;
}>();

const emit = defineEmits<{
	(event: "close"): void;
}>();
</script>

<template>
	<div v-if="open" class="mask" @click.self="emit('close')">
		<div class="dialog">
			<header>
				<h3>{{ title }}</h3>
				<button type="button" @click="emit('close')">X</button>
			</header>
			<section>
				<slot />
			</section>
		</div>
	</div>
</template>

<style scoped>
.mask {
	position: fixed;
	inset: 0;
	background: rgba(0, 0, 0, 0.35);
	display: grid;
	place-items: center;
}

.dialog {
	background: #fff;
	border-radius: 10px;
	min-width: 320px;
	max-width: 520px;
	padding: 12px;
}

header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 10px;
}

button {
	border: none;
	background: #eff4ff;
	border-radius: 6px;
	padding: 4px 8px;
	cursor: pointer;
}
</style>

