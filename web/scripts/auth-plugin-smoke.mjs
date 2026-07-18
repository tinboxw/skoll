const baseUrl = (process.env.SKOLL_API_BASE || "http://127.0.0.1:8080").replace(/\/$/, "");
const apiBasePrefix = (process.env.SKOLL_API_BASE_PREFIX || "/skoll").replace(/\/+$/, "") || "/skoll";
const account = process.env.SKOLL_AUTH_ACCOUNT || "admin";
const password = process.env.SKOLL_AUTH_PASSWORD || "Admin@123456";

async function main() {
  const resp = await fetch(`${baseUrl}${apiBasePrefix}/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      account,
      password,
      organizationId: "forged-organization",
      organizationPath: ["forged-organization"],
      role: "forged-role",
      roles: ["forged-role"]
    })
  });

  if (!resp.ok) {
    throw new Error(`auth login failed with status ${resp.status}`);
  }

  const payload = await resp.json();
  const token = payload?.data?.token;
  const tokenType = payload?.data?.tokenType;
  const expiresIn = payload?.data?.expiresIn;
  const permissions = payload?.data?.permissions;
  const user = payload?.data?.user;

  if (typeof token !== "string" || token.trim() === "") {
    throw new Error("missing token in response payload");
  }
  if (typeof tokenType !== "string" || tokenType.trim() === "") {
    throw new Error("missing tokenType in response payload");
  }
  if (typeof expiresIn !== "number" || expiresIn <= 0) {
    throw new Error("missing or invalid expiresIn in response payload");
  }
  if (!Array.isArray(permissions)) {
    throw new Error("missing permissions array in response payload");
  }
  if (!user || !Array.isArray(user.roles) || !Array.isArray(user.organizationPath)) {
    throw new Error("missing trusted role or organization profile claims");
  }
  if (user.organizationId === "forged-organization" || user.roles.includes("forged-role")) {
    throw new Error("login request fields influenced trusted identity claims");
  }

  const tokenPayload = JSON.parse(Buffer.from(token.split(".")[1], "base64url").toString("utf8"));
  if (
    tokenPayload.sub !== user.id ||
    tokenPayload.organizationId !== (user.organizationId || undefined) ||
    JSON.stringify(tokenPayload.organizationPath || []) !== JSON.stringify(user.organizationPath) ||
    JSON.stringify(tokenPayload.roles) !== JSON.stringify(user.roles)
  ) {
    throw new Error("JWT identity claims do not match the login profile");
  }

  const protectedNoAuth = await fetch(`${baseUrl}${apiBasePrefix}/v1/users`);
  if (protectedNoAuth.status !== 401) {
	throw new Error(`expected ${apiBasePrefix}/v1/users without auth to return 401, got ${protectedNoAuth.status}`);
  }

  const protectedWithAuth = await fetch(`${baseUrl}${apiBasePrefix}/v1/users`, {
    headers: {
      Authorization: `Bearer ${token}`
    }
  });
  if (protectedWithAuth.status === 401) {
    throw new Error(`expected ${apiBasePrefix}/v1/users with bearer token to pass auth middleware`);
  }

  console.log("auth smoke passed", {
    endpoint: `${baseUrl}${apiBasePrefix}/v1/auth/login`,
    tokenType,
    expiresIn,
    permissionsCount: permissions.length,
    organizationId: user.organizationId,
    roles: user.roles,
    protectedNoAuthStatus: protectedNoAuth.status,
    protectedWithAuthStatus: protectedWithAuth.status
  });
}

main().catch((err) => {
  console.error("auth smoke failed:", err instanceof Error ? err.message : String(err));
  process.exit(1);
});

