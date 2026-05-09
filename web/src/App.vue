<script setup lang="ts">
import { computed } from "vue";
import { usePluginStore } from "./stores/plugins";

const store = usePluginStore();
const pluginCount = computed(() => store.items.length);
</script>

<template>
	<main class="app-shell">
		<header class="toolbar">
			<h1>Skoll Frontend Plugin System</h1>
			<p>Loaded Plugins: {{ pluginCount }}</p>
		</header>

		<section class="content-grid">
			<aside class="panel">
				<h2>Plugin Registry</h2>
				<ul>
					<li v-for="item in store.items" :key="item.id">
						<strong>{{ item.name }}</strong>
						<span>({{ item.id }}@{{ item.version }})</span>
					</li>
				</ul>
			</aside>

			<section class="panel">
				<h2>Plugin View</h2>
				<RouterView />
			</section>
		</section>
	</main>
</template>

<style scoped>
.app-shell {
	padding: 20px;
	font-family: "Segoe UI", sans-serif;
	color: #182233;
}

.toolbar {
	margin-bottom: 16px;
}

.content-grid {
	display: grid;
	grid-template-columns: 320px 1fr;
	gap: 16px;
}

.panel {
	border: 1px solid #d7dfea;
	border-radius: 10px;
	padding: 12px;
	background: #ffffff;
}

@media (max-width: 860px) {
	.content-grid {
		grid-template-columns: 1fr;
	}
}
</style>

