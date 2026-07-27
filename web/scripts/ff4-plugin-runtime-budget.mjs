import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { gzipSync } from "node:zlib";

const root = process.cwd();
const dist = path.join(root, "dist");
const manifestPath = path.join(dist, ".vite", "manifest.json");
const budgetPath = path.join(root, "config", "frontend-quality-budgets.json");
const reportPath = path.join(root, "test-results", "ff4-plugin-runtime", "bundle.json");

if (!fs.existsSync(manifestPath)) {
	throw new Error("Vite manifest is missing. Run npm run build before check:plugin-runtime:bundle.");
}

const manifest = JSON.parse(fs.readFileSync(manifestPath, "utf8"));
const budget = JSON.parse(fs.readFileSync(budgetPath, "utf8")).pluginRuntime.bundle;
const chunk = manifest[budget.routeSource];
const failures = [];

if (!chunk) {
	failures.push(`${budget.routeSource}: missing from Vite manifest`);
}

const content = chunk ? fs.readFileSync(path.join(dist, chunk.file)) : Buffer.alloc(0);
const actual = {
	file: chunk?.file ?? null,
	dynamic: chunk?.isDynamicEntry === true,
	rawBytes: content.length,
	gzipBytes: gzipSync(content, { level: 9 }).length
};

if (!actual.dynamic) {
	failures.push(`${budget.routeSource}: plugin runtime is not emitted as a dynamic entry`);
}
if (actual.gzipBytes > budget.maxRouteGzipBytes) {
	failures.push(`routeGzipBytes: ${actual.gzipBytes} > ${budget.maxRouteGzipBytes}`);
}

const report = {
	schemaVersion: 1,
	budget,
	actual,
	failures
};
fs.mkdirSync(path.dirname(reportPath), { recursive: true });
fs.writeFileSync(reportPath, JSON.stringify(report, null, 2) + "\n");
console.log(JSON.stringify(report, null, 2));
if (failures.length > 0) {
	throw new Error(`Plugin runtime bundle budget failed:\n${failures.join("\n")}`);
}
