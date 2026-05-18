const DEFAULT_API_BASE_PREFIX = "/skoll";

function normalizeAPIPrefix(raw: string): string {
  const trimmed = raw.trim();
  if (trimmed === "") {
    return DEFAULT_API_BASE_PREFIX;
  }
  const withSlash = trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
  const normalized = withSlash.replace(/\/+$/, "");
  return normalized === "" ? DEFAULT_API_BASE_PREFIX : normalized;
}

function resolveInjectedPrefix(): string {
  if (typeof __SKOLL_API_BASE_PREFIX__ === "string" && __SKOLL_API_BASE_PREFIX__.trim() !== "") {
    return __SKOLL_API_BASE_PREFIX__;
  }
  return DEFAULT_API_BASE_PREFIX;
}

export const API_BASE_PREFIX = normalizeAPIPrefix(resolveInjectedPrefix());
