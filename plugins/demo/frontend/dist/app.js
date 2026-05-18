const output = document.querySelector("#output");
const statsContainer = document.querySelector("#stats");
const focusList = document.querySelector("#focus-list");

const fallbackStats = [
  { label: "Pipeline Value", value: "$12.4M", change: "+18% quarter-on-quarter" },
  { label: "Live Regions", value: "06", change: "APAC launch unlocked" },
  { label: "Studio Utilization", value: "92%", change: "Two new lab sprints added" }
];

const fallbackFocus = [
  {
    title: "Retail Atelier",
    summary: "Convert flagship stores into low-friction pickup lounges with guided discovery walls.",
    region: "Shanghai",
    stage: "prototype"
  },
  {
    title: "Field Service Kit",
    summary: "Bundle diagnostics, repair scripts, and concierge messaging into one tablet workflow.",
    region: "Shenzhen",
    stage: "pilot"
  },
  {
    title: "Executive Briefing Deck",
    summary: "Turn weekly platform signals into a board-ready narrative with market and risk framing.",
    region: "Global",
    stage: "active"
  }
];

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
    renderOverviewSections(data);
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

function renderStats(items) {
  if (!statsContainer) {
    return;
  }
  const rows = Array.isArray(items) && items.length > 0 ? items : fallbackStats;
  statsContainer.innerHTML = "";
  rows.forEach((item) => {
    const card = document.createElement("article");
    card.className = "stat";
    card.innerHTML = `<span>${item.label || "Metric"}</span><strong>${item.value || "-"}</strong><p>${item.change || ""}</p>`;
    statsContainer.appendChild(card);
  });
}

function renderFocus(items) {
  if (!focusList) {
    return;
  }
  const rows = Array.isArray(items) && items.length > 0 ? items : fallbackFocus;
  focusList.innerHTML = "";
  rows.forEach((item) => {
    const li = document.createElement("li");
    li.innerHTML = `<strong>${item.title || "Focus"}</strong><p>${item.summary || ""}</p><small>${item.region || "-"} · ${item.stage || "-"}</small>`;
    focusList.appendChild(li);
  });
}

function renderOverviewSections(payload) {
  const obj = payload && typeof payload === "object" ? payload : {};
  renderStats(obj.stats);
  renderFocus(obj.focusAreas);
}

renderStats(fallbackStats);
renderFocus(fallbackFocus);
