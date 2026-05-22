<template>
	<section class="main-content" :class="{ 'plugin-main-content': isPluginRoute }">
		<slot />
	</section>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";

const route = useRoute();

const isPluginRoute = computed(() => {
	const name = String(route.name || "");
	const path = String(route.path || "");
	return name.startsWith("plugin-") || name.startsWith("app-home-") || path.startsWith("/skoll/plugins/");
});
</script>

<style scoped>
.main-content {
	border-radius: var(--radius-lg);
	border: 1px solid var(--color-border);
	background: var(--color-surface);
	padding: 16px;
	min-height: 70vh;
	box-shadow: 0 10px 26px -24px var(--color-shadow);
}

.plugin-main-content {
	padding: 8px;
	min-height: 0;
	height: 100%;
	overflow: hidden;
}
</style>

