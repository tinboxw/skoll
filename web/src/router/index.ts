import { createRouter, createWebHistory, type RouteRecordRaw } from "vue-router";
import { defineComponent, h } from "vue";

const HomePage = defineComponent({
	name: "HomePage",
	setup() {
		return () => h("div", "Frontend plugin host is ready.");
	}
});

const routes: RouteRecordRaw[] = [
	{
		path: "/",
		name: "home",
		component: HomePage
	}
];

export const router = createRouter({
	history: createWebHistory(),
	routes
});

