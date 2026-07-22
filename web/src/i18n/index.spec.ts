import { afterEach, describe, expect, it } from "vitest";

import { localizeKnownError, setLocale, useI18n } from "./index";

afterEach(() => {
	setLocale("zh-CN");
});

describe("platform internationalization", () => {
	it("switches Chinese and English platform copy", () => {
		const { t } = useI18n();

		setLocale("zh-CN");
		expect(t("workflow.title")).toBe("工作流");
		expect(t("todo.tab.pending", { count: 3 })).toBe("待办 3");

		setLocale("en-US");
		expect(t("workflow.title")).toBe("Workflow");
		expect(t("todo.tab.pending", { count: 3 })).toBe("Pending 3");
		expect(localStorage.getItem("skoll.ui.locale")).toBe("en-US");
		expect(document.documentElement.lang).toBe("en-US");
	});

	it("localizes known errors without changing unknown backend detail", () => {
		const { t } = useI18n();
		setLocale("en-US");
		const english = t("error.forbidden");

		setLocale("zh-CN");
		expect(localizeKnownError(english)).toBe(t("error.forbidden"));
		expect(localizeKnownError("domain-specific failure")).toBe("domain-specific failure");
	});
});
