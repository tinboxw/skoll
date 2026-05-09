<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute } from "vue-router";

import { useI18n } from "../../i18n";

const route = useRoute();
const role = ref("viewer");
const userId = computed(() => String(route.params.id ?? "unknown"));
const { t } = useI18n();
</script>

<template>
	<section>
		<h2>{{ t("page.userEdit") }}</h2>
		<p>{{ t("user.editing") }}: {{ userId }}</p>
		<label>
			<span>{{ t("role.field") }}</span>
			<select v-model="role">
				<option value="viewer">{{ t("role.viewer") }}</option>
				<option value="editor">{{ t("role.editor") }}</option>
				<option value="super_admin">{{ t("role.superAdmin") }}</option>
			</select>
		</label>
	</section>
</template>

<style scoped>
p,
span {
	color: var(--color-text-muted);
}

label {
	display: grid;
	gap: 6px;
	max-width: 260px;
}

select {
	padding: 8px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
}
</style>

