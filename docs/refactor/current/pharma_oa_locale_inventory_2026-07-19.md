# Pharma OA Locale Baseline 2026-07-19

> Work Item: `H4-01`
> Default locale: `zh-CN`
> Supported locales: `zh-CN`, `en-US`
> Executable baseline: `web/i18n/pharma-oa-baseline.json`

## Scope

This inventory covers the active Pharma OA routes, menus, forms, validation and messages, page states, and plugin bridge. H4-01 records and gates the current debt; H4-02 replaces the recorded hard-coded copy with stable Chinese and English locale keys.

## View Inventory

| View | Namespace | Current state | Main copy surfaces |
| --- | --- | --- | --- |
| PharmaAnnouncement | `pharma.announcement` | Hard-coded | page title, form, publish/read actions, messages, loading/empty/error |
| PharmaColdChain | `pharma.coldChain` | Hard-coded | context/record/anomaly/job tabs, forms, retry messages, states |
| PharmaComplianceDashboard | `pharma.complianceDashboard` | Hard-coded | metrics, risk filters, export action, loading/empty/error |
| PharmaContract | `pharma.contract` | Hard-coded | ledger, approval/rejection form, expiry scan, messages, states |
| PharmaCustomer | `customer` | Partial | main workflow and states use locale keys; business defaults remain inventory debt |
| PharmaCustomerFollowUp | `pharma.customerFollowUp` | Hard-coded | plans, completion/cancel actions, validation, messages, states |
| PharmaDashboard | `pharma.dashboard` | Hard-coded | metric cards, trend/risk panels, export jobs, states |
| PharmaDrugRecall | `pharma.drugRecall` | Hard-coded | batch/scope/task flow, completion messages, states |
| PharmaEmployee | `pharma.employee` | Hard-coded | archive form, qualification fields, leave flow, messages, states |
| PharmaPaymentInvoice | `pharma.paymentInvoice` | Hard-coded | payment/invoice tabs, receive/void/retry actions, messages, states |
| PharmaPurchaseInbound | `pharma.purchaseInbound` | Hard-coded | inbound list/detail, status labels, loading/empty/error |
| PharmaQualification | `pharma.qualification` | Hard-coded | qualification ledger, expiry scan, status labels, states |
| PharmaQualityComplaint | `pharma.qualityComplaint` | Hard-coded | complaint/batch flow, resolve/reject actions, messages, states |
| PharmaSales | `pharma.sales` | Hard-coded | order/outbound tabs, create actions, status labels, states |
| PharmaSalesOpportunity | `pharma.salesOpportunity` | Hard-coded | opportunity form/stages/statistics, advance messages, states |

## Plugin Bridge

`plugins/pharma_oa/plugin.yaml` already declares Chinese and English plugin names, menu labels, configuration labels, and `i18n_locales: [zh-CN, en-US]`. The locale baseline check fails if either locale or either bilingual name/menu field is removed.

## Automated Policy

`npm run check:i18n` enforces:

1. `zh-CN` is the explicit fallback locale and the first supported dictionary.
2. Chinese and English dictionaries have identical, duplicate-free key sets.
3. Every literal `t("key")` or `translate("key")` reference under `web/src` exists in both dictionaries.
4. All `Pharma*` main views remain represented in the versioned inventory.
5. Per-view translation-key and hard-coded-copy counts cannot drift silently.
6. The Pharma OA plugin bridge keeps bilingual manifest metadata.

`npm run typecheck` runs this policy before `vue-tsc`, so missing locale keys and unreviewed inventory changes fail CI.
