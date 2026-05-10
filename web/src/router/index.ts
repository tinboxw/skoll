import { createRouter, createWebHistory, type RouteRecordRaw } from "vue-router";

import { waitForPluginBootstrap } from "../plugins";
import { clearDefaultHomePath, getDefaultHomePath, getSystemDefaultHomePath } from "../stores/plugins";
import { getStoredPermissions, getStoredUserRole } from "../stores/user";
import { getToken } from "../utils/auth";
import DashboardPage from "../views/Dashboard/index.vue";
import LoginPage from "../views/Login/index.vue";
import PermissionPage from "../views/Permission/index.vue";
import PluginPage from "../views/Plugin/index.vue";
import ProfilePage from "../views/Profile/index.vue";
import RoleEditPage from "../views/Role/edit.vue";
import RoleListPage from "../views/Role/list.vue";
import SettingPage from "../views/Setting/index.vue";
import UserAddPage from "../views/User/add.vue";
import UserBatchPage from "../views/User/batch.vue";
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
		redirect: () => resolveSafeDefaultHomePath()
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
		path: "/user/batch-add",
		name: "user-batch-add",
		component: UserBatchPage
	},
	{
		path: "/user/:id/edit",
		name: "user-edit",
		component: UserEditPage
	},
	{
		path: "/profile",
		name: "profile",
		component: ProfilePage
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
		meta: { permissions: ["permission.manage"] }
	},
	{
		path: "/setting",
		name: "setting",
		component: SettingPage,
		meta: { permissions: ["role.manage"] }
	}
];

export const router = createRouter({
	history: createWebHistory(),
	routes
});

function isKnownStaticPath(path: string): boolean {
	return routes.some((route) => typeof route.path === "string" && route.path === path);
}

function resolveSafeDefaultHomePath(): string {
	const fallback = getSystemDefaultHomePath();
	const target = getDefaultHomePath(fallback);
	if (target.startsWith("/plugins/")) {
		return target;
	}
	if (isKnownStaticPath(target)) {
		return target;
	}
	clearDefaultHomePath();
	return fallback;
}

async function resolveSafeTargetPath(path: string): Promise<string> {
	if (!path.startsWith("/plugins/")) {
		return path;
	}
	await waitForPluginBootstrap();
	if (router.resolve(path).matched.length > 0) {
		return path;
	}
	const fallback = getSystemDefaultHomePath();
	if (getDefaultHomePath(fallback) === path) {
		clearDefaultHomePath();
	}
	return fallback;
}

router.beforeEach(async (to) => {
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
			: resolveSafeDefaultHomePath();
		return await resolveSafeTargetPath(redirect);
	}

	if (to.path.startsWith("/plugins/")) {
		const safePath = await resolveSafeTargetPath(to.path);
		if (safePath !== to.path) {
			return safePath;
		}
	}

	const requiredRoles = Array.isArray(to.meta.roles) ? to.meta.roles : null;
	const requiredPermissions = Array.isArray(to.meta.permissions) ? to.meta.permissions : null;
	if (requiredRoles && requiredRoles.length > 0) {
		const currentRole = getStoredUserRole();
		if (!requiredRoles.includes(currentRole)) {
			return resolveSafeDefaultHomePath();
		}
	}

	if (requiredPermissions && requiredPermissions.length > 0) {
		const currentRole = getStoredUserRole();
		if (currentRole !== "super_admin") {
			const permissions = getStoredPermissions();
			const allowed = requiredPermissions.every((item) => permissions.includes(item));
			if (!allowed) {
				return resolveSafeDefaultHomePath();
			}
		}
	}

	return true;
});

declare module "vue-router" {
	interface RouteMeta {
		public?: boolean;
		roles?: string[];
		permissions?: string[];
	}
}

