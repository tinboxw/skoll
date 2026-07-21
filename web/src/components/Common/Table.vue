<template>
	<el-card class="table-shell" shadow="never">
		<table class="common-table">
			<slot />
		</table>
	</el-card>
</template>

<style scoped>
.table-shell {
	overflow: auto;
	border: 1px solid var(--color-border);
}

.common-table {
	width: 100%;
	border-collapse: collapse;
	font-size: 0.92rem;
}

:deep(.el-card__body) {
	padding: 0;
	overflow: auto;
}
</style>
