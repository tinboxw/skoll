# Pharma OA Locale Baseline 2026-07-19

> Work Items: `H4-01`, `H4-02`
> Default locale: `zh-CN`
> Supported locales: `zh-CN`, `en-US`
> Executable baseline: `web/i18n/pharma-oa-baseline.json`

## Scope

This inventory covers the active Pharma OA routes, menus, forms, validation and messages, page states, runtime business values, and plugin bridge. H4-01 established the executable baseline; H4-02 completed the Chinese/English migration and strengthened the baseline so newly introduced visible copy fails CI.

## View Inventory

| View | Namespace | Current state | Main copy surfaces |
| --- | --- | --- | --- |
| PharmaAnnouncement | `pharma.announcement` | Localized | page title, form, publish/read actions, messages, loading/empty/error |
| PharmaColdChain | `pharma.coldChain` | Localized | context/record/anomaly/job tabs, forms, retry messages, states |
| PharmaComplianceDashboard | `pharma.complianceDashboard` | Localized | metrics, risk filters, trace values, export action, loading/empty/error |
| PharmaContract | `pharma.contract` | Localized | ledger, approval/rejection form, expiry scan, messages, states |
| PharmaCustomer | `customer` | Localized | data-scope workflow, business values, forms, validation, messages, states |
| PharmaCustomerFollowUp | `pharma.customerFollowUp` | Localized | plans, completion/cancel actions, validation, messages, states |
| PharmaDashboard | `pharma.dashboard` | Localized | metric cards, trend/risk panels, export jobs, states |
| PharmaDrugRecall | `pharma.drugRecall` | Localized | batch/scope/task flow, completion messages, states |
| PharmaEmployee | `pharma.employee` | Localized | archive form, qualification fields, leave flow, messages, states |
| PharmaPaymentInvoice | `pharma.paymentInvoice` | Localized | payment/invoice tabs, receive/void/retry actions, messages, states |
| PharmaPurchaseInbound | `pharma.purchaseInbound` | Localized | inbound list/detail, status labels, loading/empty/error |
| PharmaQualification | `pharma.qualification` | Localized | qualification ledger, expiry scan, status labels, states |
| PharmaQualityComplaint | `pharma.qualityComplaint` | Localized | complaint/batch flow, resolve/reject actions, messages, states |
| PharmaSales | `pharma.sales` | Localized | order/outbound tabs, create actions, status labels, states |
| PharmaSalesOpportunity | `pharma.salesOpportunity` | Localized | opportunity form/stages/statistics, advance messages, states |

The executable inventory currently validates 15 localized views, 1,361 keys per locale, 1,254 static references, and zero detected hard-coded user-facing strings.

## Plugin Bridge

`plugins/pharma_oa/plugin.yaml` already declares Chinese and English plugin names, menu labels, configuration labels, and `i18n_locales: [zh-CN, en-US]`. The locale baseline check fails if either locale or either bilingual name/menu field is removed.

## Automated Policy

`npm run check:i18n` enforces:

1. `zh-CN` is the explicit fallback locale and the first supported dictionary.
2. Chinese and English dictionaries have identical, duplicate-free key sets.
3. Every literal `t("key")` or `translate("key")` reference under `web/src` exists in both dictionaries, including the Pharma OA catalog in `web/src/i18n/pharma.json`.
4. All `Pharma*` main views remain represented in the versioned inventory.
5. Template text, user-facing attributes, interpolation branches, messages, confirmation prompts/buttons, validation callbacks, and assigned error text cannot reintroduce hard-coded copy.
6. Runtime status, risk, stage, channel, source, subject, party, and time-bucket values resolve through the locale catalog instead of displaying API enums.
7. Per-view translation-key and hard-coded-copy counts cannot drift silently.
8. The Pharma OA plugin bridge keeps bilingual manifest metadata.

`npm run typecheck` runs this policy before `vue-tsc`, so missing locale keys and unreviewed inventory changes fail CI.
