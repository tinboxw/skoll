# Plugin Business Document UI Kit

`@skoll/document-ui` is the public frontend package for schema-driven business documents. It is independent of the Skoll host source tree and works in integrated or standalone plugin pages that receive the current theme and locale bridge.

## Package Boundary

Plugins install these peer dependencies:

```bash
npm install @skoll/document-ui element-plus lucide-vue-next vue
```

Application startup imports the control-system theme first, then the document kit:

```ts
import "element-plus/dist/index.css";
import "@skoll/document-ui/style.css";
```

Never import from `web/src/business-documents`, another plugin, or an internal Go package. The supported boundary is:

- Go: `pkg/pluginsdk` and `pkg/pluginclient`.
- Browser: `window.__SKOLL_HOST__` and `@skoll/document-ui`.
- Data: the current camelCase document, workflow, collaboration, search, print, and export contracts.

## Components

| Export | Responsibility |
| --- | --- |
| `DocumentList` | Scoped search controls, document states, stable cursor navigation, create/open/export commands |
| `DocumentForm` | Schema header and line editing, exact typed values, validation, draft/save/submit commands |
| `DocumentDetail` | Metadata, fields, lines, approval, attachments, comments, timeline, print/export commands |
| `DocumentApprovalPanel` | Approve, reject, delegate, withdraw, and cancel decision input |
| `DocumentAttachments` | Add, download, remove, and removed-file presentation |
| `DocumentComments` | Immutable attributed comment entry and history |
| `DocumentTimeline` | Ordered document activity history |
| `DocumentPrintView` | Permission-filtered, redaction-safe print rendering |
| `DocumentFieldInput` | Public low-level field editor used by custom plugin forms |

The components never call host APIs. A plugin loads data with `window.__SKOLL_HOST__.request`, passes current values as props, handles emitted commands, calls its backend, and then replaces the controlled state with the response.

## Composition Example

```vue
<script setup lang="ts">
import {
  DocumentDetail,
  type DocumentActionRequest,
  type DocumentRecord,
  type DocumentSchema,
  type WorkflowInstance
} from "@skoll/document-ui";

defineProps<{
  schema: DocumentSchema;
  document: DocumentRecord;
  workflow: WorkflowInstance;
}>();

async function act(request: DocumentActionRequest) {
  await window.__SKOLL_HOST__?.request("/v1/plugins/medical_oa/api/documents/action", {
    method: "POST",
    body: request
  });
}
</script>

<template>
  <DocumentDetail
    :schema="schema"
    :document="document"
    :workflow="workflow"
    :available-actions="['approve', 'reject', 'delegate']"
    locale="zh-CN"
    @action="act"
  />
</template>
```

## State And Security Rules

- Pass `loading`, `error`, and `forbidden` explicitly; do not turn permission failures into empty data.
- Use host-provided print payloads and `redactedFields`; never decide sensitive-field access in the browser.
- Keep decimal, money, quantity, and integer values as strings until the backend validates them.
- Convert selected date-time values to RFC3339 UTC; the kit emits ISO timestamps.
- Treat document and workflow versions as current optimistic-concurrency values after every mutation.
- Do not add compatibility payloads, alternate component paths, legacy state names, or fallback API calls.

## Acceptance

Run the package gate from `packages/skoll-document-ui`:

```bash
npm ci
npm run typecheck
npm test
npm run build
npm pack --dry-run
```

Run host bridge and visual acceptance from `web`:

```bash
npm run test:documents:browser
npm run typecheck
npm run test:components
npm run build
npm run check:bundle
```

The browser matrix covers desktop/mobile, light/dark, comfortable/compact, ready/loading/empty/error/forbidden, form, detail, and redaction-safe print states.
