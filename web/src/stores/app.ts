import { defineStore } from "pinia";

type AppState = {
	sidebarCollapsed: boolean;
	busy: boolean;
};

export const useAppStore = defineStore("app", {
	state: (): AppState => ({
		sidebarCollapsed: false,
		busy: false
	}),
	actions: {
		toggleSidebar(): void {
			this.sidebarCollapsed = !this.sidebarCollapsed;
		},
		setBusy(value: boolean): void {
			this.busy = value;
		}
	}
});

