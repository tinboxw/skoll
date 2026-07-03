import http from "k6/http";
import { check, group, sleep } from "k6";
import { Trend } from "k6/metrics";

const baseURL = (__ENV.SKOLL_BASE_URL || "http://127.0.0.1:18080/skoll").replace(/\/$/, "");
const adminUser = __ENV.SKOLL_ADMIN_USER || "admin";
const adminPassword = __ENV.SKOLL_ADMIN_PASSWORD || "admin123";

const loginLatency = new Trend("skoll_login_latency", true);
const userListLatency = new Trend("skoll_user_list_latency", true);
const roleListLatency = new Trend("skoll_role_list_latency", true);
const pluginListLatency = new Trend("skoll_plugin_list_latency", true);
const auditQueryLatency = new Trend("skoll_audit_query_latency", true);

export const options = {
	scenarios: {
		baseline: {
			executor: "constant-vus",
			vus: Number(__ENV.SKOLL_PERF_VUS || 5),
			duration: __ENV.SKOLL_PERF_DURATION || "1m"
		}
	},
	thresholds: {
		http_req_failed: ["rate<0.05"],
		http_req_duration: ["p(95)<1000", "p(99)<2000"],
		skoll_login_latency: ["p(95)<1000", "p(99)<2000"],
		skoll_user_list_latency: ["p(95)<800", "p(99)<1500"],
		skoll_role_list_latency: ["p(95)<800", "p(99)<1500"],
		skoll_plugin_list_latency: ["p(95)<800", "p(99)<1500"],
		skoll_audit_query_latency: ["p(95)<1200", "p(99)<2500"]
	}
};

function endpoint(path) {
	return `${baseURL}${path}`;
}

function authHeaders(token) {
	const headers = { "Content-Type": "application/json" };
	if (token) {
		headers.Authorization = `Bearer ${token}`;
	}
	return { headers };
}

function extractToken(response) {
	try {
		const body = response.json();
		return body?.data?.token || body?.data?.accessToken || body?.token || body?.accessToken || "";
	} catch {
		return "";
	}
}

function timedRequest(metric, fn) {
	const response = fn();
	metric.add(response.timings.duration);
	return response;
}

export default function () {
	let token = "";

	group("login", () => {
		const response = timedRequest(loginLatency, () =>
			http.post(endpoint("/v1/auth/login"), JSON.stringify({ account: adminUser, password: adminPassword }), authHeaders())
		);
		check(response, {
			"login returns 200": (r) => r.status === 200,
			"login returns token": (r) => extractToken(r) !== ""
		});
		token = extractToken(response);
	});

	group("user list", () => {
		const response = timedRequest(userListLatency, () =>
			http.get(endpoint("/v1/users?offset=0&limit=20"), authHeaders(token))
		);
		check(response, { "user list returns 200": (r) => r.status === 200 });
	});

	group("role list", () => {
		const response = timedRequest(roleListLatency, () =>
			http.get(endpoint("/v1/roles?offset=0&limit=20"), authHeaders(token))
		);
		check(response, { "role list returns 200": (r) => r.status === 200 });
	});

	group("plugin list", () => {
		const response = timedRequest(pluginListLatency, () =>
			http.get(endpoint("/v1/plugins?offset=0&limit=20"), authHeaders(token))
		);
		check(response, { "plugin list returns 200": (r) => r.status === 200 });
	});

	group("audit query", () => {
		const response = timedRequest(auditQueryLatency, () =>
			http.get(endpoint("/v1/audit?offset=0&limit=20"), authHeaders(token))
		);
		check(response, { "audit query returns 200": (r) => r.status === 200 });
	});

	sleep(Number(__ENV.SKOLL_PERF_SLEEP_SECONDS || 1));
}
