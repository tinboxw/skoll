import { mount } from "@vue/test-utils";
import { defineComponent } from "vue";
import { createMemoryHistory, createRouter } from "vue-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import Header from "./Header.vue";
import PinnedTabs from "./PinnedTabs.vue";
import Sidebar from "./Sidebar.vue";

const ButtonStub = defineComponent({
	name: "ElButton",
	emits: ["click"],
	template: "<button @click=\"$emit('click')\"><slot /></button>"
});
const SegmentedStub = defineComponent({
	name: "ElSegmented",
	props: ["modelValue"],
	emits: ["update:modelValue"],
	template: "<div class=\"segmented-stub\" />"
});
const InlineStub = defineComponent({
	template: "<span><slot /></span>"
});
const ContainerStub = defineComponent({
	template: "<div><slot /><slot name=\"dropdown\" /></div>"
});
const elementStubs = {
	ElAvatar: InlineStub,
	ElButton: ButtonStub,
	ElDropdown: ContainerStub,
	ElDropdownItem: InlineStub,
	ElDropdownMenu: ContainerStub,
	ElSegmented: SegmentedStub,
	ElTag: InlineStub,
	ElTooltip: ContainerStub
};

function createTestRouter() {
	return createRouter({
		history: createMemoryHistory(),
		routes: [
			{ path: "/", component: { template: "<div />" } },
			{ path: "/skoll/", component: { template: "<div />" } },
			{ path: "/users", component: { template: "<div />" } },
			{ path: "/roles", component: { template: "<div />" } }
		]
	});
}

describe("layout controls", () => {
	beforeEach(() => {
		localStorage.clear();
	});

	it("routes header choices and commands through explicit contracts", async () => {
		const setLocale = vi.fn();
		const setColorScheme = vi.fn();
		const setDensity = vi.fn();
		const openProfile = vi.fn();
		const logout = vi.fn();
		const wrapper = mount(Header, {
			global: { stubs: elementStubs },
			props: {
				title: "Skoll",
				subtitle: "Admin",
				userName: "Ada",
				userAvatarUrl: "",
				pluginCount: 4,
				synced: true,
				error: null,
				locale: "zh-CN",
				setLocale,
				colorScheme: "light",
				setColorScheme,
				density: "comfortable",
				setDensity,
				onOpenProfile: openProfile,
				onLogout: logout
			}
		});

		const segmentedControls = wrapper.findAllComponents({ name: "ElSegmented" });
		expect(segmentedControls).toHaveLength(3);
		segmentedControls[0].vm.$emit("update:modelValue", "en-US");
		segmentedControls[1].vm.$emit("update:modelValue", "dark");
		segmentedControls[2].vm.$emit("update:modelValue", "compact");
		await wrapper.find("button[aria-label='切换菜单']").trigger("click");

		expect(setLocale).toHaveBeenCalledWith("en-US");
		expect(setColorScheme).toHaveBeenCalledWith("dark");
		expect(setDensity).toHaveBeenCalledWith("compact");
		expect(wrapper.emitted("toggle-sidebar")).toHaveLength(1);
		expect(wrapper.text()).toContain("插件 4");
	});

	it("opens, pins, and closes workspace tabs without ambiguous events", async () => {
		const router = createTestRouter();
		await router.push("/users");
		await router.isReady();
		const wrapper = mount(PinnedTabs, {
			global: { plugins: [router], stubs: elementStubs },
			props: {
				pinnedItems: [{ id: "users", path: "/users", label: "Users" }],
				recentItems: [{ id: "roles", path: "/roles", label: "Roles" }]
			}
		});

		await wrapper.findAll(".tab-main")[0].trigger("click");
		await wrapper.get(".tab-pin").trigger("click");
		await wrapper.findAll(".tab-close")[0].trigger("click");
		await wrapper.findAll(".tab-close")[1].trigger("click");

		expect(wrapper.emitted("open")?.[0]).toEqual(["/users"]);
		expect(wrapper.emitted("pin")?.[0]).toEqual(["/roles"]);
		expect(wrapper.emitted("unpin")?.[0]).toEqual(["/users"]);
		expect(wrapper.emitted("removeRecent")?.[0]).toEqual(["/roles"]);
		expect(wrapper.get(".tab-group.active").text()).toContain("Users");
	});

	it("keeps the mobile sidebar visible and closes it after navigation", async () => {
		const router = createTestRouter();
		await router.push("/");
		await router.isReady();
		const wrapper = mount(Sidebar, {
			global: { plugins: [router] },
			props: {
				collapsed: false,
				mobile: true,
				items: [{ to: "/users", label: "Users", icon: "users", order: 10, source: "system" }]
			}
		});

		expect(wrapper.get("aside").classes()).toContain("sidebar--mobile");
		await wrapper.get(".link").trigger("click");
		expect(wrapper.emitted("navigate")).toHaveLength(1);
	});
});
