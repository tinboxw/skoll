import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { gzipSync } from "node:zlib";

const root = process.cwd();
const dist = path.join(root, "dist");
const manifestPath = path.join(dist, ".vite", "manifest.json");
const budgetPath = path.join(root, "config", "frontend-quality-budgets.json");
const reportPath = path.join(root, "test-results", "bf3-plugin-control-center", "bundle.json");

if (!fs.existsSync(manifestPath)) {
	throw new Error("Vite manifest is missing. Run npm run build before check:plugin-center:bundle.");
}

const manifest = JSON.parse(fs.readFileSync(manifestPath, "utf8"));
const budgets = JSON.parse(fs.readFileSync(budgetPath, "utf8")).pluginControlCenter.bundle;
const failures = [];
const routes = budgets.routeSources.map((source) => {
	const chunk = manifest[source];
	if (!chunk) {
		failures.push(`${source}: missing from Vite manifest`);
		return { source, file: null, dynamic: false, rawBytes: 0, gzipBytes: 0 };
	}
	const content = fs.readFileSync(path.join(dist, chunk.file));
	const metric = {
		source,
		file: chunk.file,
		dynamic: chunk.isDynamicEntry === true,
		rawBytes: content.length,
		gzipBytes: gzipSync(content, { level: 9 }).length
	};
	if (!metric.dynamic) failures.push(`${source}: route is not emitted as a dynamic entry`);
	if (metric.gzipBytes > budgets.maxRouteGzipBytes) {
		failures.push(`${source}: ${metric.gzipBytes} > ${budgets.maxRouteGzipBytes} gzip bytes`);
	}
	return metric;
});

const totalRouteGzipBytes = routes.reduce((total, route) => total + route.gzipBytes, 0);
if (totalRouteGzipBytes > budgets.totalRouteGzipBytes) {
	failures.push(`totalRouteGzipBytes: ${totalRouteGzipBytes} > ${budgets.totalRouteGzipBytes}`);
}

const report = {
	schemaVersion: 1,
	budgets,
	actual: {
		routeCount: routes.length,
		dynamicRouteCount: routes.filter((route) => route.dynamic).length,
		maxRouteGzipBytes: Math.max(...routes.map((route) => route.gzipBytes)),
		totalRouteGzipBytes
	},
	routes,
	failures
};

fs.mkdirSync(path.dirname(reportPath), { recursive: true });
fs.writeFileSync(reportPath, JSON.stringify(report, null, 2) + "\n");
console.log(JSON.stringify(report, null, 2));
if (failures.length > 0) throw new Error(`Plugin control-center bundle budget failed:\n${failures.join("\n")}`);
