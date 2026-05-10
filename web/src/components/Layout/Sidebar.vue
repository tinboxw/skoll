<script setup lang="ts">
defineProps<{
	collapsed: boolean;
	items: Array<{ label: string; to: string }>;
}>();
</script>

<template>
	<aside class="sidebar" :class="{ collapsed }">
		<RouterLink to="/" class="brand" title="Home">Skoll</RouterLink>
		<nav>
			<RouterLink
				v-for="item in items"
				:key="item.to"
				:to="item.to"
				class="link"
				active-class="active"
			>
				{{ collapsed ? item.label.slice(0, 1) : item.label }}
			</RouterLink>
		</nav>
	</aside>
</template>

<style scoped>
.sidebar {
	padding: 14px 10px;
	border-right: 1px solid var(--color-border);
	background: linear-gradient(180deg, var(--color-sidebar-bg) 0%, #182230 100%);
	color: var(--color-on-primary);
}

.brand {
	display: inline-flex;
	align-items: center;
	font-size: 1.2rem;
	font-weight: 700;
	margin-bottom: 14px;
	padding-left: 8px;
	color: var(--color-on-primary);
	text-decoration: none;
}

.brand:hover {
	opacity: 0.9;
}

nav {
	display: grid;
	gap: 8px;
}

.link {
	padding: 8px 10px;
	border-radius: 8px;
	text-decoration: none;
	color: var(--color-sidebar-link);
	font-size: 0.95rem;
}

.link.active,
.link:hover {
	background: var(--color-sidebar-active);
	color: var(--color-on-primary);
}

.collapsed {
	width: 72px;
}

@media (max-width: 860px) {
	.sidebar {
		display: none;
	}
}
</style>

