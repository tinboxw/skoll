<script setup lang="ts">
import { Bell, ClipboardList, FileText, FolderOpen, GitPullRequest, LayoutDashboard, ListTree, Puzzle, Settings2, ShieldCheck, Users, UserRoundCog } from "lucide-vue-next";
import type { SidebarItem } from "../../navigation/menu";

withDefaults(defineProps<{
	collapsed: boolean;
	items: SidebarItem[];
	mobile?: boolean;
}>(), {
	mobile: false
});

const emit = defineEmits<{
	(event: "navigate"): void;
}>();

const iconMap = {
	dashboard: LayoutDashboard,
	users: Users,
	roles: UserRoundCog,
	permissions: ShieldCheck,
	menus: ListTree,
	audit: ClipboardList,
	workflow: GitPullRequest,
	todo: Bell,
	forms: FileText,
	plugins: Puzzle,
	settings: Settings2,
	files: FolderOpen
} as const;

function resolveIconComponent(icon: string) {
	return iconMap[icon as keyof typeof iconMap] ?? Puzzle;
}
</script>

<template>
	<aside class="sidebar" :class="{ collapsed, 'sidebar--mobile': mobile }">
		<RouterLink to="/skoll/" class="brand" title="Home">Skoll</RouterLink>
		<nav>
			<RouterLink
				v-for="item in items"
				:key="item.to"
				:to="item.to"
				class="link"
				active-class="active"
				@click="emit('navigate')"
			>
				<component :is="resolveIconComponent(item.icon)" class="icon" :stroke-width="1.8" aria-hidden="true" />
				<span v-if="!collapsed">{{ item.label }}</span>
			</RouterLink>
		</nav>
	</aside>
</template>

<style scoped>
.sidebar {
	padding: 14px 10px;
	border-right: 1px solid var(--color-border);
	background: var(--color-sidebar-bg);
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
	display: flex;
	align-items: center;
	gap: 10px;
	padding: 8px 10px;
	border-radius: 8px;
	text-decoration: none;
	color: var(--color-sidebar-link);
	font-size: 0.95rem;
}

.icon {
	width: 16px;
	height: 16px;
	flex-shrink: 0;
}

.link.active,
.link:hover {
	background: var(--color-sidebar-active);
	color: var(--color-on-primary);
}

.collapsed {
	width: 72px;
}

.collapsed .link {
	justify-content: center;
	padding: 8px;
}

@media (max-width: 860px) {
	.sidebar:not(.sidebar--mobile) {
		display: none;
	}
}

.sidebar--mobile {
	width: 100%;
	min-height: 100%;
	border-right: 0;
}
</style>

