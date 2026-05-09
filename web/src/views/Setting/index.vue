<script setup lang="ts">
import { ref } from "vue";

import { useI18n } from "../../i18n";

const auditRetentionDays = ref(90);
const pluginAutoEnable = ref(true);
const { t } = useI18n();
</script>

<template>
	<section>
		<h2>{{ t("page.settings") }}</h2>
		<p>{{ t("settings.desc") }}</p>
		<div class="grid">
			<label>
				<span>{{ t("settings.auditRetention") }}</span>
				<input v-model.number="auditRetentionDays" type="number" min="1" />
			</label>
			<label class="toggle">
				<input v-model="pluginAutoEnable" type="checkbox" />
				<span>{{ t("settings.autoEnable") }}</span>
			</label>
		</div>
	</section>
</template>

<style scoped>
.grid {
	display: grid;
	gap: 12px;
	max-width: 380px;
}

label {
	display: grid;
	gap: 6px;
	color: var(--color-text-muted);
}

input[type="number"] {
	padding: 8px;
	border-radius: var(--radius-md);
	border: 1px solid var(--color-border);
	background: var(--color-surface-soft);
}

.toggle {
	grid-template-columns: auto 1fr;
	align-items: center;
	column-gap: 8px;
}
</style>

