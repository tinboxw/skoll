import { createApp } from "vue";
import "element-plus/dist/index.css";

import "../../src/styles/variables.scss";
import "../../src/styles/global.scss";
import DocumentKitHarness from "./DocumentKitHarness.vue";

const params = new URLSearchParams(location.search);
document.documentElement.dataset.theme = params.get("theme") === "dark" ? "dark" : "light";
document.documentElement.dataset.density = params.get("density") === "compact" ? "compact" : "comfortable";
createApp(DocumentKitHarness).mount("#app");
