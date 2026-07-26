import { gzipSync } from "node:zlib";
import { readdir, readFile, stat } from "node:fs/promises";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("../dist/", import.meta.url));
const limits = {
  entryRawBytes: 500 * 1024,
  initialJavaScriptGzipBytes: 200 * 1024,
  asyncChunkGzipBytes: 8 * 1024,
  totalJavaScriptGzipBytes: 210 * 1024,
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
const gzipSizes = new Map(await Promise.all(javascript.map(async (path) => [relative(root, path).replaceAll("\\", "/"), gzipSync(await readFile(path)).byteLength])));
const javascriptGzip = [...gzipSizes.values()].reduce((sum, size) => sum + size, 0);
const cssGzip = (await Promise.all(css.map(async (path) => gzipSync(await readFile(path)).byteLength))).reduce((sum, size) => sum + size, 0);
const manifest = JSON.parse(await readFile(join(root, ".vite", "manifest.json"), "utf8"));
const entryManifest = manifest["index.html"];
if (!entryManifest?.isEntry) throw new Error("bundle budget: Vite entry manifest is missing");

const initialFiles = new Set();
function collectInitial(key) {
  const chunk = manifest[key];
  if (!chunk || initialFiles.has(chunk.file)) return;
  initialFiles.add(chunk.file);
  for (const imported of chunk.imports ?? []) collectInitial(imported);
}
collectInitial("index.html");

const initialJavaScriptGzip = [...initialFiles].reduce((sum, path) => sum + (gzipSizes.get(path) ?? 0), 0);
const asyncChunks = [...gzipSizes.entries()].filter(([path]) => !initialFiles.has(path));
const largestAsyncChunkGzip = Math.max(0, ...asyncChunks.map(([, size]) => size));
const report = {
  entry: relative(root, entry),
  entryBytes,
  initialJavaScriptGzip,
  largestAsyncChunkGzip,
  javascriptGzip,
  cssGzip,
  limits
};

if (entryBytes > limits.entryRawBytes) throw new Error(`bundle budget: entry ${entryBytes} > ${limits.entryRawBytes}`);
if (initialJavaScriptGzip > limits.initialJavaScriptGzipBytes) throw new Error(`bundle budget: initial JavaScript gzip ${initialJavaScriptGzip} > ${limits.initialJavaScriptGzipBytes}`);
if (largestAsyncChunkGzip > limits.asyncChunkGzipBytes) throw new Error(`bundle budget: async chunk gzip ${largestAsyncChunkGzip} > ${limits.asyncChunkGzipBytes}`);
if (javascriptGzip > limits.totalJavaScriptGzipBytes) throw new Error(`bundle budget: JavaScript gzip ${javascriptGzip} > ${limits.totalJavaScriptGzipBytes}`);
if (cssGzip > limits.totalCSSGzipBytes) throw new Error(`bundle budget: CSS gzip ${cssGzip} > ${limits.totalCSSGzipBytes}`);
console.log(JSON.stringify(report));
