<script setup lang="ts">
import { useI18n } from "../../i18n";

const rows = [
	{ id: "u-001", name: "Admin", role: "super_admin" },
	{ id: "u-002", name: "Auditor", role: "auditor" }
];

const { t } = useI18n();
</script>

<template>
	<section>
		<h2>{{ t("page.users") }}</h2>
		<table>
			<thead>
				<tr>
					<th>{{ t("table.id") }}</th>
					<th>{{ t("table.name") }}</th>
					<th>{{ t("table.role") }}</th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="item in rows" :key="item.id">
					<td>{{ item.id }}</td>
					<td>{{ item.name }}</td>
					<td>{{ item.role }}</td>
				</tr>
			</tbody>
		</table>
	</section>
</template>

<style scoped>
table {
	width: 100%;
	border-collapse: collapse;
}

th,
td {
	text-align: left;
	padding: 8px;
	border-bottom: 1px solid var(--color-border);
}
</style>

