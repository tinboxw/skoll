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

const api = read("src/pharma-oa/api.ts");
const http = read("src/utils/api.ts");
const table = read("src/components/Common/DataTable.vue");
const app = read("src/App.vue");
const employee = read("src/views/PharmaEmployee/index.vue");
const customer = read("src/views/PharmaCustomer/index.vue");

for (const resource of ["employees", "customers", "suppliers", "products"]) {
	requirePattern(api, new RegExp(`loadListPage<[^>]+>\\(\\"${resource}\\"`), `${resource} must use the bounded page loader`);
}

for (const field of ["total", "hasMore", "nextCursor", "sort"]) {
	requirePattern(api, new RegExp(`\\b${field}:`), `page contract missing ${field}`);
}

requirePattern(api, /listPageCacheTTL\s*=\s*10_000/, "page cache TTL must remain explicit");
requirePattern(api, /invalidateListPageCache\("employees"\)/, "employee writes must invalidate page cache");
requirePattern(api, /invalidateListPageCache\("customers"\)/, "customer writes must invalidate page cache");
requirePattern(http, /Omit<RequestInit,\s*"method">/, "GET requests must accept AbortSignal through RequestInit");
requirePattern(http, /error\.name === "AbortError"/, "aborted requests must remain distinguishable");
requirePattern(table, /<el-table-v2/, "heavy list rendering must use Element Plus virtual table");
requirePattern(app, /<el-config-provider :locale="elementLocale">/, "Element Plus locale must follow the application locale");

for (const [name, view] of [["employee", employee], ["customer", customer]]) {
	requirePattern(view, /new AbortController\(\)/, `${name} list must cancel superseded requests`);
	requirePattern(view, /offset:\s*\(currentPage\.value - 1\) \* pageSize\.value/, `${name} list must use server offsets`);
	requirePattern(view, /signal:\s*controller\.signal/, `${name} list must pass AbortSignal`);
	requirePattern(view, /<el-pagination/, `${name} list must expose pagination controls`);
	requirePattern(view, /virtualized/, `${name} list must enable virtual rendering`);
	if (/filteredRows/.test(view)) throw new Error(`${name} list must not filter the full result set in memory`);
}

console.log("H5 large-list check passed: 4 cached page clients, 2 cancellable virtualized views, default zh-CN locale binding.");
