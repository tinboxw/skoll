const baseUrl = (process.env.SKOLL_API_BASE || "http://127.0.0.1:8080").replace(/\/$/, "");

async function main() {
  const resp = await fetch(`${baseUrl}/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ account: "admin", password: "admin" })
  });

  if (!resp.ok) {
    throw new Error(`auth login failed with status ${resp.status}`);
  }

  const payload = await resp.json();
  const token = payload?.data?.token;
  const tokenType = payload?.data?.tokenType;
  const expiresIn = payload?.data?.expiresIn;
  const permissions = payload?.data?.permissions;

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

  const protectedNoAuth = await fetch(`${baseUrl}/v1/users`);
  if (protectedNoAuth.status !== 401) {
    throw new Error(`expected /v1/users without auth to return 401, got ${protectedNoAuth.status}`);
  }

  const protectedWithAuth = await fetch(`${baseUrl}/v1/users`, {
    headers: {
      Authorization: `Bearer ${token}`
    }
  });
  if (protectedWithAuth.status === 401) {
    throw new Error("expected /v1/users with bearer token to pass auth middleware");
  }

  console.log("auth smoke passed", {
    endpoint: `${baseUrl}/v1/auth/login`,
    tokenType,
    expiresIn,
    permissionsCount: permissions.length,
    protectedNoAuthStatus: protectedNoAuth.status,
    protectedWithAuthStatus: protectedWithAuth.status
  });
}

main().catch((err) => {
  console.error("auth smoke failed:", err instanceof Error ? err.message : String(err));
  process.exit(1);
});
