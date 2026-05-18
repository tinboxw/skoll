const widgetGrid = document.querySelector("#widget-grid");
const output = document.querySelector("#output");
const list = document.querySelector("#program-list");

const programs = [
	{
		name: "North Harbor Flagship",
		summary: "Rebuild showroom flow around guided diagnostics and concierge transitions."
	},
	{
		name: "Pulse Control Room",
		summary: "Merge campaign telemetry with plugin runtime health for executive reviews."
	},
	{
		name: "Service Atelier",
		summary: "Design a premium post-sale workflow with live route insight and response scripts."
	}
];

function appendPrograms() {
	if (!list) {
		return;
	}
	list.innerHTML = "";
	programs.forEach((item) => {
		const li = document.createElement("li");
		li.innerHTML = `<strong>${item.name}</strong><p>${item.summary}</p>`;
		list.appendChild(li);
	});
}

async function requestJSON(url) {
	const resp = await fetch(url);
	if (!resp.ok) {
		throw new Error(`request failed: ${resp.status}`);
	}
	return await resp.json();
}

function renderWidgets(data) {
	if (!widgetGrid) {
		return;
	}
	const rows = Array.isArray(data) ? data : [];
	widgetGrid.innerHTML = "";
	rows.forEach((item) => {
		const card = document.createElement("article");
		card.className = "widget";
		card.innerHTML = `
			<h3>${item.label || item.name || "Metric"}</h3>
			<strong>${item.value || "-"}</strong>
			<p>${item.hint || ""}</p>
			<span class="pill">${item.delta || item.status || "ok"}</span>
		`;
		widgetGrid.appendChild(card);
	});
}

async function loadWidgets() {
	if (output) {
		output.textContent = "Loading /demo-monolith/widgets ...";
	}
	try {
		const data = await requestJSON("/demo-monolith/widgets?visitors=96&error_rate=1.4");
		renderWidgets(data);
		if (output) {
			output.textContent = JSON.stringify(data, null, 2);
		}
	} catch (error) {
		if (output) {
			output.textContent = String(error);
		}
	}
}

async function loadManifest() {
	if (output) {
		output.textContent = "Loading /demo-monolith/manifest ...";
	}
	try {
		const data = await requestJSON("/demo-monolith/manifest");
		if (output) {
			output.textContent = JSON.stringify(data, null, 2);
		}
	} catch (error) {
		if (output) {
			output.textContent = String(error);
		}
	}
}

document.querySelector("#load-widgets")?.addEventListener("click", () => {
	void loadWidgets();
});

document.querySelector("#load-manifest")?.addEventListener("click", () => {
	void loadManifest();
});

appendPrograms();
void loadWidgets();
