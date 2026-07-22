import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";

import PluginFleetTable from "./PluginFleetTable.vue";

describe("PluginFleetTable", () => {
	it("renders lifecycle truth and opens the routed control center", async () => {
		const wrapper = mount(PluginFleetTable, {
			props: {
				items: [
					{ id: "ready", name: "Ready", version: "1.0.0", enabled: true, uiMode: "separated" },
					{ id: "disabled", name: "Disabled", version: "1.0.0", enabled: false, uiMode: "backend_only" }
				]
			}
		});
		expect(wrapper.findAll(".el-table__row")).toHaveLength(2);
		await wrapper.findAll(".el-table__row")[0].trigger("click");
		expect(wrapper.emitted("open")?.[0]).toEqual(["ready"]);
		expect(wrapper.text()).toContain("Ready");
		expect(wrapper.text()).toContain("Disabled");
	});
});
