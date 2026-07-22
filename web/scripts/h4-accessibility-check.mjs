import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { baseParse } from "@vue/compiler-dom";
import { parse as parseSFC } from "@vue/compiler-sfc";
import ts from "typescript";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const webRoot = path.resolve(scriptDir, "..");
const failures = [];
let iconButtonCount = 0;
let confirmationCount = 0;

function walkVueFiles(root, output = []) {
	for (const entry of fs.readdirSync(root, { withFileTypes: true })) {
		const fullPath = path.join(root, entry.name);
		if (entry.isDirectory()) walkVueFiles(fullPath, output);
		else if (entry.name.endsWith(".vue") && !entry.name.endsWith(".spec.vue")) output.push(fullPath);
	}
	return output;
}

const uiFiles = walkVueFiles(path.join(webRoot, "src")).sort();

function relative(filePath) {
	return path.relative(webRoot, filePath).replaceAll("\\", "/");
}

function propertyName(node) {
	if (ts.isIdentifier(node) || ts.isStringLiteral(node) || ts.isNumericLiteral(node)) return node.text;
	return "";
}

function templatePropName(prop) {
	if (prop.type === 6) return prop.name;
	if (prop.type === 7 && prop.arg?.type === 4) return prop.arg.content;
	return "";
}

function hasTemplateProp(node, name) {
	return (node.props || []).some((prop) => templatePropName(prop) === name);
}

function hasDirective(node, name) {
	return (node.props || []).some((prop) => prop.type === 7 && prop.name === name);
}

function hasVisibleButtonContent(node) {
	return (node.children || []).some((child) => {
		if (child.type === 2) return child.content.trim().length > 0;
		if (child.type === 5) return true;
		if (child.type === 1 && child.tag !== "template") return true;
		return child.type === 1 && hasVisibleButtonContent(child);
	});
}

function inspectTemplate(templateSource, filePath) {
	const ast = baseParse(templateSource);
	function visit(node) {
		if (node.type === 1) {
			if (node.tag === "el-button" && hasTemplateProp(node, "icon") && !hasVisibleButtonContent(node)) {
				iconButtonCount += 1;
				if (!hasTemplateProp(node, "aria-label")) {
					failures.push(`${relative(filePath)}:${node.loc.start.line} icon-only button requires aria-label`);
				}
			}
			if (node.tag === "img" && !hasTemplateProp(node, "alt")) {
				failures.push(`${relative(filePath)}:${node.loc.start.line} image requires alt text`);
			}
			if (node.tag === "iframe" && !hasTemplateProp(node, "title")) {
				failures.push(`${relative(filePath)}:${node.loc.start.line} iframe requires a localized title`);
			}
			const isNativeInteractive = ["a", "button", "input", "select", "textarea", "summary"].includes(node.tag);
			const isComponent = /[-A-Z]/.test(node.tag);
			if (hasDirective(node, "on") && !isNativeInteractive && !isComponent && !hasTemplateProp(node, "role")) {
				const click = (node.props || []).some((prop) => prop.type === 7 && prop.name === "on" && prop.arg?.type === 4 && prop.arg.content === "click");
				if (click) failures.push(`${relative(filePath)}:${node.loc.start.line} clickable ${node.tag} requires native semantics or role`);
			}
			for (const prop of node.props || []) {
				if (templatePropName(prop) === "tabindex" && prop.type === 6 && prop.value && Number(prop.value.content) > 0) {
					failures.push(`${relative(filePath)}:${node.loc.start.line} positive tabindex is not allowed`);
				}
			}
		}
		for (const child of node.children || []) visit(child);
	}
	visit(ast);
}

function inspectMessageBoxes(scriptSource, filePath) {
	const source = ts.createSourceFile(filePath, scriptSource, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
	function visit(node) {
		if (
			ts.isCallExpression(node)
			&& ts.isPropertyAccessExpression(node.expression)
			&& node.expression.expression.getText(source) === "ElMessageBox"
			&& ["confirm", "prompt"].includes(node.expression.name.text)
		) {
			confirmationCount += 1;
			const options = [...node.arguments].reverse().find(ts.isObjectLiteralExpression);
			const names = new Set((options?.properties || [])
				.filter(ts.isPropertyAssignment)
				.map((property) => propertyName(property.name)));
			const line = source.getLineAndCharacterOfPosition(node.getStart(source)).line + 1;
			if (!names.has("confirmButtonText")) failures.push(`${relative(filePath)}:${line} confirmation requires localized confirmButtonText`);
			if (!names.has("cancelButtonText")) failures.push(`${relative(filePath)}:${line} confirmation requires localized cancelButtonText`);
			if (node.expression.name.text === "prompt" && !names.has("inputValidator")) {
				failures.push(`${relative(filePath)}:${line} destructive prompt requires inputValidator`);
			}
		}
		ts.forEachChild(node, visit);
	}
	visit(source);
}

for (const filePath of uiFiles) {
	const sourceText = fs.readFileSync(filePath, "utf8");
	const { descriptor, errors } = parseSFC(sourceText, { filename: filePath });
	if (errors.length > 0) {
		failures.push(`${relative(filePath)} cannot be parsed: ${errors.join(", ")}`);
		continue;
	}
	if (descriptor.template) inspectTemplate(descriptor.template.content, filePath);
	for (const script of [descriptor.script?.content, descriptor.scriptSetup?.content]) {
		if (script) inspectMessageBoxes(script, filePath);
	}
}

const invariants = [
	["src/components/Common/DetailDrawer.vue", ["@closed=\"restoreFocus\"", "target.focus({ preventScroll: true })"]],
	["src/components/Common/StateBlock.vue", [":role=", ":aria-live=", "aria-atomic=\"true\""]],
	["src/components/Common/DataTable.vue", [":aria-label=\"tableLabel\"", ":label=\"t('common.actions')\""]],
	["src/components/Common/PageShell.vue", [":aria-labelledby=\"titleId\"", ":id=\"titleId\""]],
	["src/styles/global.scss", [":focus-visible", "prefers-reduced-motion: reduce"]]
];

for (const [relativePath, requiredTokens] of invariants) {
	const source = fs.readFileSync(path.join(webRoot, relativePath), "utf8");
	for (const token of requiredTokens) {
		if (!source.includes(token)) failures.push(`${relativePath} is missing accessibility invariant: ${token}`);
	}
	if (/outline\s*:\s*(?:none|0)\b/.test(source)) failures.push(`${relativePath} removes focus outlines`);
}

if (failures.length > 0) {
	console.error("H4 accessibility check failed:\n- " + failures.join("\n- "));
	process.exit(1);
}

console.log(`H4 accessibility check passed: ${uiFiles.length} UI files, ${iconButtonCount} named icon buttons, ${confirmationCount} guarded confirmations.`);
