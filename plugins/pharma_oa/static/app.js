(function () {
	const root = document.querySelector(".pharma-shell");
	const status = document.querySelector("#status-pill");
	const seedButton = document.querySelector("#seed-button");
	const labels = {
		ready: "Ready",
		loading: "Loading",
		empty: "Empty",
		error: "Error",
		"no-permission": "No permission",
		saving: "Saving",
		destructive: "Confirm"
	};

	function setState(state) {
		root.dataset.state = state;
		status.textContent = labels[state] || "Ready";
		seedButton.textContent = state === "destructive" ? "Confirm reset" : state === "saving" ? "Saving..." : "Prepare seed";
	}

	document.querySelectorAll("[data-state-button]").forEach((button) => {
		button.addEventListener("click", () => setState(button.dataset.stateButton));
	});

	seedButton.addEventListener("click", () => {
		if (root.dataset.state === "destructive") {
			setState("ready");
			return;
		}
		setState("saving");
		window.setTimeout(() => setState("ready"), 500);
	});
})();
