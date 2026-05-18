const commandMatrix = {
  "system.check": "System health check passed. core services and plugin routes are reachable.",
  "system.sync": "Metadata sync completed. registry, manifests, and local caches are aligned.",
  "system.audit": "Audit digest generated. no critical risk detected; two medium-priority items queued."
};

const governanceBoard = [
  {
    title: "Identity hardening",
    detail: "Migrate remaining long-lived tokens to rotating service grants across edge gateways."
  },
  {
    title: "Policy baseline",
    detail: "Consolidate plugin-level mount policy into one signed profile for all admin spaces."
  },
  {
    title: "Operational drill",
    detail: "Run failover rehearsal every Wednesday with command output archived to audit digest."
  }
];

function renderBoard() {
  const list = document.querySelector("#board-list");
  if (!list) {
    return;
  }
  list.innerHTML = "";
  governanceBoard.forEach((item) => {
    const li = document.createElement("li");
    li.innerHTML = `<strong>${item.title}</strong><p>${item.detail}</p>`;
    list.appendChild(li);
  });
}

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
  const buttons = document.querySelectorAll("[data-command]");
  buttons.forEach((button) => {
    button.addEventListener("click", () => {
      runCommand(button.getAttribute("data-command") || "");
    });
  });
}

bindEvents();
renderBoard();
