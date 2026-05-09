import { createRouter, createWebHistory, type RouteRecordRaw } from "vue-router";

import { getDefaultHomePath } from "../stores/plugins";
import { getToken } from "../utils/auth";
import DashboardPage from "../views/Dashboard/index.vue";
import LoginPage from "../views/Login/index.vue";
import PermissionPage from "../views/Permission/index.vue";
import PluginPage from "../views/Plugin/index.vue";
import RoleEditPage from "../views/Role/edit.vue";
import RoleListPage from "../views/Role/list.vue";
import SettingPage from "../views/Setting/index.vue";
import UserAddPage from "../views/User/add.vue";
import UserEditPage from "../views/User/edit.vue";
import UserListPage from "../views/User/list.vue";

const routes: RouteRecordRaw[] = [
	{
		path: "/login",
		name: "login",
		component: LoginPage,
		meta: { public: true }
	},
	{
		path: "/",
		redirect: () => getDefaultHomePath("/dashboard")
	},
	{
		path: "/dashboard",
		name: "dashboard",
		component: DashboardPage
	},
	{
		path: "/plugin",
		name: "plugin",
		component: PluginPage
	},
	{
		path: "/user",
		name: "user-list",
		component: UserListPage
	},
	{
		path: "/user/add",
		name: "user-add",
		component: UserAddPage
	},
	{
		path: "/user/:id/edit",
		name: "user-edit",
		component: UserEditPage
	},
	{
		path: "/role",
		name: "role-list",
		component: RoleListPage
	},
	{
		path: "/role/:id/edit",
		name: "role-edit",
		component: RoleEditPage
	},
	{
		path: "/permission",
		name: "permission",
		component: PermissionPage,
		meta: { roles: ["super_admin"] }
	},
	{
		path: "/setting",
		name: "setting",
		component: SettingPage,
		meta: { roles: ["super_admin"] }
	}
];

export const router = createRouter({
	history: createWebHistory(),
	routes
});

router.beforeEach((to) => {
	const token = getToken().trim();
	const isPublic = to.meta.public === true || to.path.startsWith("/plugins/auth");

	if (token === "" && !isPublic) {
		return {
			path: "/login",
			query: { redirect: to.fullPath }
		};
	}

	if (token !== "" && to.path === "/login") {
		const redirect = typeof to.query.redirect === "string" && to.query.redirect.trim() !== ""
			? to.query.redirect
			: getDefaultHomePath("/dashboard");
		return redirect;
	}

	const requiredRoles = Array.isArray(to.meta.roles) ? to.meta.roles : null;
	if (requiredRoles && requiredRoles.length > 0) {
		const currentRole = "super_admin";
		if (!requiredRoles.includes(currentRole)) {
			return getDefaultHomePath("/dashboard");
		}
	}

	return true;
});

declare module "vue-router" {
	interface RouteMeta {
		public?: boolean;
		roles?: string[];
	}
}

