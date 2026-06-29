import type { PluginConfigField, PluginConfigSchema } from "./types";

const supportedFieldTypes = new Set(["string", "textarea", "number", "boolean", "select"]);

export function normalizePluginConfigSchema(schema: unknown): PluginConfigSchema | null {
  if (!schema || typeof schema !== "object") {
    return null;
  }
  const row = schema as PluginConfigSchema;
  const fields = Array.isArray(row.fields)
    ? row.fields
      .filter((field): field is PluginConfigField => isConfigField(field))
      .map((field) => ({
        ...field,
        key: field.key.trim(),
        type: normalizeFieldType(field.type)
      }))
    : [];
  if (fields.length === 0) {
    return null;
  }
  return {
    ...row,
    fields
  };
}

export function applyConfigDefaults(config: Record<string, unknown>, schema: PluginConfigSchema | null): Record<string, unknown> {
  const out = { ...config };
  for (const field of schema?.fields ?? []) {
    if (!field.key || out[field.key] !== undefined) {
      continue;
    }
    out[field.key] = coerceConfigValue(field.default, field.type);
  }
  return out;
}

export function coerceConfigValue(value: unknown, type?: PluginConfigField["type"]): unknown {
  if (type === "boolean") {
    return value === true || value === "true" || value === "1";
  }
  if (type === "number") {
    const parsed = Number(value ?? 0);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return value ?? "";
}

export function serializeConfigValue(value: unknown): string {
  if (typeof value === "boolean") {
    return value ? "true" : "false";
  }
  if (value === null || value === undefined) {
    return "";
  }
  return String(value);
}

function isConfigField(value: unknown): value is PluginConfigField {
  if (!value || typeof value !== "object") {
    return false;
  }
  const field = value as PluginConfigField;
  return typeof field.key === "string" && field.key.trim() !== "" && normalizeFieldType(field.type) !== undefined;
}

function normalizeFieldType(type: PluginConfigField["type"] | undefined): PluginConfigField["type"] {
  if (!type) {
    return "string";
  }
  return supportedFieldTypes.has(type) ? type : undefined;
}
