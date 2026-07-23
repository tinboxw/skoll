import { gzipSync } from "node:zlib";
import { readdir, readFile, stat } from "node:fs/promises";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("../dist/", import.meta.url));
const limits = {
  entryRawBytes: 500 * 1024,
  totalJavaScriptGzipBytes: 200 * 1024,
  totalCSSGzipBytes: 25 * 1024
};

async function files(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const nested = await Promise.all(entries.map((entry) => {
    const path = join(directory, entry.name);
    return entry.isDirectory() ? files(path) : [path];
  }));
  return nested.flat();
}

const assets = await files(root);
const javascript = assets.filter((path) => path.endsWith(".js"));
const css = assets.filter((path) => path.endsWith(".css"));
const entry = javascript.find((path) => /[/\\]index-[^/\\]+\.js$/.test(path));
if (!entry) throw new Error("bundle budget: application entry chunk is missing");

const entryBytes = (await stat(entry)).size;
const javascriptGzip = (await Promise.all(javascript.map(async (path) => gzipSync(await readFile(path)).byteLength))).reduce((sum, size) => sum + size, 0);
const cssGzip = (await Promise.all(css.map(async (path) => gzipSync(await readFile(path)).byteLength))).reduce((sum, size) => sum + size, 0);
const report = { entry: relative(root, entry), entryBytes, javascriptGzip, cssGzip, limits };

if (entryBytes > limits.entryRawBytes) throw new Error(`bundle budget: entry ${entryBytes} > ${limits.entryRawBytes}`);
if (javascriptGzip > limits.totalJavaScriptGzipBytes) throw new Error(`bundle budget: JavaScript gzip ${javascriptGzip} > ${limits.totalJavaScriptGzipBytes}`);
if (cssGzip > limits.totalCSSGzipBytes) throw new Error(`bundle budget: CSS gzip ${cssGzip} > ${limits.totalCSSGzipBytes}`);
console.log(JSON.stringify(report));
