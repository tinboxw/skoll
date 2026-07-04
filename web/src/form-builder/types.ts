export type FormFieldType =
	| "string"
	| "textarea"
	| "number"
	| "boolean"
	| "date"
	| "datetime"
	| "select"
	| "multi_select"
	| "dictionary"
	| "user"
	| "department"
	| "attachment"
	| "detail_table";

export type FormOption = {
	label: string;
	value: string;
};

export type AttachmentConfig = {
	maxFiles: number;
	maxSizeMB: number;
	accept: string[];
	required?: boolean;
};

export type DetailTableConfig = {
	minRows: number;
	maxRows?: number;
	columns: FormField[];
};

export type FormField = {
	key: string;
	label: string;
	type: FormFieldType;
	required?: boolean;
	dictionary?: string;
	options?: FormOption[];
	attachment?: AttachmentConfig;
	detailTable?: DetailTableConfig;
	displayOrder?: number;
};

export type FormSchema = {
	id: string;
	key: string;
	name: string;
	version: number;
	businessType: string;
	description?: string;
	fields: FormField[];
	createdAt: string;
	updatedAt: string;
};

export type FormSchemaDraft = Omit<FormSchema, "createdAt" | "updatedAt"> & {
	createdAt?: string;
	updatedAt?: string;
};

export const FORM_SCHEMA_STORAGE_KEY = "skoll.formBuilder.schemas";

const FIELD_TYPES = new Set<FormFieldType>([
	"string",
	"textarea",
	"number",
	"boolean",
	"date",
	"datetime",
	"select",
	"multi_select",
	"dictionary",
	"user",
	"department",
	"attachment",
	"detail_table"
]);

export const FORM_FIELD_TYPE_OPTIONS: Array<{ label: string; value: FormFieldType }> = [
	{ label: "Text", value: "string" },
	{ label: "Textarea", value: "textarea" },
	{ label: "Number", value: "number" },
	{ label: "Switch", value: "boolean" },
	{ label: "Date", value: "date" },
	{ label: "Date time", value: "datetime" },
	{ label: "Select", value: "select" },
	{ label: "Multi select", value: "multi_select" },
	{ label: "Dictionary", value: "dictionary" },
	{ label: "User", value: "user" },
	{ label: "Department", value: "department" },
	{ label: "Attachment", value: "attachment" },
	{ label: "Detail table", value: "detail_table" }
];

export function loadFormSchemas(): FormSchema[] {
	const raw = localStorage.getItem(FORM_SCHEMA_STORAGE_KEY);
	if (!raw) {
		return [];
	}
	const parsed = JSON.parse(raw) as unknown;
	if (!Array.isArray(parsed)) {
		throw new Error("Stored form schema list is not an array.");
	}
	return parsed.map(normalizeSchema).sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
}

export function saveFormSchemas(schemas: FormSchema[]): void {
	localStorage.setItem(FORM_SCHEMA_STORAGE_KEY, JSON.stringify(schemas.map(normalizeSchema)));
}

export function upsertFormSchema(draft: FormSchemaDraft): FormSchema {
	const errors = validateFormSchema(draft);
	if (errors.length > 0) {
		throw new Error(errors[0]);
	}
	const now = new Date().toISOString();
	const schema = normalizeSchema({
		...draft,
		createdAt: draft.createdAt || now,
		updatedAt: now
	});
	const next = loadFormSchemas().filter((item) => item.id !== schema.id);
	saveFormSchemas([schema, ...next]);
	return schema;
}

export function deleteFormSchema(id: string): void {
	saveFormSchemas(loadFormSchemas().filter((item) => item.id !== id));
}

export function validateFormSchema(schema: FormSchemaDraft): string[] {
	const errors: string[] = [];
	if (!schema.id.trim()) {
		errors.push("Form ID is required.");
	}
	if (!schema.key.trim()) {
		errors.push("Form key is required.");
	}
	if (!/^[a-z][a-z0-9.:-]*$/.test(schema.key.trim())) {
		errors.push("Form key must start with a lowercase letter and use lowercase letters, numbers, dot, colon, or dash.");
	}
	if (!schema.name.trim()) {
		errors.push("Form name is required.");
	}
	if (!Number.isInteger(schema.version) || schema.version < 1) {
		errors.push("Version must be a positive integer.");
	}
	if (!schema.businessType.trim()) {
		errors.push("Business type is required.");
	}
	if (schema.fields.length === 0) {
		errors.push("Add at least one field.");
	}

	const seen = new Set<string>();
	for (const field of schema.fields) {
		const fieldErrors = validateField(field, "Field");
		errors.push(...fieldErrors);
		if (seen.has(field.key)) {
			errors.push(`Duplicate field key: ${field.key}.`);
		}
		seen.add(field.key);
	}
	return errors;
}

export function cloneFormSchema(schema: FormSchema): FormSchema {
	return normalizeSchema(JSON.parse(JSON.stringify(schema)) as FormSchema);
}

export function createBlankSchema(): FormSchema {
	const now = new Date().toISOString();
	return {
		id: createFormID("form"),
		key: "oa.request",
		name: "OA Request",
		version: 1,
		businessType: "oa.request",
		description: "",
		fields: [],
		createdAt: now,
		updatedAt: now
	};
}

export function createPurchaseRequestSchema(): FormSchema {
	const now = new Date().toISOString();
	return {
		id: createFormID("form"),
		key: "oa.purchase.request",
		name: "Purchase Request",
		version: 1,
		businessType: "oa.purchase",
		description: "Minimal purchase approval form for workflow launch.",
		fields: [
			{ key: "applicant", label: "Applicant", type: "user", required: true, displayOrder: 10 },
			{ key: "department", label: "Department", type: "department", required: true, displayOrder: 20 },
			{ key: "supplier", label: "Supplier", type: "string", required: true, displayOrder: 30 },
			{ key: "amount", label: "Estimated amount", type: "number", required: true, displayOrder: 40 },
			{
				key: "items",
				label: "Purchase lines",
				type: "detail_table",
				required: true,
				displayOrder: 50,
				detailTable: {
					minRows: 1,
					maxRows: 20,
					columns: [
						{ key: "product", label: "Product", type: "string", required: true },
						{ key: "quantity", label: "Quantity", type: "number", required: true },
						{ key: "remark", label: "Remark", type: "string" }
					]
				}
			},
			{
				key: "attachments",
				label: "Attachments",
				type: "attachment",
				displayOrder: 60,
				attachment: { maxFiles: 5, maxSizeMB: 20, accept: [".pdf", ".jpg", ".png"] }
			}
		],
		createdAt: now,
		updatedAt: now
	};
}

export function createFormID(prefix: string): string {
	const random = typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`;
	return `${prefix}-${random}`;
}

function normalizeSchema(input: FormSchemaDraft): FormSchema {
	return {
		id: String(input.id || "").trim(),
		key: String(input.key || "").trim(),
		name: String(input.name || "").trim(),
		version: Number(input.version || 1),
		businessType: String(input.businessType || "").trim(),
		description: String(input.description || "").trim(),
		fields: [...input.fields].map(normalizeField).sort((a, b) => (a.displayOrder || 0) - (b.displayOrder || 0)),
		createdAt: String(input.createdAt || new Date().toISOString()),
		updatedAt: String(input.updatedAt || new Date().toISOString())
	};
}

function normalizeField(input: FormField): FormField {
	const type = FIELD_TYPES.has(input.type) ? input.type : "string";
	return {
		key: String(input.key || "").trim(),
		label: String(input.label || "").trim(),
		type,
		required: Boolean(input.required),
		dictionary: String(input.dictionary || "").trim(),
		options: Array.isArray(input.options) ? input.options.map((item) => ({ label: String(item.label || "").trim(), value: String(item.value || "").trim() })).filter((item) => item.label && item.value) : [],
		attachment: input.attachment ? {
			maxFiles: Math.max(1, Number(input.attachment.maxFiles || 1)),
			maxSizeMB: Math.max(1, Number(input.attachment.maxSizeMB || 1)),
			accept: Array.isArray(input.attachment.accept) ? input.attachment.accept.map((item) => String(item).trim()).filter(Boolean) : [],
			required: Boolean(input.attachment.required)
		} : undefined,
		detailTable: input.detailTable ? {
			minRows: Math.max(0, Number(input.detailTable.minRows || 0)),
			maxRows: input.detailTable.maxRows ? Math.max(1, Number(input.detailTable.maxRows)) : undefined,
			columns: input.detailTable.columns.map((item) => normalizeField({ ...item, type: item.type === "detail_table" ? "string" : item.type }))
		} : undefined,
		displayOrder: Number(input.displayOrder || 0)
	};
}

function validateField(field: FormField, labelPrefix: string): string[] {
	const errors: string[] = [];
	if (!field.key.trim()) {
		errors.push(`${labelPrefix} key is required.`);
	}
	if (!/^[a-z][a-z0-9_]*$/.test(field.key.trim())) {
		errors.push(`${labelPrefix} key must start with a lowercase letter and use lowercase letters, numbers, or underscore.`);
	}
	if (!field.label.trim()) {
		errors.push(`${labelPrefix} label is required.`);
	}
	if (!FIELD_TYPES.has(field.type)) {
		errors.push(`${labelPrefix} type is invalid.`);
	}
	if ((field.type === "select" || field.type === "multi_select") && (!field.options || field.options.length === 0)) {
		errors.push(`${field.label || field.key} requires options.`);
	}
	if (field.type === "dictionary" && !field.dictionary?.trim()) {
		errors.push(`${field.label || field.key} requires a dictionary code.`);
	}
	if (field.type === "attachment" && !field.attachment) {
		errors.push(`${field.label || field.key} requires attachment settings.`);
	}
	if (field.type === "detail_table") {
		if (!field.detailTable || field.detailTable.columns.length === 0) {
			errors.push(`${field.label || field.key} requires detail table columns.`);
		} else {
			const seen = new Set<string>();
			for (const column of field.detailTable.columns) {
				if (column.type === "detail_table") {
					errors.push("Nested detail tables are not supported.");
				}
				errors.push(...validateField(column, "Detail column"));
				if (seen.has(column.key)) {
					errors.push(`Duplicate detail column key: ${column.key}.`);
				}
				seen.add(column.key);
			}
		}
	}
	return errors;
}
