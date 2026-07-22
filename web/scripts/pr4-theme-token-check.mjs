import fs from "node:fs";
import path from "node:path";

const sourceRoot = path.resolve("src");
const tokenFile = path.join(sourceRoot, "styles", "variables.scss");
const sourceExtensions = new Set([".ts", ".vue", ".scss", ".css"]);
const rawColorPattern = /#[0-9a-f]{3,8}\b|rgba?\s*\(|(?:linear|radial)-gradient\s*\(/gi;
const oldThemePattern = /ThemeMode|data-theme-mode|skoll\.ui\.theme|themeCompact/;

function collectFiles(directory) {
	return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
		const target = path.join(directory, entry.name);
		if (entry.isDirectory()) return collectFiles(target);
		return sourceExtensions.has(path.extname(entry.name)) ? [target] : [];
	});
}

const violations = [];
for (const file of collectFiles(sourceRoot)) {
	if (file === tokenFile) continue;
	const source = fs.readFileSync(file, "utf8");
	for (const match of source.matchAll(rawColorPattern)) {
		const line = source.slice(0, match.index).split(/\r?\n/).length;
		violations.push(`${path.relative(process.cwd(), file)}:${line} raw color or gradient ${match[0]}`);
	}
}

for (const file of collectFiles(sourceRoot).filter((candidate) => !candidate.endsWith(".spec.ts"))) {
	const source = fs.readFileSync(file, "utf8");
	if (oldThemePattern.test(source)) violations.push(`${path.relative(process.cwd(), file)} retains the removed theme mode contract`);
}

const tokens = fs.readFileSync(tokenFile, "utf8");
for (const marker of [
	':root[data-theme="light"]',
	':root[data-theme="dark"]',
	':root[data-density="compact"]',
	"--color-warning-border",
	"--color-code-surface",
	"--el-color-primary",
	"--el-bg-color-overlay",
	"--el-mask-color"
]) {
	if (!tokens.includes(marker)) violations.push(`src/styles/variables.scss is missing ${marker}`);
}

if (violations.length > 0) {
	console.error(violations.join("\n"));
	process.exit(1);
}

console.log(`PR4 theme token check passed: ${collectFiles(sourceRoot).length} frontend source files, current two-axis contract.`);
