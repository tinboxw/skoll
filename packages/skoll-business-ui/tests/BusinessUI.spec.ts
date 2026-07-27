import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";

import BusinessCommandBar from "../src/BusinessCommandBar.vue";
import BusinessFilterBar from "../src/BusinessFilterBar.vue";
import BusinessList from "../src/BusinessList.vue";
import BusinessState from "../src/BusinessState.vue";
import BusinessWorkspace from "../src/BusinessWorkspace.vue";
import type { BusinessFilterField, BusinessState as BusinessStateName } from "../src/types";

describe("business workspace composition primitives", () => {
	it.each(["loading", "empty", "error", "forbidden", "conflict", "destructive", "success"] satisfies BusinessStateName[])(
		"renders the %s state with stable semantics",
		(state) => {
			const wrapper = mount(BusinessState, { props: { state: state as Exclude<BusinessStateName, "ready">, locale: "en-US" } });
			expect(wrapper.classes()).toContain(`business-state--${state}`);
			expect(wrapper.get("h3").text().length).toBeGreaterThan(0);
			expect(wrapper.attributes("role")).toBe(["error", "forbidden", "conflict", "destructive"].includes(state) ? "alert" : "status");
		}
	);

	it("keeps workspace content out of non-ready states", () => {
		const wrapper = mount(BusinessWorkspace, {
			props: { title: "Purchase workspace", state: "conflict", locale: "en-US" },
			slots: { default: "<p>stale content</p>", stateActions: "<button>Refresh</button>" }
		});
		expect(wrapper.text()).toContain("Data changed");
		expect(wrapper.text()).toContain("Refresh");
		expect(wrapper.text()).not.toContain("stale content");
	});

	it("updates, searches, and resets schema-driven filters", async () => {
		const fields: BusinessFilterField[] = [
			{ key: "keyword", label: "Keyword", type: "search", defaultValue: "" }
		];
		const wrapper = mount(BusinessFilterBar, { props: { fields, modelValue: { keyword: "" }, locale: "en-US" } });
		await wrapper.get("input").setValue("aspirin");
		await wrapper.get("form").trigger("submit");
		expect(wrapper.emitted("update:modelValue")?.[0]).toEqual([{ keyword: "aspirin" }]);
		expect(wrapper.emitted("search")).toHaveLength(1);
		await wrapper.findAll("button")[0].trigger("click");
		expect(wrapper.emitted("reset")).toHaveLength(1);
		expect(wrapper.emitted("update:modelValue")?.at(-1)).toEqual([{ keyword: "" }]);
	});

	it("renders records, pagination, and controlled list states", async () => {
		const wrapper = mount(BusinessList, {
			props: {
				items: [{ id: "1", number: "PO-001", supplier: "North", status: "open" }],
				columns: [
					{ key: "number", label: "Number" },
					{ key: "supplier", label: "Supplier" },
					{ key: "status", label: "Status" }
				],
				canNext: true,
				actionWidth: 132,
				locale: "en-US"
			}
		});
		expect(wrapper.text()).toContain("PO-001");
		expect(wrapper.findAllComponents({ name: "ElTableColumn" }).at(-1)?.props("width")).toBe(132);
		await wrapper.get(".business-list__record > button").trigger("click");
		expect(wrapper.emitted("open")?.[0]?.[0]).toMatchObject({ id: "1" });
		await wrapper.findAll("button").at(-1)?.trigger("click");
		expect(wrapper.emitted("next")).toHaveLength(1);

		await wrapper.setProps({ state: "forbidden" });
		expect(wrapper.get('[role="alert"]').text()).toContain("Access denied");
		expect(wrapper.text()).not.toContain("PO-001");
	});

	it("executes direct commands and requires confirmation for destructive commands", async () => {
		const commands = [
			{ id: "refresh", label: "Refresh" },
			{ id: "delete", label: "Delete", destructive: true, confirmation: "Delete this record?" }
		];
		const wrapper = mount(BusinessCommandBar, {
			props: { commands, locale: "en-US" },
			global: {
				stubs: {
					ElPopconfirm: {
						emits: ["confirm"],
						template: '<div><slot name="reference" /><button class="confirm" @click="$emit(\'confirm\')">Confirm</button></div>'
					}
				}
			}
		});

		await wrapper.findAll("button").find((button) => button.text() === "Refresh")?.trigger("click");
		expect(wrapper.emitted("command")?.[0]?.[0]).toMatchObject({ id: "refresh" });

		await wrapper.findAll("button").find((button) => button.text() === "Delete")?.trigger("click");
		expect(wrapper.emitted("command")).toHaveLength(1);
		await wrapper.get("button.confirm").trigger("click");
		expect(wrapper.emitted("command")?.[1]?.[0]).toMatchObject({ id: "delete" });
	});

	it("reacts to locale changes", async () => {
		const wrapper = mount(BusinessFilterBar, {
			props: {
				fields: [{ key: "keyword", label: "Keyword", type: "search" }],
				modelValue: { keyword: "" },
				locale: "en-US"
			}
		});
		expect(wrapper.text()).toContain("Search");
		await wrapper.setProps({ locale: "zh-CN" });
		expect(wrapper.text()).toContain("查询");
	});
});
