<script setup lang="ts">
import { ref } from "vue";

import { useI18n } from "../../i18n";

const permissions = ref([
	{ key: "user.read", enabled: true },
	{ key: "user.write", enabled: false },
	{ key: "plugin.manage", enabled: true }
]);
const { t } = useI18n();
</script>

<template>
	<section>
		<h2>{{ t("page.roleEdit") }}</h2>
		<ul>
			<li v-for="item in permissions" :key="item.key">
				<label>
					<input v-model="item.enabled" type="checkbox" />
					<span>{{ item.key }}</span>
				</label>
			</li>
		</ul>
	</section>
</template>

<style scoped>
ul {
	padding: 0;
	list-style: none;
	display: grid;
	gap: 8px;
}

label {
	display: flex;
	align-items: center;
	gap: 8px;
	color: var(--color-text-muted);
}
</style>

