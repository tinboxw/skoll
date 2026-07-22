import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";

import MetricStrip from "./MetricStrip.vue";

describe("MetricStrip", () => {
	it("renders operational metrics with warning and wide states", () => {
		const wrapper = mount(MetricStrip, {
			props: {
				items: [
					{ label: "Enabled", value: 12, note: "Disabled 1" },
					{ label: "Risk", value: 2, tone: "warning" },
					{ label: "Default home", value: "/workspace", wide: true }
				]
			}
		});

		expect(wrapper.findAll(".metric-strip__item")).toHaveLength(3);
		expect(wrapper.get(".metric-strip__item--warning").text()).toContain("Risk");
		expect(wrapper.get(".metric-strip__item--wide").text()).toContain("/workspace");
	});
});
