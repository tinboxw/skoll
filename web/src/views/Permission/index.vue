<script setup lang="ts">
import { useI18n } from "../../i18n";

const { t } = useI18n();
</script>

<template>
	<section>
		<h2>{{ t("page.permissions") }}</h2>
		<p>{{ t("permission.desc") }}</p>
		<ul>
			<li>{{ t("permission.hint1") }}</li>
			<li>{{ t("permission.hint2") }}</li>
		</ul>
	</section>
</template>

<style scoped>
p,
li {
	color: var(--color-text-muted);
}
</style>

