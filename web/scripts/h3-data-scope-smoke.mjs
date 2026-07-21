const baseUrl = (process.env.SKOLL_API_BASE || "http://127.0.0.1:8080").replace(/\/$/, "");
const apiPrefix = (process.env.SKOLL_API_BASE_PREFIX || "/skoll").replace(/\/+$/, "") || "/skoll";
const password = process.env.SKOLL_SCOPE_MATRIX_PASSWORD || "Scope@123456";

const matrix = [
	{ account: "scope_self", scope: "self", include: ["SCOPE-SELF"], exclude: ["SCOPE-DEPT", "SCOPE-CHILD", "SCOPE-OTHER"] },
	{ account: "scope_department", scope: "department", include: ["SCOPE-SELF", "SCOPE-DEPT"], exclude: ["SCOPE-CHILD", "SCOPE-OTHER"] },
	{ account: "scope_tree", scope: "department_tree", include: ["SCOPE-SELF", "SCOPE-DEPT", "SCOPE-CHILD"], exclude: ["SCOPE-OTHER"] },
	{ account: "scope_all", scope: "all", include: ["SCOPE-SELF", "SCOPE-DEPT", "SCOPE-CHILD", "SCOPE-OTHER"], exclude: [] },
	{ account: "scope_denied", denied: true, include: [], exclude: [] }
];

async function api(path, options = {}) {
	return fetch(`${baseUrl}${apiPrefix}${path}`, options);
}

async function login(account) {
	const response = await api("/v1/auth/login", {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ account, password })
	});
	if (!response.ok) {
		throw new Error(`${account} login returned ${response.status}`);
	}
	const payload = await response.json();
	const token = payload?.data?.token;
	if (typeof token !== "string" || token === "") {
		throw new Error(`${account} login did not return a token`);
	}
	return token;
}

async function verifyIdentity(entry) {
	const token = await login(entry.account);
	const headers = { Authorization: `Bearer ${token}` };
	const scopeResponse = await api("/v1/rbac/data-scope?resource=pharma_oa.customer&action=read", { headers });
	const customerResponse = await api("/v1/plugins/pharma_oa/api/customers?includeAll=true&organizationId=scope-org-other&ownerId=scope-user-scope_all", { headers });
	if (entry.denied) {
		if (scopeResponse.status !== 403 || customerResponse.status !== 403) {
			throw new Error(`${entry.account} expected 403/403, got ${scopeResponse.status}/${customerResponse.status}`);
		}
		return { account: entry.account, denied: true };
	}
	if (!scopeResponse.ok || !customerResponse.ok) {
		throw new Error(`${entry.account} scope/list returned ${scopeResponse.status}/${customerResponse.status}`);
	}
	const scopePayload = await scopeResponse.json();
	const customerPayload = await customerResponse.json();
	if (scopePayload?.data?.scope !== entry.scope) {
		throw new Error(`${entry.account} expected ${entry.scope}, got ${scopePayload?.data?.scope}`);
	}
	const codes = (customerPayload?.data?.items || []).map((item) => item.code);
	for (const code of entry.include) {
		if (!codes.includes(code)) {
			throw new Error(`${entry.account} is missing ${code}: ${codes.join(",")}`);
		}
	}
	for (const code of entry.exclude) {
		if (codes.includes(code)) {
			throw new Error(`${entry.account} leaked ${code}: ${codes.join(",")}`);
		}
	}
	return { account: entry.account, scope: entry.scope, codes };
}

async function main() {
	const results = [];
	for (const entry of matrix) {
		results.push(await verifyIdentity(entry));
	}
	console.log("H3 data-scope matrix smoke passed", results);
}

main().catch((error) => {
	console.error("H3 data-scope matrix smoke failed:", error instanceof Error ? error.message : String(error));
	process.exit(1);
});
