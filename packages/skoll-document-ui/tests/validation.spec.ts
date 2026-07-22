import { describe, expect, it } from "vitest";

import { documentTestSchema } from "./test-fixtures";
import type { DocumentSchema } from "../src/types";
import { createDocumentDraft, validateDocumentDraft } from "../src/validation";

describe("business document validation", () => {
	it("creates exact typed values and stable field paths", () => {
		const draft = createDocumentDraft(documentTestSchema, { id: "po-1", number: "PO-001", title: "Quarterly order" });
		const invalid = validateDocumentDraft(documentTestSchema, draft, "en-US");
		expect(Object.keys(invalid)).toEqual([
			"header.subject",
			"header.budget",
			"header.supplier",
			"lines.items.0.product",
			"lines.items.0.quantity"
		]);

		draft.header.subject = { type: "string", value: "Order" };
		draft.header.budget = { type: "money", value: "1200.50", currency: "CNY" };
		draft.header.supplier = { type: "reference", reference: { type: "supplier", id: "supplier-1", label: "Supplier One" } };
		draft.lines.items[0].values.product = { type: "string", value: "Drug A" };
		draft.lines.items[0].values.quantity = { type: "quantity", value: "12.5", unit: "box" };
		expect(validateDocumentDraft(documentTestSchema, draft, "en-US")).toEqual({});
	});

	it("rejects lossy or incomplete money, quantity, reference, and json values", () => {
		const schema: DocumentSchema = {
			...documentTestSchema,
			header: [
				{ key: "money", label: "Money", type: "money", required: true },
				{ key: "quantity", label: "Quantity", type: "quantity", required: true },
				{ key: "reference", label: "Reference", type: "reference", referenceType: "customer", required: true },
				{ key: "json", label: "JSON", type: "json", required: true }
			],
			lines: []
		};
		const draft = createDocumentDraft(schema, { id: "doc-1", number: "DOC-1", title: "Strict values" });
		draft.header.money = { type: "money", value: "1.2", currency: "cn" };
		draft.header.quantity = { type: "quantity", value: "1e3", unit: "box" };
		draft.header.reference = { type: "reference", reference: { type: "supplier", id: "bad id" } };
		draft.header.json = { type: "json", value: "{" };
		expect(Object.keys(validateDocumentDraft(schema, draft))).toEqual([
			"header.money",
			"header.quantity",
			"header.reference",
			"header.json"
		]);
	});
});
