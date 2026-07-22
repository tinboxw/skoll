import type { DocumentSchema } from "../src/types";

export const documentTestSchema: DocumentSchema = {
	key: "purchase_order",
	name: "Purchase order",
	version: 1,
	initialState: "draft",
	header: [
		{ key: "subject", label: "Subject", type: "string", required: true, rules: [{ type: "min_length", value: "3" }] },
		{ key: "budget", label: "Budget", type: "money", required: true, sensitive: true },
		{ key: "supplier", label: "Supplier", type: "reference", referenceType: "supplier", required: true }
	],
	lines: [{
		key: "items",
		name: "Items",
		minItems: 1,
		maxItems: 20,
		fields: [
			{ key: "product", label: "Product", type: "string", required: true },
			{ key: "quantity", label: "Quantity", type: "quantity", required: true, rules: [{ type: "min", value: "0.01" }] }
		]
	}],
	states: [
		{ key: "draft", name: "Draft", terminal: false },
		{ key: "pending", name: "Pending", terminal: false },
		{ key: "approved", name: "Approved", terminal: true }
	],
	actions: [{ key: "submit", name: "Submit", from: ["draft"], to: "pending", requiresComment: false }]
};
