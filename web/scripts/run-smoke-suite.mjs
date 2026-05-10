import { spawn } from "node:child_process";

const scripts = ["smoke:health", "smoke:auth", "smoke:plugins", "smoke:auth-actor"];

function runScript(name) {
  return new Promise((resolve, reject) => {
    const child = spawn(`npm run ${name}`, {
      shell: true,
      stdio: "inherit",
      env: process.env,
      cwd: process.cwd()
    });

    child.on("error", reject);
    child.on("exit", (code) => {
      if (code === 0) {
        resolve();
        return;
      }
      reject(new Error(`${name} failed with exit code ${code}`));
    });
  });
}

async function main() {
  for (const script of scripts) {
    await runScript(script);
  }
  console.log("smoke suite passed", { scripts });
}

main().catch((err) => {
  console.error("smoke suite failed:", err instanceof Error ? err.message : String(err));
  process.exit(1);
});
