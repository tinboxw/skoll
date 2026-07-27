export type NotificationCategory = "todo" | "message" | "reminder";
export type NotificationStatus = "pending" | "done" | "read";

export type NotificationTarget = {
	type: string;
	id: string;
	path: string;
};

export type NotificationItem = {
	id: string;
	category: NotificationCategory;
	status: NotificationStatus;
	title: string;
	body: string;
	actorId: string;
	target: NotificationTarget;
	dueAt?: string;
	createdAt: string;
	updatedAt: string;
};

export type NotificationDemoCopy = {
	qualificationTitle: string;
	qualificationBody: string;
	workflowTitle: string;
	workflowBody: string;
};

export const NOTIFICATION_STORAGE_KEY = "skoll.notification.items";

export function loadNotifications(): NotificationItem[] {
	const raw = localStorage.getItem(NOTIFICATION_STORAGE_KEY);
	if (!raw) {
		return [];
	}
	const parsed = JSON.parse(raw) as unknown;
	if (!Array.isArray(parsed)) {
		throw new Error("Stored notification list is not an array.");
	}
	return parsed.map(normalizeItem).sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
}

export function saveNotifications(items: NotificationItem[]): void {
	localStorage.setItem(NOTIFICATION_STORAGE_KEY, JSON.stringify(items.map(normalizeItem)));
}

export function upsertNotification(input: Omit<NotificationItem, "createdAt" | "updatedAt" | "status"> & { status?: NotificationStatus; createdAt?: string; updatedAt?: string }): NotificationItem {
	const now = new Date().toISOString();
	const item = normalizeItem({
		...input,
		status: input.status || "pending",
		createdAt: input.createdAt || now,
		updatedAt: now
	});
	const next = loadNotifications().filter((existing) => existing.id !== item.id);
	saveNotifications([item, ...next].slice(0, 200));
	return item;
}

export function markNotificationDone(id: string, actorId: string): NotificationItem | null {
	const items = loadNotifications();
	const index = items.findIndex((item) => item.id === id && item.actorId === actorId);
	if (index < 0) {
		return null;
	}
	const item = {
		...items[index],
		status: items[index].category === "message" ? "read" as NotificationStatus : "done" as NotificationStatus,
		updatedAt: new Date().toISOString()
	};
	items.splice(index, 1, item);
	saveNotifications(items);
	return item;
}

export function completeWorkflowNotifications(instanceId: string, actorId: string): void {
	const items = loadNotifications().map((item) => {
		if (item.target.type !== "workflow" || item.target.id !== instanceId || item.actorId !== actorId || item.status !== "pending") {
			return item;
		}
		return {
			...item,
			status: item.category === "message" ? "read" as NotificationStatus : "done" as NotificationStatus,
			updatedAt: new Date().toISOString()
		};
	});
	saveNotifications(items);
}

export function createWorkflowTodo(input: {
	instanceId: string;
	title: string;
	actorId: string;
	body?: string;
}): NotificationItem {
	return upsertNotification({
		id: `todo-workflow-${input.instanceId}-${input.actorId}`,
		category: "todo",
		title: input.title,
		body: input.body || "Workflow task is waiting for your action.",
		actorId: input.actorId,
		target: {
			type: "workflow",
			id: input.instanceId,
			path: `/skoll/workflow?instance=${encodeURIComponent(input.instanceId)}`
		}
	});
}

export function createBusinessReminder(input: {
	id: string;
	title: string;
	body: string;
	actorId: string;
	target: NotificationTarget;
	dueAt?: string;
}): NotificationItem {
	return upsertNotification({
		id: input.id,
		category: "reminder",
		title: input.title,
		body: input.body,
		actorId: input.actorId,
		target: input.target,
		dueAt: input.dueAt
	});
}

export function seedNotificationDemo(actorId: string, copy: NotificationDemoCopy): NotificationItem[] {
	const now = new Date().toISOString();
	const items = [
		createBusinessReminder({
			id: `reminder-qualification-${actorId}`,
			title: copy.qualificationTitle,
			body: copy.qualificationBody,
			actorId,
			target: { type: "reminder", id: "qualification-demo-001", path: "/skoll/todo" },
			dueAt: now
		}),
		upsertNotification({
			id: `message-workflow-copied-${actorId}`,
			category: "message",
			title: copy.workflowTitle,
			body: copy.workflowBody,
			actorId,
			target: { type: "workflow", id: "wf-copy-demo", path: "/skoll/workflow?instance=wf-copy-demo" }
		})
	];
	return items;
}

function normalizeItem(input: NotificationItem): NotificationItem {
	return {
		id: String(input.id || "").trim(),
		category: normalizeCategory(input.category),
		status: normalizeStatus(input.status),
		title: String(input.title || "").trim(),
		body: String(input.body || "").trim(),
		actorId: String(input.actorId || "").trim(),
		target: {
			type: String(input.target?.type || "").trim(),
			id: String(input.target?.id || "").trim(),
			path: String(input.target?.path || "").trim()
		},
		dueAt: input.dueAt ? String(input.dueAt) : "",
		createdAt: String(input.createdAt || new Date().toISOString()),
		updatedAt: String(input.updatedAt || new Date().toISOString())
	};
}

function normalizeCategory(value: string): NotificationCategory {
	return value === "message" || value === "reminder" ? value : "todo";
}

function normalizeStatus(value: string): NotificationStatus {
	return value === "done" || value === "read" ? value : "pending";
}
