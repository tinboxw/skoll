import fs from "node:fs";
import path from "node:path";
import process from "node:process";

const root = path.resolve(process.cwd());

function read(relativePath) {
	return fs.readFileSync(path.join(root, relativePath), "utf8");
}

function requirePattern(content, pattern, message) {
	if (!pattern.test(content)) throw new Error(message);
}

const http = read("src/utils/api.ts");
const table = read("src/components/Common/DataTable.vue");
const app = read("src/App.vue");
const pluginRuntime = read("src/plugins/index.ts");
const audit = read("src/views/Audit/index.vue");
const workflow = read("src/views/Workflow/index.vue");
const todo = read("src/views/TodoCenter/index.vue");
const formBuilder = read("src/views/FormBuilder/index.vue");

requirePattern(http, /Omit<RequestInit,\s*"method">/, "GET requests must accept AbortSignal through RequestInit");
requirePattern(http, /error\.name === "AbortError"/, "aborted requests must remain distinguishable");
requirePattern(table, /<el-table-v2/, "heavy list rendering must use Element Plus virtual table");
requirePattern(table, /virtualized:\s*false/, "virtual rendering must remain an explicit reusable table option");
requirePattern(table, /<slot name="pagination"/, "the reusable table must expose server-pagination composition");
requirePattern(app, /<el-config-provider :locale="elementLocale">/, "Element Plus locale must follow the application locale");
requirePattern(pluginRuntime, /new AbortController\(\)/, "plugin route loading must cancel superseded requests");
requirePattern(pluginRuntime, /signal:\s*controller\.signal/, "plugin route loading must pass AbortSignal");
requirePattern(audit, /<el-pagination/, "audit records must expose server pagination");
for (const [name, view] of [["workflow", workflow], ["todo", todo], ["form builder", formBuilder]]) {
	requirePattern(view, /<DataTable/, `${name} must use the shared table contract`);
}

console.log("H5 large-list check passed: reusable virtual table, cancellable plugin loading, server pagination, and Element Plus locale binding.");
