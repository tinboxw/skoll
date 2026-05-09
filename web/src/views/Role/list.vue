<script setup lang="ts">
import { useI18n } from "../../i18n";

const rows = [
	{ id: "r-001", name: "super_admin", users: 1 },
	{ id: "r-002", name: "auditor", users: 2 }
];

const { t } = useI18n();
</script>

<template>
	<section>
		<h2>{{ t("page.roles") }}</h2>
		<table>
			<thead>
				<tr>
					<th>{{ t("table.id") }}</th>
					<th>{{ t("table.name") }}</th>
					<th>{{ t("table.users") }}</th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="item in rows" :key="item.id">
					<td>{{ item.id }}</td>
					<td>{{ item.name }}</td>
					<td>{{ item.users }}</td>
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

