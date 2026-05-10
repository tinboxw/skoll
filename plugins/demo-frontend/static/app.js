/*
 * Demo Frontend-Only Plugin
 *
 * Module A: release feed filtering and rendering
 * Module B: command simulation panel for plugin operations
 *
 * Usage example:
 *   open index.html in browser and try filtering by keyword or running commands.
 */

const releaseData = [
	{
		version: "0.1.0",
		title: "Initial frontend-only plugin scaffold",
		tags: ["init", "ui"],
		date: "2026-05-01"
	},
	{
		version: "0.2.0",
		title: "Add operation command simulator",
		tags: ["ops", "demo"],
		date: "2026-05-06"
	},
	{
		version: "0.3.0",
		title: "Improve release filter and detail drawer",
		tags: ["ux", "filter"],
		date: "2026-05-09"
	}
];

const commandMatrix = {
	"plugin.validate": "Manifest validation passed. Dependencies: 0, Permissions: 1",
	"plugin.install": "Plugin installed successfully (frontend-only mode)",
	"plugin.enable": "Plugin state switched to ENABLED",
	"plugin.disable": "Plugin state switched to DISABLED"
};

/**
 * Renders release items by keyword.
 * @param {string} keyword - Search text against title/version/tags.
 */
function renderReleaseFeed(keyword) {
	const list = document.querySelector("#release-list");
	if (!list) {
		return;
	}

	const q = String(keyword || "").trim().toLowerCase();
	const filtered = releaseData.filter((item) => {
		if (!q) {
			return true;
		}
		return (
			item.title.toLowerCase().includes(q) ||
			item.version.toLowerCase().includes(q) ||
			item.tags.join(",").toLowerCase().includes(q)
		);
	});

	list.innerHTML = "";
	if (filtered.length === 0) {
		const empty = document.createElement("li");
		empty.className = "empty";
		empty.textContent = "No release notes matched your keyword.";
		list.appendChild(empty);
		return;
	}

	filtered.forEach((item) => {
		const row = document.createElement("li");
		row.className = "release-item";
		row.innerHTML = `
			<div class="release-head">
				<strong>${item.version}</strong>
				<span>${item.date}</span>
			</div>
			<p>${item.title}</p>
			<small>${item.tags.map((tag) => `#${tag}`).join(" ")}</small>
		`;
		list.appendChild(row);
	});
}

/**
 * Runs a mock command and prints execution log.
 * @param {string} commandName - Command key defined in commandMatrix.
 */
function runCommand(commandName) {
	const output = document.querySelector("#command-output");
	if (!output) {
		return;
	}

	const result = commandMatrix[commandName] || "Unknown command";
	const now = new Date().toISOString();
	output.textContent = `[${now}] ${commandName}\n${result}`;
}

function bindEvents() {
	const input = document.querySelector("#release-filter");
	if (input) {
		input.addEventListener("input", (event) => {
			const value = event && event.target ? event.target.value : "";
			renderReleaseFeed(value);
		});
	}

	const buttons = document.querySelectorAll("[data-command]");
	buttons.forEach((button) => {
		button.addEventListener("click", () => {
			runCommand(button.getAttribute("data-command") || "");
		});
	});
}

bindEvents();
renderReleaseFeed("");
