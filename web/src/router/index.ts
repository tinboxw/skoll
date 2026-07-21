import { createRouter, createWebHistory, type RouteRecordRaw } from "vue-router";

import { canAccessRoute, isPublicRoute } from "../permissions/route";
import { waitForPluginBootstrap } from "../plugins";
import { clearDefaultHomePath, getDefaultHomePath, getSystemDefaultHomePath, resolveValidatedDefaultHomePath, usePluginStore } from "../stores/plugins";
import { getStoredPermissions, getStoredUserRole } from "../stores/user";
import { getToken } from "../utils/auth";

const LoginPage = () => import("../views/Login/index.vue");
const DashboardPage = () => import("../views/Dashboard/index.vue");
const AuditPage = () => import("../views/Audit/index.vue");
const DictionaryPage = () => import("../views/Dictionary/index.vue");
const FilePage = () => import("../views/File/index.vue");
const FormBuilderPage = () => import("../views/FormBuilder/index.vue");
const ForbiddenPage = () => import("../views/Forbidden/index.vue");
const MenuPage = () => import("../views/Menu/index.vue");
const OrganizationPage = () => import("../views/Organization/index.vue");
const PermissionPage = () => import("../views/Permission/index.vue");
const PharmaCustomerPage = () => import("../views/PharmaCustomer/index.vue");
const PharmaCustomerFollowUpPage = () => import("../views/PharmaCustomerFollowUp/index.vue");
const PharmaSalesOpportunityPage = () => import("../views/PharmaSalesOpportunity/index.vue");
const PharmaPaymentInvoicePage = () => import("../views/PharmaPaymentInvoice/index.vue");
const PharmaAnnouncementPage = () => import("../views/PharmaAnnouncement/index.vue");
const PharmaContractPage = () => import("../views/PharmaContract/index.vue");
const PharmaQualificationPage = () => import("../views/PharmaQualification/index.vue");
const PharmaQualityComplaintPage = () => import("../views/PharmaQualityComplaint/index.vue");
const PharmaDrugRecallPage = () => import("../views/PharmaDrugRecall/index.vue");
const PharmaColdChainPage = () => import("../views/PharmaColdChain/index.vue");
const PharmaComplianceDashboardPage = () => import("../views/PharmaComplianceDashboard/index.vue");
const PharmaDashboardPage = () => import("../views/PharmaDashboard/index.vue");
const PharmaPurchaseInboundPage = () => import("../views/PharmaPurchaseInbound/index.vue");
const PharmaSalesPage = () => import("../views/PharmaSales/index.vue");
const PharmaEmployeePage = () => import("../views/PharmaEmployee/index.vue");
const PluginPage = () => import("../views/Plugin/index.vue");
const ProfilePage = () => import("../views/Profile/index.vue");
const RoleEditPage = () => import("../views/Role/edit.vue");
const RoleListPage = () => import("../views/Role/list.vue");
const SettingPage = () => import("../views/Setting/index.vue");
const TodoCenterPage = () => import("../views/TodoCenter/index.vue");
const UserAddPage = () => import("../views/User/add.vue");
const UserBatchPage = () => import("../views/User/batch.vue");
const UserEditPage = () => import("../views/User/edit.vue");
const UserListPage = () => import("../views/User/list.vue");
const WorkflowPage = () => import("../views/Workflow/index.vue");

const ADMIN_PREFIX = "/skoll";
const ADMIN_LOGIN_PATH = `${ADMIN_PREFIX}/login`;
const ADMIN_FORBIDDEN_PATH = `${ADMIN_PREFIX}/forbidden`;

const routes: RouteRecordRaw[] = [
	{
		path: ADMIN_LOGIN_PATH,
		name: "login",
		component: LoginPage,
		meta: { public: true }
	},
	{
		path: "/",
		redirect: () => resolveSafeDefaultHomePath()
	},
	{
		path: `${ADMIN_PREFIX}`,
		redirect: () => resolveSafeDefaultHomePath()
	},
	{
		path: `${ADMIN_PREFIX}/`,
		redirect: () => resolveSafeDefaultHomePath()
	},
	{
		path: `${ADMIN_PREFIX}/dashboard`,
		name: "dashboard",
		component: DashboardPage
	},
	{
		path: ADMIN_FORBIDDEN_PATH,
		name: "forbidden",
		component: ForbiddenPage,
		meta: { requiresAuth: true }
	},
	{
		path: `${ADMIN_PREFIX}/plugin`,
		name: "plugin",
		component: PluginPage,
		meta: { permissions: ["plugin.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/user`,
		name: "user-list",
		component: UserListPage,
		meta: { permissions: ["user.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/user/add`,
		name: "user-add",
		component: UserAddPage,
		meta: { permissions: ["user.create"] }
	},
	{
		path: `${ADMIN_PREFIX}/user/batch-add`,
		name: "user-batch-add",
		component: UserBatchPage,
		meta: { permissions: ["user.create"] }
	},
	{
		path: `${ADMIN_PREFIX}/user/:id/edit`,
		name: "user-edit",
		component: UserEditPage,
		meta: { permissions: ["user.update"] }
	},
	{
		path: `${ADMIN_PREFIX}/profile`,
		name: "profile",
		component: ProfilePage
	},
	{
		path: `${ADMIN_PREFIX}/role`,
		name: "role-list",
		component: RoleListPage,
		meta: { permissions: ["role.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/role/:id/edit`,
		name: "role-edit",
		component: RoleEditPage,
		meta: { permissions: ["role.update"] }
	},
	{
		path: `${ADMIN_PREFIX}/permission`,
		name: "permission",
		component: PermissionPage,
		meta: { permissions: ["permission.manage"] }
	},
	{
		path: `${ADMIN_PREFIX}/menu`,
		name: "menu",
		component: MenuPage,
		meta: { permissions: ["system.manage"] }
	},
	{
		path: `${ADMIN_PREFIX}/dictionary`,
		name: "dictionary",
		component: DictionaryPage,
		meta: { permissions: ["dict.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/files`,
		name: "files",
		component: FilePage
	},
	{
		path: `${ADMIN_PREFIX}/organization`,
		name: "organization",
		component: OrganizationPage,
		meta: { permissions: ["org.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/audit`,
		name: "audit",
		component: AuditPage,
		meta: { permissions: ["audit.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/workflow`,
		name: "workflow",
		component: WorkflowPage
	},
	{
		path: `${ADMIN_PREFIX}/todo`,
		name: "todo-center",
		component: TodoCenterPage
	},
	{
		path: `${ADMIN_PREFIX}/form-builder`,
		name: "form-builder",
		component: FormBuilderPage
	},
	{
		path: `${ADMIN_PREFIX}/pharma-oa/announcements`,
		name: "pharma-oa-announcements",
		component: PharmaAnnouncementPage,
		meta: { requiresAuth: true, permissions: ["pharma_oa.announcement.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/pharma-oa/contracts`,
		name: "pharma-oa-contracts",
		component: PharmaContractPage,
		meta: { requiresAuth: true, permissions: ["pharma_oa.contract.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/pharma-oa/qualifications`,
		name: "pharma-oa-qualifications",
		component: PharmaQualificationPage,
		meta: { requiresAuth: true, permissions: ["pharma_oa.qualification.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/pharma-oa/quality-complaints`,
		name: "pharma-oa-quality-complaints",
		component: PharmaQualityComplaintPage,
		meta: { requiresAuth: true, permissions: ["pharma_oa.quality_complaint.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/pharma-oa/drug-recalls`,
		name: "pharma-oa-drug-recalls",
		component: PharmaDrugRecallPage,
		meta: { requiresAuth: true, permissions: ["pharma_oa.drug_recall.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/pharma-oa/cold-chain`,
		name: "pharma-oa-cold-chain",
		component: PharmaColdChainPage,
		meta: { requiresAuth: true, permissions: ["pharma_oa.cold_chain.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/pharma-oa/compliance-dashboard`,
		name: "pharma-oa-compliance-dashboard",
		component: PharmaComplianceDashboardPage,
		meta: { requiresAuth: true, permissions: ["pharma_oa.compliance_dashboard.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/pharma-oa/dashboard`,
		name: "pharma-oa-dashboard",
		component: PharmaDashboardPage,
		meta: { requiresAuth: true, permissions: ["pharma_oa.business_metrics.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/pharma-oa/employees`,
		name: "pharma-oa-employees",
		component: PharmaEmployeePage,
		meta: { permissions: ["pharma_oa.employee.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/pharma-oa/customers`,
		name: "pharma-oa-customers",
		component: PharmaCustomerPage,
		meta: { permissions: ["pharma_oa.customer.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/pharma-oa/customer-follow-ups`,
		name: "pharma-oa-customer-follow-ups",
		component: PharmaCustomerFollowUpPage,
		meta: { requiresAuth: true, permissions: ["pharma_oa.customer_follow_up.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/pharma-oa/sales-opportunities`,
		name: "pharma-oa-sales-opportunities",
		component: PharmaSalesOpportunityPage,
		meta: { requiresAuth: true, permissions: ["pharma_oa.sales_opportunity.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/pharma-oa/payment-invoices`,
		name: "pharma-oa-payment-invoices",
		component: PharmaPaymentInvoicePage,
		meta: { requiresAuth: true, permissions: ["pharma_oa.payment_invoice.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/pharma-oa/purchase-inbounds`,
		name: "pharma-oa-purchase-inbounds",
		component: PharmaPurchaseInboundPage,
		meta: { requiresAuth: true, permissions: ["pharma_oa.inbound.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/pharma-oa/sales`,
		name: "pharma-oa-sales",
		component: PharmaSalesPage,
		meta: { requiresAuth: true, permissions: ["pharma_oa.sales.order.read", "pharma_oa.sales.outbound.read"] }
	},
	{
		path: `${ADMIN_PREFIX}/setting`,
		name: "setting",
		component: SettingPage,
		meta: { permissions: ["system.manage"] }
	}
];

export const router = createRouter({
	history: createWebHistory("/"),
	routes
});

function isKnownStaticPath(path: string): boolean {
	return routes.some((route) => typeof route.path === "string" && route.path === path);
}

function isPluginHomePath(path: string): boolean {
	if (path.startsWith(`${ADMIN_PREFIX}/plugins/`)) {
		return true;
	}
	return /^\/(?!skoll(?:\/|$))[^/]+\/?$/.test(path);
}

function normalizePluginEntryPath(path: string): string {
	const normalized = path.trim();
	if (normalized.startsWith("/plugins/")) {
		return `${ADMIN_PREFIX}${normalized}`;
	}
	return normalized;
}

function resolveSafeDefaultHomePath(): string {
	const fallback = getSystemDefaultHomePath();
	const target = normalizePluginEntryPath(getDefaultHomePath(fallback));
	if (isPluginHomePath(target)) {
		return target;
	}
	if (isKnownStaticPath(target)) {
		return target;
	}
	clearDefaultHomePath();
	return fallback;
}

async function resolveSafeTargetPath(path: string): Promise<string> {
	path = normalizePluginEntryPath(path);
	if (!isPluginHomePath(path)) {
		return path;
	}
	await waitForPluginBootstrap();
	const pluginStore = usePluginStore();
	const fallback = getSystemDefaultHomePath();
	if (getDefaultHomePath(fallback) === path) {
		const validated = resolveValidatedDefaultHomePath(pluginStore.items, fallback);
		if (validated !== path) {
			clearDefaultHomePath();
			return validated;
		}
	}
	if (router.resolve(path).matched.length > 0) {
		return path;
	}
	if (getDefaultHomePath(fallback) === path) {
		clearDefaultHomePath();
	}
	return fallback;
}


function normalizeRedirectPath(raw: string): string {
	const normalized = raw.trim();
	const withSlash = normalized.startsWith("/") ? normalized : `/${normalized}`;
	if (withSlash.startsWith("/plugins/")) {
		return `${ADMIN_PREFIX}${withSlash}`;
	}
	if (
		withSlash === "/dashboard" || withSlash.startsWith("/dashboard/") ||
		withSlash === "/forbidden" || withSlash.startsWith("/forbidden/") ||
		withSlash === "/plugin" || withSlash.startsWith("/plugin/") ||
		withSlash === "/user" || withSlash.startsWith("/user/") ||
		withSlash === "/role" || withSlash.startsWith("/role/") ||
		withSlash === "/permission" || withSlash.startsWith("/permission/") ||
		withSlash === "/menu" || withSlash.startsWith("/menu/") ||
		withSlash === "/dictionary" || withSlash.startsWith("/dictionary/") ||
		withSlash === "/files" || withSlash.startsWith("/files/") ||
		withSlash === "/organization" || withSlash.startsWith("/organization/") ||
		withSlash === "/audit" || withSlash.startsWith("/audit/") ||
		withSlash === "/workflow" || withSlash.startsWith("/workflow/") ||
		withSlash === "/todo" || withSlash.startsWith("/todo/") ||
		withSlash === "/form-builder" || withSlash.startsWith("/form-builder/") ||
		withSlash === "/pharma-oa" || withSlash.startsWith("/pharma-oa/") ||
		withSlash === "/setting" || withSlash.startsWith("/setting/") ||
		withSlash === "/profile" || withSlash.startsWith("/profile/") ||
		withSlash === "/login" || withSlash.startsWith("/login/")
	) {
		return `${ADMIN_PREFIX}${withSlash}`;
	}
	return withSlash;
}

router.beforeEach(async (to) => {
	const token = getToken().trim();
	const isPublic = isPublicRoute(to);

	if (token === "" && !isPublic) {
		return {
			path: ADMIN_LOGIN_PATH,
			query: { redirect: to.fullPath }
		};
	}

	if (token !== "" && to.path === ADMIN_LOGIN_PATH) {
		const redirect = typeof to.query.redirect === "string" && to.query.redirect.trim() !== ""
			? normalizeRedirectPath(to.query.redirect)
			: resolveSafeDefaultHomePath();
		return await resolveSafeTargetPath(redirect);
	}

	if (isPluginHomePath(to.path)) {
		const safePath = await resolveSafeTargetPath(to.path);
		if (safePath !== to.path) {
			return safePath;
		}
		// If the target plugin route was added during bootstrap, rematch the same URL once.
		if (to.matched.length === 0 && router.resolve(to.path).matched.length > 0) {
			return {
				path: to.fullPath,
				replace: true
			};
		}
	}

	if (!canAccessRoute(to, getStoredUserRole(), getStoredPermissions())) {
		return {
			path: ADMIN_FORBIDDEN_PATH,
			query: { from: to.fullPath }
		};
	}

	return true;
});

declare module "vue-router" {
	interface RouteMeta {
		public?: boolean;
		roles?: string[];
		permissions?: string[];
		mode?: "all" | "any";
	}
}

