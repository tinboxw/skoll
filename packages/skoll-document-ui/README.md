# @skoll/document-ui

Schema-driven Element Plus components for approval-backed business documents in independent Skoll plugins.

## Install

```bash
npm install @skoll/document-ui element-plus lucide-vue-next vue
```

Import Element Plus theme CSS before the kit CSS:

```ts
import "element-plus/dist/index.css";
import "@skoll/document-ui/style.css";
```

## Use

```vue
<script setup lang="ts">
import { DocumentForm, createDocumentDraft, type DocumentSchema } from "@skoll/document-ui";

const schema: DocumentSchema = loadPluginDocumentSchema();
const draft = createDocumentDraft(schema, {
  id: crypto.randomUUID(),
  number: "PO-20260723-001",
  title: "Purchase order"
});
</script>

<template>
  <DocumentForm :schema="schema" :model-value="draft" locale="zh-CN" />
</template>
```

The package exports list, form, detail, approval, attachment, comment, timeline, and redaction-safe print components. Components are controlled and emit commands; the plugin owns API calls, permission decisions, uploads, workflow actions, export jobs, and error recovery.

Do not import files from the Skoll host `web/src` tree. This package and the public host bridge are the frontend plugin boundary.

## Verify

From `packages/skoll-document-ui`:

```bash
npm ci
npm run typecheck
npm test
npm run build
npm pack --dry-run
```
