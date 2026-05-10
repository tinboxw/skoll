const output = document.querySelector("#output");

async function requestJSON(url) {
  const resp = await fetch(url);
  if (!resp.ok) {
    throw new Error(`request failed: ${resp.status}`);
  }
  return await resp.json();
}

async function loadOverview() {
  try {
    const data = await requestJSON("/demo-separated/overview?plugin_id=demo");
    output.textContent = JSON.stringify(data, null, 2);
  } catch (error) {
    output.textContent = String(error);
  }
}

async function loadRecommendation() {
  try {
    const data = await requestJSON("/demo-separated/recommendations?active_users=45&error_count=1&rpm=210");
    output.textContent = JSON.stringify(data, null, 2);
  } catch (error) {
    output.textContent = String(error);
  }
}

document.querySelector("#load-overview")?.addEventListener("click", () => {
  void loadOverview();
});
document.querySelector("#load-recommend")?.addEventListener("click", () => {
  void loadRecommendation();
});
