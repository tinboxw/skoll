const baseUrl = (process.env.SKOLL_API_BASE || process.env.SKOLL_API_PROXY_TARGET || "http://127.0.0.1:8080").replace(/\/$/, "");
const apiBasePrefix = (process.env.SKOLL_API_BASE_PREFIX || "/skoll").replace(/\/+$/, "") || "/skoll";

async function getJSON(path) {
  const resp = await fetch(`${baseUrl}${path}`);
  let payload = null;
  try {
    payload = await resp.json();
  } catch {
    payload = null;
  }
  return { resp, payload };
}

async function main() {
  const listAll = await getJSON(`${apiBasePrefix}/v1/plugins`);
  if (!listAll.resp.ok) {
	throw new Error(`GET ${apiBasePrefix}/v1/plugins failed with status ${listAll.resp.status}`);
  }

  const allItems = Array.isArray(listAll.payload?.data) ? listAll.payload.data : [];
  if (allItems.length === 0) {
    throw new Error("plugin list is empty");
  }

  const listEnabled = await getJSON(`${apiBasePrefix}/v1/plugins?enabled=true`);
  if (!listEnabled.resp.ok) {
	throw new Error(`GET ${apiBasePrefix}/v1/plugins?enabled=true failed with status ${listEnabled.resp.status}`);
  }

  const selected = allItems.find((item) => typeof item?.id === "string" && item.id.trim() !== "") || allItems[0];
  const pluginId = String(selected.id);

  const debugResp = await getJSON(`${apiBasePrefix}/v1/plugins/${pluginId}/debug`);
  if (!debugResp.resp.ok) {
	throw new Error(`GET ${apiBasePrefix}/v1/plugins/${pluginId}/debug failed with status ${debugResp.resp.status}`);
  }
  if (String(debugResp.payload?.data?.id || "") !== pluginId) {
    throw new Error(`debug payload id mismatch, expected ${pluginId}`);
  }

  const logsResp = await fetch(`${baseUrl}${apiBasePrefix}/v1/plugins/${pluginId}/logs`);
  if (logsResp.status !== 200 && logsResp.status !== 404) {
    throw new Error(`GET /skoll/v1/plugins/${pluginId}/logs expected 200 or 404, got ${logsResp.status}`);
  }

  const notFoundDebug = await fetch(`${baseUrl}${apiBasePrefix}/v1/plugins/skoll-smoke-not-found/debug`);
  if (notFoundDebug.status !== 404) {
    throw new Error(`expected 404 for unknown plugin debug, got ${notFoundDebug.status}`);
  }

  const notFoundLogs = await fetch(`${baseUrl}${apiBasePrefix}/v1/plugins/skoll-smoke-not-found/logs`);
  if (notFoundLogs.status !== 404) {
    throw new Error(`expected 404 for unknown plugin logs, got ${notFoundLogs.status}`);
  }

  console.log("plugin sync smoke passed", {
    endpoint: `${baseUrl}${apiBasePrefix}/v1/plugins`,
    totalPlugins: allItems.length,
    enabledPlugins: Array.isArray(listEnabled.payload?.data) ? listEnabled.payload.data.length : 0,
    inspectedPlugin: pluginId,
    logsStatus: logsResp.status,
    fallbackDebugStatus: notFoundDebug.status,
    fallbackLogsStatus: notFoundLogs.status
  });
}

main().catch((err) => {
  console.error("plugin sync smoke failed:", err instanceof Error ? err.message : String(err));
  process.exit(1);
});

