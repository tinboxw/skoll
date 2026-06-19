const scenarios = [
	{
		id: "login",
		name: "Login",
		route: "/login",
		role: "admin",
		viewport: "desktop",
		action: "Submit valid admin credentials and wait for the authenticated shell.",
		expected: "The main admin layout is visible and protected navigation is reachable."
	},
	{
		id: "permission-denied",
		name: "Permission denied",
		route: "/permission",
		role: "restricted",
		viewport: "desktop",
		action: "Open a route or action outside the restricted role permissions.",
		expected: "The UI shows a denial state through route guard, menu/action state, or API denial feedback."
	},
	{
		id: "plugin-operation",
		name: "Plugin operation",
		route: "/plugin",
		role: "admin",
		viewport: "desktop",
		action: "Open plugin management and inspect one safe list/detail/config workflow.",
		expected: "The plugin page remains usable and operation state is visible."
	},
	{
		id: "audit-export",
		name: "Audit export",
		route: "/audit",
		role: "admin",
		viewport: "desktop",
		action: "Apply audit filters and trigger the export path.",
		expected: "The export control is reachable and success or failure feedback is visible."
	},
	{
		id: "narrow-navigation",
		name: "Narrow navigation",
		route: "/dashboard",
		role: "admin",
		viewport: "390x844",
		action: "Navigate authenticated routes at the minimum narrow viewport.",
		expected: "Navigation remains reachable and primary content does not overlap or disappear."
	}
];

const requiredEnv = [
	"SKOLL_E2E_BASE_URL",
	"SKOLL_E2E_ADMIN_USER",
	"SKOLL_E2E_ADMIN_PASSWORD",
	"SKOLL_E2E_RESTRICTED_USER",
	"SKOLL_E2E_RESTRICTED_PASSWORD"
];

const args = new Set(process.argv.slice(2));
const outputJson = args.has("--json");
const failOnBlocked = args.has("--fail-on-blocked");

const missingEnv = requiredEnv.filter((name) => !process.env[name]);
const result = missingEnv.length > 0 ? "Blocked" : "Ready";
const blockedReason =
	missingEnv.length > 0
		? `Missing required browser smoke environment: ${missingEnv.join(", ")}`
		: "Minimum browser smoke fixtures are present; run the Playwright implementation when available.";

const rows = scenarios.map((scenario) => ({
	...scenario,
	result,
	evidence: result === "Blocked" ? blockedReason : "Pending browser runner execution"
}));

if (outputJson) {
	console.log(JSON.stringify({ result, requiredEnv, missingEnv, scenarios: rows }, null, 2));
} else {
	console.log("# FE5 Browser Smoke Minimum Set");
	console.log("");
	console.log(`Result: ${result}`);
	console.log(`Evidence: ${blockedReason}`);
	console.log("");
	console.log("| Scenario | Route | Role | Viewport | Action | Expected | Result | Evidence |");
	console.log("|---|---|---|---|---|---|---|---|");
	for (const row of rows) {
		console.log(
			`| ${row.name} | ${row.route} | ${row.role} | ${row.viewport} | ${row.action} | ${row.expected} | ${row.result} | ${row.evidence} |`
		);
	}
}

if (failOnBlocked && result === "Blocked") {
	process.exitCode = 2;
}
