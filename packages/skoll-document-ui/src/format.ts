import type { DocumentFieldSchema, DocumentKitMessages, DocumentValue } from "./types";

export function formatDocumentValue(
	field: DocumentFieldSchema,
	value: DocumentValue | undefined,
	labels: Pick<DocumentKitMessages, "yes" | "no">
): string {
	if (!value) return "-";
	if (field.type === "boolean") return value.value === "true" ? labels.yes : labels.no;
	if (field.type === "money") return [value.currency, value.value].filter(Boolean).join(" ") || "-";
	if (field.type === "quantity") return [value.value, value.unit].filter(Boolean).join(" ") || "-";
	if (field.type === "reference") return value.reference?.label || value.reference?.id || "-";
	if (field.type === "json") {
		try {
			return JSON.stringify(JSON.parse(value.value || "null"), null, 2);
		} catch {
			return value.value || "-";
		}
	}
	return value.value || "-";
}

export function documentStateTone(state: string): "success" | "warning" | "danger" | "info" {
	if (["approved", "completed", "active"].includes(state)) return "success";
	if (["rejected", "canceled", "failed"].includes(state)) return "danger";
	if (["pending", "running", "draft"].includes(state)) return "warning";
	return "info";
}

export function formatDocumentDate(value: string, locale = "zh-CN"): string {
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return value || "-";
	return new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" }).format(date);
}

export function formatFileSize(bytes: number): string {
	if (!Number.isFinite(bytes) || bytes < 0) return "-";
	if (bytes < 1024) return `${bytes} B`;
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
	return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}
