import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { gzipSync } from "node:zlib";

const root = process.cwd();
const dist = path.join(root, "dist");
const manifestPath = path.join(dist, ".vite", "manifest.json");
const budgetPath = path.join(root, "config", "frontend-quality-budgets.json");
const reportPath = path.join(root, "test-results", "pr4-quality", "bundle.json");

if (!fs.existsSync(manifestPath)) {
	throw new Error("Vite manifest is missing. Run npm run build before check:bundle.");
}

const manifest = JSON.parse(fs.readFileSync(manifestPath, "utf8"));
const budgets = JSON.parse(fs.readFileSync(budgetPath, "utf8")).bundle;
const entryKey = Object.keys(manifest).find((key) => manifest[key].isEntry);
if (!entryKey) throw new Error("Vite manifest does not contain an application entry.");

const fileMetrics = new Map();
function measure(relativePath) {
	if (!relativePath || fileMetrics.has(relativePath)) return fileMetrics.get(relativePath);
	const content = fs.readFileSync(path.join(dist, relativePath));
	const metric = { file: relativePath, rawBytes: content.length, gzipBytes: gzipSync(content, { level: 9 }).length };
	fileMetrics.set(relativePath, metric);
	return metric;
}

const initialFiles = new Set();
function collectInitial(key) {
	const chunk = manifest[key];
	if (!chunk) return;
	if (chunk.file) initialFiles.add(chunk.file);
	for (const css of chunk.css || []) initialFiles.add(css);
	for (const importedKey of chunk.imports || []) collectInitial(importedKey);
}
collectInitial(entryKey);

const allFiles = new Set();
for (const chunk of Object.values(manifest)) {
	if (chunk.file) allFiles.add(chunk.file);
	for (const css of chunk.css || []) allFiles.add(css);
}
const metrics = [...allFiles].map(measure);
const js = metrics.filter((item) => item.file.endsWith(".js"));
const css = metrics.filter((item) => item.file.endsWith(".css"));
const initial = [...initialFiles].map(measure);
const entry = measure(manifest[entryKey].file);
const asyncJs = js.filter((item) => !initialFiles.has(item.file));
const sum = (items, field) => items.reduce((total, item) => total + item[field], 0);
const maximum = (items, field) => items.reduce((current, item) => Math.max(current, item[field]), 0);

const actual = {
	initialGzipBytes: sum(initial, "gzipBytes"),
	entryGzipBytes: entry.gzipBytes,
	asyncChunkGzipBytes: maximum(asyncJs, "gzipBytes"),
	anyJavaScriptRawBytes: maximum(js, "rawBytes"),
	totalJavaScriptGzipBytes: sum(js, "gzipBytes"),
	totalCssGzipBytes: sum(css, "gzipBytes"),
	minimumAsyncJavaScriptChunks: asyncJs.length
};

const failures = [];
for (const [name, limit] of Object.entries(budgets)) {
	const value = actual[name];
	if (name === "minimumAsyncJavaScriptChunks") {
		if (value < limit) failures.push(`${name}: ${value} < ${limit}`);
	} else if (value > limit) {
		failures.push(`${name}: ${value} > ${limit}`);
	}
}

const report = {
	schemaVersion: 1,
	entry: entry.file,
	budgets,
	actual,
	largestJavaScript: [...js].sort((left, right) => right.rawBytes - left.rawBytes).slice(0, 10),
	failures
};
fs.mkdirSync(path.dirname(reportPath), { recursive: true });
fs.writeFileSync(reportPath, JSON.stringify(report, null, 2) + "\n");
console.log(JSON.stringify(report, null, 2));
if (failures.length > 0) throw new Error(`Frontend bundle budget failed:\n${failures.join("\n")}`);
