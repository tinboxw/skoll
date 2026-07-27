import assert from "node:assert/strict";

import {
	PLUGIN_UI_STATES,
	assertPluginPerformanceBudget,
	createPluginVisualMatrix,
	measurePluginAction,
	runPluginStateMatrix
} from "../dist/index.js";

const visited = [];
await runPluginStateMatrix(
	PLUGIN_UI_STATES,
	async (state) => ({ state }),
	async (state, context) => {
		assert.equal(context.state, state);
		visited.push(state);
	}
);
assert.deepEqual(visited, PLUGIN_UI_STATES);

const visual = createPluginVisualMatrix("medical_oa", {
	states: ["success", "error"],
	viewports: ["desktop", "mobile"],
	themes: ["light"],
	locales: ["zh-CN"]
});
assert.equal(visual.length, 4);
assert.equal(visual[0].snapshot, "medical-oa-success-desktop-light-zh-cn.png");

let now = 10;
const duration = await measurePluginAction(
	async () => {
		now = 15;
	},
	async () => {
		now = 24;
	},
	() => now
);
assert.equal(duration, 14);

assert.doesNotThrow(() => assertPluginPerformanceBudget(
	{ routeReadyMs: 100, interactionMs: 30, longTasks: [10, 20], heapGrowthBytes: 1024 },
	{ routeReadyMs: 200, interactionMs: 50, maxLongTaskMs: 30, totalLongTaskMs: 40, heapGrowthBytes: 2048 }
));
assert.throws(() => assertPluginPerformanceBudget(
	{ routeReadyMs: 201, interactionMs: 30, longTasks: [], heapGrowthBytes: 0 },
	{ routeReadyMs: 200, interactionMs: 50, maxLongTaskMs: 30, totalLongTaskMs: 40, heapGrowthBytes: 2048 }
), /route readiness/);

console.log(JSON.stringify({ states: visited.length, visualCases: visual.length, duration }));
