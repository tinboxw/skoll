import type { PluginControlSnapshot } from "./api";

export type PluginRuntimeState = "loading" | "ready" | "degraded" | "crashed" | "disabled" | "forbidden" | "stale";

export function deriveRuntimeState(snapshot: PluginControlSnapshot | null, now = Date.now()): PluginRuntimeState {
	if (!snapshot) return "loading";
	if (snapshot.runtime.state !== "enabled") return "disabled";
	const staleAt = Date.parse(snapshot.staleAfter);
	if (Number.isFinite(staleAt) && now >= staleAt) return "stale";
	const health = snapshot.runtime.health;
	if (!health) return "degraded";
	if (health.status === "healthy" || health.status === "not_applicable") return "ready";
	if (["health_unreachable", "health_timeout", "health_cancelled"].includes(health.code)) return "crashed";
	return "degraded";
}

export function runtimeTagType(state: PluginRuntimeState): "success" | "warning" | "danger" | "info" {
	if (state === "ready") return "success";
	if (state === "crashed" || state === "forbidden") return "danger";
	if (state === "degraded" || state === "stale") return "warning";
	return "info";
}

export function formatTimestamp(value?: string): string {
	if (!value) return "-";
	const date = new Date(value);
	return Number.isNaN(date.getTime()) ? "-" : date.toLocaleString();
}
