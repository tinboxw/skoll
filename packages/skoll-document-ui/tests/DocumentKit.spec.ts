import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import { nextTick } from "vue";

import DocumentApprovalPanel from "../src/DocumentApprovalPanel.vue";
import DocumentForm from "../src/DocumentForm.vue";
import DocumentPrintView from "../src/DocumentPrintView.vue";
import type { WorkflowInstance } from "../src/types";
import { documentTestSchema } from "./test-fixtures";
import { createDocumentDraft } from "../src/validation";

const workflow: WorkflowInstance = {
	id: "workflow-1",
	definitionId: "definition-1",
	definitionKey: "purchase_order",
	businessType: "purchase_order",
	businessId: "po-1",
	title: "Quarterly order",
	status: "running",
	starter: { id: "user-1", name: "User One" },
	currentNode: "manager_review",
	tasks: [{ id: "task-1", instanceId: "workflow-1", nodeId: "manager_review", assignee: { id: "manager-1", name: "Manager" }, status: "pending", createdAt: "2026-07-22T08:00:00Z" }],
	timeline: [],
	createdAt: "2026-07-22T08:00:00Z",
	updatedAt: "2026-07-22T08:00:00Z"
};

describe("business document components", () => {
	it("renders a schema form with required line controls and disables invalid submit", () => {
		const draft = createDocumentDraft(documentTestSchema, { id: "po-1", number: "PO-001", title: "Quarterly order" });
		const wrapper = mount(DocumentForm, { props: { schema: documentTestSchema, modelValue: draft, locale: "en-US" } });
		expect(wrapper.findAll(".document-form__table tbody tr")).toHaveLength(1);
		expect(wrapper.text()).toContain("Add line");
		const submit = wrapper.findAll("button").find((button) => button.text().includes("Submit"));
		expect(submit?.attributes("disabled")).toBeDefined();
	});

	it("emits an approval request from the active task", async () => {
		const wrapper = mount(DocumentApprovalPanel, { props: { workflow, availableActions: ["approve", "reject"], locale: "en-US" } });
		const approvalButtons = wrapper.findAll("button").filter((button) => button.text().includes("Approve"));
		expect(approvalButtons).toHaveLength(1);
		await approvalButtons[0].trigger("click");
		await nextTick();
		const confirmButtons = wrapper.findAll("button").filter((button) => button.text().includes("Approve"));
		const confirm = confirmButtons[confirmButtons.length - 1];
		await confirm?.trigger("click");
		expect(wrapper.emitted("action")?.[0]).toEqual([{ action: "approve", comment: "", taskId: "task-1", target: undefined }]);
	});

	it("renders redacted fields without exposing sensitive values", () => {
		const draft = createDocumentDraft(documentTestSchema, { id: "po-1", number: "PO-001", title: "Quarterly order" });
		draft.header.subject = { type: "string", value: "Order" };
		draft.header.budget = { type: "money", value: "999999", currency: "CNY" };
		const wrapper = mount(DocumentPrintView, {
			props: {
				locale: "en-US",
				payload: {
					schema: documentTestSchema,
					document: {
						...draft,
						state: "pending",
						version: 1,
						metadata: { createdAt: "2026-07-22T08:00:00Z", updatedAt: "2026-07-22T08:00:00Z", createdBy: "user-1", updatedBy: "user-1" }
					},
					workflow,
					redactedFields: ["header.budget"],
					generatedAt: "2026-07-22T09:00:00Z"
				}
			}
		});
		expect(wrapper.text()).toContain("Redacted");
		expect(wrapper.text()).not.toContain("999999");
	});
});
