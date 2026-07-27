import { createApp } from "vue";
import ElementPlus from "element-plus";
import zhCN from "element-plus/es/locale/lang/zh-cn";
import enUS from "element-plus/es/locale/lang/en";
import "element-plus/dist/index.css";
import "../../src/styles/variables.scss";
import "../../../packages/skoll-business-ui/src/theme.css";

import BusinessUIHarness from "./BusinessUIHarness.vue";

const query = new URLSearchParams(window.location.search);
const locale = query.get("locale") === "en-US" ? "en-US" : "zh-CN";
const colorScheme = query.get("theme") === "dark" ? "dark" : "light";
const density = query.get("density") === "compact" ? "compact" : "comfortable";

document.documentElement.lang = locale;
document.documentElement.dataset.theme = colorScheme;
document.documentElement.dataset.density = density;
document.documentElement.style.colorScheme = colorScheme;

createApp(BusinessUIHarness, { locale })
	.use(ElementPlus, { locale: locale === "en-US" ? enUS : zhCN })
	.mount("#app");
