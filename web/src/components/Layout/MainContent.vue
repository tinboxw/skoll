<template>
	<section class="main-content">
		<slot />
	</section>
</template>

<style scoped>
.main-content {
	border-radius: var(--radius-lg);
	border: 1px solid var(--color-border);
	background: var(--color-surface);
	padding: 16px;
	min-height: 70vh;
	box-shadow: 0 10px 26px -24px var(--color-shadow);
}
</style>

