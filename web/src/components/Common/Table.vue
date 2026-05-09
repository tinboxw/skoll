<template>
	<div class="table-shell">
		<table>
			<slot />
		</table>
	</div>
</template>

<style scoped>
.table-shell {
	overflow: auto;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
}

table {
	width: 100%;
	border-collapse: collapse;
	font-size: 0.92rem;
}
</style>

