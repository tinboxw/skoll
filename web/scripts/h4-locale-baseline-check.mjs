import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { baseParse } from "@vue/compiler-dom";
import { parse as parseSFC } from "@vue/compiler-sfc";
import ts from "typescript";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const webRoot = path.resolve(scriptDir, "..");
const repoRoot = path.resolve(webRoot, "..");
const baselinePath = path.join(webRoot, "i18n", "pharma-oa-baseline.json");
const localeSourcePath = path.join(webRoot, "src", "i18n", "index.ts");
const baseline = JSON.parse(fs.readFileSync(baselinePath, "utf8"));
const failures = [];

const userFacingAttributes = new Set([
	"aria-label", "cancel-label", "confirm-label", "description", "empty-text", "help", "label", "placeholder", "title"
]);
const userFacingProperties = new Set([
	"cancelLabel", "confirmLabel", "description", "emptyText", "help", "label", "message", "placeholder", "title"
]);

function propertyName(node) {
	if (ts.isIdentifier(node) || ts.isStringLiteral(node) || ts.isNumericLiteral(node)) return node.text;
	return "";
}

function extractMessages(sourceText) {
	const source = ts.createSourceFile(localeSourcePath, sourceText, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
	let messagesObject;
	function visit(node) {
		if (ts.isVariableDeclaration(node) && ts.isIdentifier(node.name) && node.name.text === "messages" && ts.isObjectLiteralExpression(node.initializer)) {
			messagesObject = node.initializer;
		}
		ts.forEachChild(node, visit);
	}
	visit(source);
	if (!messagesObject) throw new Error("messages locale dictionary was not found");

	const locales = new Map();
	for (const localeProperty of messagesObject.properties) {
		if (!ts.isPropertyAssignment(localeProperty) || !ts.isObjectLiteralExpression(localeProperty.initializer)) continue;
		const locale = propertyName(localeProperty.name);
		const keys = [];
		for (const item of localeProperty.initializer.properties) {
			if (ts.isPropertyAssignment(item)) keys.push(propertyName(item.name));
		}
		locales.set(locale, keys);
	}
	return locales;
}

function walkFiles(root, extensions, output = []) {
	for (const entry of fs.readdirSync(root, { withFileTypes: true })) {
		const fullPath = path.join(root, entry.name);
		if (entry.isDirectory()) walkFiles(fullPath, extensions, output);
		else if (extensions.has(path.extname(entry.name))) output.push(fullPath);
	}
	return output;
}

function translationKeys(sourceText) {
	const keys = new Set();
	const pattern = /\b(?:t|translate)\(\s*(["'])(.*?)\1/g;
	for (const match of sourceText.matchAll(pattern)) keys.add(match[2]);
	return keys;
}

function isHumanCopy(value) {
	const text = value.trim();
	return text.length > 1 && /[\p{L}\p{Script=Han}]/u.test(text) && !/^[-_.:/\w]+$/.test(text);
}

function collectTemplateCopy(templateSource) {
	const values = [];
	const ast = baseParse(templateSource);
	function visit(node) {
		if (node.type === 2 && isHumanCopy(node.content)) values.push(node.content.trim());
		if (node.type === 1) {
			for (const prop of node.props || []) {
				if (prop.type === 6 && userFacingAttributes.has(prop.name) && prop.value && isHumanCopy(prop.value.content)) {
					values.push(prop.value.content.trim());
				}
			}
		}
		for (const child of node.children || []) visit(child);
	}
	visit(ast);
	return values;
}

function stringValue(node) {
	if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) return node.text;
	return "";
}

function collectScriptCopy(scriptSource, filePath) {
	const values = [];
	const source = ts.createSourceFile(filePath, scriptSource, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
	function visit(node) {
		if (ts.isPropertyAssignment(node) && userFacingProperties.has(propertyName(node.name))) {
			const value = stringValue(node.initializer);
			if (isHumanCopy(value)) values.push(value.trim());
		}
		if (ts.isCallExpression(node) && ts.isPropertyAccessExpression(node.expression) && node.expression.expression.getText(source) === "ElMessage") {
			const value = node.arguments.length > 0 ? stringValue(node.arguments[0]) : "";
			if (isHumanCopy(value)) values.push(value.trim());
		}
		if (ts.isBinaryExpression(node) && node.operatorToken.kind === ts.SyntaxKind.EqualsToken && /(?:error|message)\.value$/i.test(node.left.getText(source))) {
			const value = stringValue(node.right);
			if (isHumanCopy(value)) values.push(value.trim());
		}
		ts.forEachChild(node, visit);
	}
	visit(source);
	return values;
}

function inspectView(relativePath) {
	const absolutePath = path.join(webRoot, relativePath);
	const sourceText = fs.readFileSync(absolutePath, "utf8");
	const { descriptor, errors } = parseSFC(sourceText, { filename: absolutePath });
	if (errors.length > 0) throw new Error(`${relativePath} cannot be parsed: ${errors.join(", ")}`);
	const keys = translationKeys(sourceText);
	const templateCopy = descriptor.template ? collectTemplateCopy(descriptor.template.content) : [];
	const scriptCopy = [descriptor.script?.content || "", descriptor.scriptSetup?.content || ""]
		.flatMap((source) => source ? collectScriptCopy(source, absolutePath) : []);
	return {
		translationKeyCount: keys.size,
		hardcodedCopyCount: templateCopy.length + scriptCopy.length
	};
}

function compareLists(label, actual, expected) {
	if (JSON.stringify(actual) !== JSON.stringify(expected)) failures.push(`${label}: expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
}

const localeSource = fs.readFileSync(localeSourcePath, "utf8");
const localeMessages = extractMessages(localeSource);
const actualLocales = [...localeMessages.keys()];
compareLists("supported locale dictionaries", actualLocales, baseline.supportedLocales);
if (!localeSource.includes(`export const DEFAULT_LOCALE: Locale = "${baseline.defaultLocale}";`) || !/return\s+DEFAULT_LOCALE\s*;/.test(localeSource)) {
	failures.push(`default locale must be declared and returned as ${baseline.defaultLocale}`);
}

const baselineKeys = new Set(localeMessages.get(baseline.defaultLocale) || []);
for (const locale of baseline.supportedLocales) {
	const keys = localeMessages.get(locale) || [];
	const duplicates = keys.filter((key, index) => keys.indexOf(key) !== index);
	if (duplicates.length > 0) failures.push(`${locale} contains duplicate keys: ${[...new Set(duplicates)].join(", ")}`);
	const missing = [...baselineKeys].filter((key) => !keys.includes(key));
	const extra = keys.filter((key) => !baselineKeys.has(key));
	if (missing.length > 0 || extra.length > 0) failures.push(`${locale} key parity mismatch; missing=${missing.join(",") || "none"}; extra=${extra.join(",") || "none"}`);
}

const referencedKeys = new Set();
for (const filePath of walkFiles(path.join(webRoot, "src"), new Set([".ts", ".vue"]))) {
	for (const key of translationKeys(fs.readFileSync(filePath, "utf8"))) referencedKeys.add(key);
}
const missingReferences = [...referencedKeys].filter((key) => !baselineKeys.has(key)).sort();
if (missingReferences.length > 0) failures.push(`translation references missing from locale dictionaries: ${missingReferences.join(", ")}`);

const actualViewPaths = fs.readdirSync(path.join(webRoot, "src", "views"), { withFileTypes: true })
	.filter((entry) => entry.isDirectory() && entry.name.startsWith("Pharma"))
	.map((entry) => `src/views/${entry.name}/index.vue`)
	.filter((relativePath) => fs.existsSync(path.join(webRoot, relativePath)))
	.sort();
const baselineViewPaths = baseline.views.map((view) => view.path).sort();
compareLists("Pharma OA view inventory", actualViewPaths, baselineViewPaths);

for (const expected of baseline.views) {
	const actual = inspectView(expected.path);
	if (actual.translationKeyCount !== expected.translationKeyCount || actual.hardcodedCopyCount !== expected.hardcodedCopyCount) {
		failures.push(`${expected.path} inventory changed: expected keys/copy ${expected.translationKeyCount}/${expected.hardcodedCopyCount}, got ${actual.translationKeyCount}/${actual.hardcodedCopyCount}`);
	}
	if (expected.status === "hardcoded" && actual.translationKeyCount !== 0) failures.push(`${expected.path} is marked hardcoded but now references locale keys`);
	if (expected.status === "localized" && actual.hardcodedCopyCount !== 0) failures.push(`${expected.path} is marked localized but still has hard-coded copy`);
}

const manifestPath = path.join(repoRoot, baseline.pluginManifest);
const manifest = fs.readFileSync(manifestPath, "utf8");
for (const required of ["name_zh_cn:", "name_en_us:", "label_zh_cn:", "label_en_us:", "  - zh-CN", "  - en-US"]) {
	if (!manifest.includes(required)) failures.push(`${baseline.pluginManifest} is missing bilingual plugin bridge declaration: ${required}`);
}

if (failures.length > 0) {
	console.error("H4 locale baseline check failed:\n- " + failures.join("\n- "));
	process.exit(1);
}

console.log(`H4 locale baseline check passed: ${baselineKeys.size} keys, ${referencedKeys.size} references, ${baseline.views.length} Pharma OA views, default ${baseline.defaultLocale}.`);
