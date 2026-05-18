const cards = document.querySelector("#metrics-cards");
const output = document.querySelector("#output");

async function requestJSON(url) {
  const resp = await fetch(url);
  if (!resp.ok) {
    throw new Error(`request failed: ${resp.status}`);
  }
  return await resp.json();
}

function renderMetrics(data) {
  if (!cards) {
    return;
  }
  cards.innerHTML = "";

  const rows = [
    {
      title: "Availability",
      value: `${data.availabilityScore || 0}%`,
      detail: "Health score synthesized from total requests and error count"
    },
    {
      title: "Total Requests",
      value: String(data.totalRequests || 0),
      detail: "Combined route hit volume in the current snapshot"
    },
    {
      title: "Queue Signals",
      value: Array.isArray(data.queues) ? String(data.queues.length) : "0",
      detail: "Queue channels currently tracked by the backend plugin"
    }
  ];

  rows.forEach((item) => {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `<span>${item.title}</span><strong>${item.value}</strong><p>${item.detail}</p>`;
    cards.appendChild(card);
  });
}

async function loadMetrics() {
  if (output) {
    output.textContent = "Loading /demo-backend/metrics ...";
  }
  try {
    const data = await requestJSON("/demo-backend/metrics?api=168&jobs=44&cache=36&errors=3");
    renderMetrics(data);
    if (output) {
      output.textContent = JSON.stringify(data, null, 2);
    }
  } catch (error) {
    if (output) {
      output.textContent = String(error);
    }
  }
}

async function loadAudit() {
  if (output) {
    output.textContent = "Loading /demo-backend/audit/report ...";
  }
  try {
    const data = await requestJSON("/demo-backend/audit/report?events=user.create,user.update,user.create,role.bind,role.bind&limit=3");
    if (output) {
      output.textContent = JSON.stringify(data, null, 2);
    }
  } catch (error) {
    if (output) {
      output.textContent = String(error);
    }
  }
}

document.querySelector("#load-metrics")?.addEventListener("click", () => {
  void loadMetrics();
});

document.querySelector("#load-audit")?.addEventListener("click", () => {
  void loadAudit();
});

void loadMetrics();
