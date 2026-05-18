const baseUrl = (process.env.SKOLL_API_BASE || process.env.SKOLL_API_PROXY_TARGET || "http://127.0.0.1:8080").replace(/\/$/, "");
const apiBasePrefix = (process.env.SKOLL_API_BASE_PREFIX || "/skoll").replace(/\/+$/, "") || "/skoll";

async function main() {
  const resp = await fetch(`${baseUrl}${apiBasePrefix}/health`);
  if (!resp.ok) {
    throw new Error(`health check failed with status ${resp.status}`);
  }

  let payload = null;
  try {
    payload = await resp.json();
  } catch {
    payload = null;
  }

  console.log("health smoke passed", {
    endpoint: `${baseUrl}${apiBasePrefix}/health`,
    status: resp.status,
    payload
  });
}

main().catch((err) => {
  console.error("health smoke failed:", err instanceof Error ? err.message : String(err));
  process.exit(1);
});

