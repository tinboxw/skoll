# web/src/components/Common

## Purpose

Common components form the Plugin UI Kit for host pages and business-plugin pages. They wrap Element Plus with Skoll spacing, density, theme tokens, and complete operational states.

## Components

| Component | Purpose | State Coverage |
| --- | --- | --- |
| `PageShell.vue` | Page heading, metadata/actions, and page-level loading/error/forbidden states | loading, error, no-permission |
| `PageToolbar.vue` | Section toolbar for filters, summary text, and action groups | responsive action wrapping |
| `FilterBar.vue` | Predictable filter layout with action slot | responsive filter grid |
| `DataTable.vue` | Typed column table shell with empty/error/forbidden/loading states and action/pagination slots | loading, empty, error, no-permission |
| `SchemaForm.vue` | Schema-driven plugin/system forms | disabled/saving, validation errors |
| `DetailDrawer.vue` | Standard detail/edit drawer surface | loading, responsive drawer width |
| `ConfirmAction.vue` | Guarded command button for destructive or risky actions | disabled, loading, confirmation |
| `StateBlock.vue` | Reusable empty/error/forbidden block | empty, error, no-permission |

`index.ts` exports the UI Kit for explicit imports.

## Usage Rules

1. Use `PageShell` at the root of new admin or plugin pages.
2. Use `DataTable` for list pages unless a page needs highly custom `el-table` behavior.
3. Put destructive commands behind `ConfirmAction`.
4. Keep filters in `FilterBar` and repeated section commands in `PageToolbar`.
5. Cover loading, empty, error, no-permission, saving/destructive, and responsive states before marking a page complete.

## Validation

Run from `web/` after changing shared components:

```powershell
npm run typecheck
npm run build
```
