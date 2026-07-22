import { beforeEach, describe, expect, it } from "vitest";

import { loadNotifications, seedNotificationDemo } from "./types";

beforeEach(() => {
	localStorage.clear();
});

describe("notification demo copy", () => {
	it("stores the caller-provided locale copy", () => {
		seedNotificationDemo("actor-1", {
			qualificationTitle: "资质即将到期",
			qualificationBody: "请复核证照。",
			workflowTitle: "工作流抄送",
			workflowBody: "请查看审批。"
		});

		const items = loadNotifications();
		expect(items.map((item) => item.title)).toEqual(expect.arrayContaining(["资质即将到期", "工作流抄送"]));
		expect(items.map((item) => item.body)).toEqual(expect.arrayContaining(["请复核证照。", "请查看审批。"]));
	});
});
