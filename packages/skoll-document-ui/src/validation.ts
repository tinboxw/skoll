import { documentKitMessages } from "./messages";
import type {
	DocumentDraft,
	DocumentFieldSchema,
	DocumentFormErrors,
	DocumentKitMessages,
	DocumentLocale,
	DocumentSchema,
	DocumentValue
} from "./types";

const decimalPattern = /^-?(0|[1-9][0-9]*)(\.[0-9]+)?$/;
const integerPattern = /^-?(0|[1-9][0-9]*)$/;
const currencyPattern = /^[A-Z]{3}$/;
const unitPattern = /^[A-Za-z][A-Za-z0-9._/%-]{0,31}$/;

export function emptyDocumentValue(field: DocumentFieldSchema): DocumentValue {
	const base: DocumentValue = { type: field.type };
	if (field.type === "boolean") return { ...base, value: "false" };
	if (field.type === "reference") return { ...base, reference: { type: field.referenceType || "record", id: "" } };
	return { ...base, value: "" };
}

export function createDocumentDraft(schema: DocumentSchema, identity: Pick<DocumentDraft, "id" | "number" | "title">): DocumentDraft {
	return {
		...identity,
		type: schema.key,
		schemaVersion: schema.version,
		header: Object.fromEntries(schema.header.map((field) => [field.key, emptyDocumentValue(field)])),
		lines: Object.fromEntries(schema.lines.map((line) => [
			line.key,
			Array.from({ length: line.minItems }, (_, index) => ({
				id: `${line.key}-${index + 1}`,
				values: Object.fromEntries(line.fields.map((field) => [field.key, emptyDocumentValue(field)]))
			}))
		])),
		tags: []
	};
}

export function validateDocumentDraft(
	schema: DocumentSchema,
	draft: DocumentDraft,
	locale: DocumentLocale = "zh-CN",
	overrides?: Partial<DocumentKitMessages>
): DocumentFormErrors {
	const messages = documentKitMessages(locale, overrides);
	const errors: DocumentFormErrors = {};
	if (!draft.number.trim()) errors.number = messages.fieldRequired(messages.number);
	if (!draft.title.trim()) errors.title = messages.fieldRequired(messages.title);
	for (const field of schema.header) {
		const error = validateDocumentValue(field, draft.header[field.key], messages);
		if (error) errors[`header.${field.key}`] = error;
	}
	for (const lineSchema of schema.lines) {
		const rows = draft.lines[lineSchema.key] || [];
		if (rows.length < lineSchema.minItems) errors[`lines.${lineSchema.key}`] = messages.lineMinimum(lineSchema.name, lineSchema.minItems);
		if (rows.length > lineSchema.maxItems) errors[`lines.${lineSchema.key}`] = messages.lineMaximum(lineSchema.name, lineSchema.maxItems);
		rows.forEach((line, index) => {
			for (const field of lineSchema.fields) {
				const error = validateDocumentValue(field, line.values[field.key], messages);
				if (error) errors[`lines.${lineSchema.key}.${index}.${field.key}`] = error;
			}
		});
	}
	return errors;
}

export function validateDocumentValue(field: DocumentFieldSchema, value: DocumentValue | undefined, messages: DocumentKitMessages): string {
	if (!value || value.type !== field.type) return messages.fieldInvalid(field.label);
	const text = field.type === "reference" ? value.reference?.id || "" : value.value || "";
	if (field.required && field.type !== "boolean" && text.trim() === "") return messages.fieldRequired(field.label);
	if (!field.required && text === "") return "";
	if (!validValueShape(field, value)) return messages.fieldInvalid(field.label);
	for (const rule of field.rules || []) {
		if (!passesRule(rule.type, rule.value, text)) return rule.message || messages.fieldInvalid(field.label);
	}
	return "";
}

function validValueShape(field: DocumentFieldSchema, value: DocumentValue): boolean {
	const text = value.value || "";
	switch (field.type) {
	case "integer":
		return integerPattern.test(text);
	case "decimal":
		return decimalPattern.test(text);
	case "money":
		return decimalPattern.test(text) && currencyPattern.test(value.currency || "");
	case "quantity":
		return decimalPattern.test(text) && unitPattern.test(value.unit || "");
	case "date":
		return /^\d{4}-\d{2}-\d{2}$/.test(text) && !Number.isNaN(Date.parse(`${text}T00:00:00Z`));
	case "datetime":
		return !Number.isNaN(Date.parse(text));
	case "reference": {
		const reference = value.reference;
		if (!reference) return false;
		return reference.type === field.referenceType && /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(reference.id);
	}
	case "json":
		try {
			JSON.parse(text);
			return true;
		} catch {
			return false;
		}
	default:
		return true;
	}
}

function passesRule(type: string, expected: string, value: string): boolean {
	switch (type) {
	case "min":
		return decimalPattern.test(value) && Number(value) >= Number(expected);
	case "max":
		return decimalPattern.test(value) && Number(value) <= Number(expected);
	case "min_length":
		return Array.from(value).length >= Number(expected);
	case "max_length":
		return Array.from(value).length <= Number(expected);
	case "pattern":
		try {
			return new RegExp(expected).test(value);
		} catch {
			return false;
		}
	default:
		return false;
	}
}
