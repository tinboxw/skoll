# @skoll/plugin-test

`@skoll/plugin-test` is the current frontend test contract for independent Skoll plugins. It provides deterministic UI-state matrices, visual case naming, viewport overflow checks, long-task collection, action timing, and executable performance budgets without importing host source.

The package has no runtime dependency on Playwright. Its structural `PluginBrowserPage` contract accepts a Playwright `Page`, while callbacks keep assertions in the plugin's own test suite.

```ts
import {
  PLUGIN_UI_STATES,
  assertPluginPerformanceBudget,
  installPluginPerformanceObserver,
  runPluginStateMatrix
} from "@skoll/plugin-test";

await installPluginPerformanceObserver(page);
await runPluginStateMatrix(
  PLUGIN_UI_STATES,
  (state) => renderState(state),
  (state) => assertState(page, state)
);

assertPluginPerformanceBudget(measurement, budget);
```

There is one current API. Do not copy the driver into a plugin or add legacy state aliases.
