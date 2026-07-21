import type { RouteRecordRaw } from "vue-router";

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

type IntegratedRouteFactory = () => RouteRecordRaw[];

const integratedRouteFactories: Record<string, IntegratedRouteFactory> = {
	pharma_oa: createPharmaOARoutes
};

export function createIntegratedPluginRoutes(pluginID: string): RouteRecordRaw[] {
	return integratedRouteFactories[pluginID]?.() ?? [];
}

export function isIntegratedPluginRoutePath(path: string): boolean {
	return path === "/skoll/pharma-oa" || path.startsWith("/skoll/pharma-oa/");
}

function createPharmaOARoutes(): RouteRecordRaw[] {
	const prefix = "/skoll/pharma-oa";
	return [
		{ path: `${prefix}/announcements`, name: "pharma-oa-announcements", component: PharmaAnnouncementPage, meta: { requiresAuth: true, permissions: ["pharma_oa.announcement.read"] } },
		{ path: `${prefix}/contracts`, name: "pharma-oa-contracts", component: PharmaContractPage, meta: { requiresAuth: true, permissions: ["pharma_oa.contract.read"] } },
		{ path: `${prefix}/qualifications`, name: "pharma-oa-qualifications", component: PharmaQualificationPage, meta: { requiresAuth: true, permissions: ["pharma_oa.qualification.read"] } },
		{ path: `${prefix}/quality-complaints`, name: "pharma-oa-quality-complaints", component: PharmaQualityComplaintPage, meta: { requiresAuth: true, permissions: ["pharma_oa.quality_complaint.read"] } },
		{ path: `${prefix}/drug-recalls`, name: "pharma-oa-drug-recalls", component: PharmaDrugRecallPage, meta: { requiresAuth: true, permissions: ["pharma_oa.drug_recall.read"] } },
		{ path: `${prefix}/cold-chain`, name: "pharma-oa-cold-chain", component: PharmaColdChainPage, meta: { requiresAuth: true, permissions: ["pharma_oa.cold_chain.read"] } },
		{ path: `${prefix}/compliance-dashboard`, name: "pharma-oa-compliance-dashboard", component: PharmaComplianceDashboardPage, meta: { requiresAuth: true, permissions: ["pharma_oa.compliance_dashboard.read"] } },
		{ path: `${prefix}/dashboard`, name: "pharma-oa-dashboard", component: PharmaDashboardPage, meta: { requiresAuth: true, permissions: ["pharma_oa.business_metrics.read"] } },
		{ path: `${prefix}/employees`, name: "pharma-oa-employees", component: PharmaEmployeePage, meta: { permissions: ["pharma_oa.employee.read"] } },
		{ path: `${prefix}/customers`, name: "pharma-oa-customers", component: PharmaCustomerPage, meta: { permissions: ["pharma_oa.customer.read"] } },
		{ path: `${prefix}/customer-follow-ups`, name: "pharma-oa-customer-follow-ups", component: PharmaCustomerFollowUpPage, meta: { requiresAuth: true, permissions: ["pharma_oa.customer_follow_up.read"] } },
		{ path: `${prefix}/sales-opportunities`, name: "pharma-oa-sales-opportunities", component: PharmaSalesOpportunityPage, meta: { requiresAuth: true, permissions: ["pharma_oa.sales_opportunity.read"] } },
		{ path: `${prefix}/payment-invoices`, name: "pharma-oa-payment-invoices", component: PharmaPaymentInvoicePage, meta: { requiresAuth: true, permissions: ["pharma_oa.payment_invoice.read"] } },
		{ path: `${prefix}/purchase-inbounds`, name: "pharma-oa-purchase-inbounds", component: PharmaPurchaseInboundPage, meta: { requiresAuth: true, permissions: ["pharma_oa.inbound.read"] } },
		{ path: `${prefix}/sales`, name: "pharma-oa-sales", component: PharmaSalesPage, meta: { requiresAuth: true, permissions: ["pharma_oa.sales.order.read", "pharma_oa.sales.outbound.read"] } }
	];
}
