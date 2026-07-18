const baseUrl = (process.env.SKOLL_API_BASE || "http://127.0.0.1:8080").replace(/\/$/, "");
const apiBasePrefix = (process.env.SKOLL_API_BASE_PREFIX || "/skoll").replace(/\/+$/, "") || "/skoll";
const account = process.env.SKOLL_AUTH_ACCOUNT || "admin";
const password = process.env.SKOLL_AUTH_PASSWORD || "Admin@123456";

async function login() {
  const resp = await fetch(`${baseUrl}${apiBasePrefix}/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ account, password })
  });
  if (!resp.ok) {
    throw new Error(`auth login failed with status ${resp.status}`);
  }
  const payload = await resp.json();
  const token = payload?.data?.token;
  if (typeof token !== "string" || token.trim() === "") {
    throw new Error("missing token in login response");
  }
  return token;
}

async function createUser(token) {
  const suffix = `${Date.now()}`;
  const body = {
    account: `smoke_${suffix}`,
    name: `Smoke ${suffix}`,
    email: `smoke_${suffix}@example.com`,
    passwordHash: "smoke-password-hash-12345",
    actorID: "smoke-seed"
  };

  const resp = await fetch(`${baseUrl}${apiBasePrefix}/v1/users`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`
    },
    body: JSON.stringify(body)
  });

  if (!resp.ok) {
    const raw = await resp.text();
    throw new Error(`create user failed with status ${resp.status}: ${raw}`);
  }

  const payload = await resp.json();
  const id = payload?.data?.id || payload?.data?.ID;
  if (typeof id !== "string" || id.trim() === "") {
    throw new Error("missing created user id");
  }
  return { id, suffix };
}

async function patchEmailWithoutActor(token, userId, suffix) {
  const resp = await fetch(`${baseUrl}${apiBasePrefix}/v1/users/${userId}/email`, {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`
    },
    body: JSON.stringify({
      email: `smoke_patch_${suffix}@example.com`,
      actorId: ""
    })
  });

  if (!resp.ok) {
    const raw = await resp.text();
    throw new Error(`patch email with empty actorId failed: status ${resp.status}, body ${raw}`);
  }

  return resp.status;
}

async function main() {
  const token = await login();
  const { id, suffix } = await createUser(token);
  const patchStatus = await patchEmailWithoutActor(token, id, suffix);

  console.log("auth actor fallback smoke passed", {
    endpoint: `${baseUrl}${apiBasePrefix}/v1/users/${id}/email`,
    patchStatus,
    actorMode: "empty-body-actorId"
  });
}

main().catch((err) => {
  console.error("auth actor fallback smoke failed:", err instanceof Error ? err.message : String(err));
  process.exit(1);
});

