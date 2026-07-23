import { describe, expect, it } from "vitest";

import { resolveSystemNavigationLabel } from "./system-label";

const zh = (key: string): string => ({
	"menu.workflow": "工作流",
	"menu.todo": "待办中心",
	"menu.formBuilder": "表单设计器"
}[key] || key);

describe("resolveSystemNavigationLabel", () => {
	it.each([
		["/skoll/workflow", "workflow", "Workflow", "工作流"],
		["/skoll/todo", "todo-center", "Todo Center", "待办中心"],
		["/skoll/form-builder", "form-builder", "Form Builder", "表单设计器"],
		["/skoll/workflow/instance-1", "unknown", "Workflow", "工作流"]
	])("localizes %s independently of the stored label", (path, id, fallback, expected) => {
		expect(resolveSystemNavigationLabel(path, id, zh, fallback)).toBe(expected);
	});

	it("keeps unrelated navigation labels unchanged", () => {
		expect(resolveSystemNavigationLabel("/skoll/plugin", "plugin", zh, "插件中心")).toBe("插件中心");
	});
});
