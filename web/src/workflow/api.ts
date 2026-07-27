import { apiGet, apiPost, type ApiResponse } from "../utils/api";

export type WorkflowActor = {
	id: string;
	name?: string;
};

export type WorkflowNode = {
	id: string;
	key: string;
	name: string;
	type: "start" | "approval" | "cc" | "end";
	assignees?: string[];
	decision?: WorkflowDecisionRule;
};

export type WorkflowTransition = {
	from: string;
	to: string;
	condition?: WorkflowCondition;
};

export type WorkflowDecisionRule = {
	strategy: "any" | "all" | "quorum";
	quorum: number;
};

export type WorkflowValue = {
	type: "string" | "number" | "boolean";
	value: string;
};

export type WorkflowPredicate = {
	field: string;
	operator: "eq" | "ne" | "gt" | "gte" | "lt" | "lte" | "exists" | "not_exists";
	value?: WorkflowValue;
};

export type WorkflowCondition = {
	match: "all" | "any";
	predicates: WorkflowPredicate[];
};

export type WorkflowDefinition = {
	id: string;
	key: string;
	name: string;
	version: number;
	status: "draft" | "published" | "disabled";
	nodes: WorkflowNode[];
	transitions: WorkflowTransition[];
	createdAt: string;
	updatedAt: string;
};

export type WorkflowTask = {
	id: string;
	instanceId: string;
	nodeId: string;
	assignee: WorkflowActor;
	status: "pending" | "approved" | "rejected" | "transferred" | "copied" | "canceled";
	createdAt: string;
	completedAt?: string;
};

export type WorkflowAction = {
	id: string;
	type: "start" | "approve" | "reject" | "withdraw" | "transfer" | "copy" | "cancel";
	instanceId: string;
	taskId?: string;
	nodeId?: string;
	actor: WorkflowActor;
	target?: WorkflowActor;
	comment?: string;
	createdAt: string;
};

export type WorkflowInstance = {
	id: string;
	definitionId: string;
	definitionKey: string;
	businessType: string;
	businessId: string;
	title: string;
	status: "running" | "approved" | "rejected" | "withdrawn" | "canceled";
	starter: WorkflowActor;
	currentNode: string;
	activeNodes: string[];
	variables: Record<string, WorkflowValue>;
	tasks: WorkflowTask[];
	timeline: WorkflowAction[];
	createdAt: string;
	updatedAt: string;
};

export type WorkflowDefinitionRequest = {
	id: string;
	key: string;
	name: string;
	version: number;
	nodes: WorkflowNode[];
	transitions: WorkflowTransition[];
};

export type WorkflowStartRequest = {
	id: string;
	definitionId: string;
	businessType: string;
	businessId: string;
	title: string;
	starter: WorkflowActor;
	variables: Record<string, WorkflowValue>;
};

export type WorkflowTaskActionRequest = {
	actor: WorkflowActor;
	comment?: string;
};

export type WorkflowTaskTargetActionRequest = WorkflowTaskActionRequest & {
	target: WorkflowActor;
};

type ItemPayload<T> = {
	item: T;
};

export async function createWorkflowDefinition(body: WorkflowDefinitionRequest): Promise<WorkflowDefinition> {
	const payload = await apiPost<ApiResponse<ItemPayload<WorkflowDefinition>>>("/v1/workflows/definitions", body);
	return payload.data.item;
}

export async function publishWorkflowDefinition(id: string): Promise<WorkflowDefinition> {
	const payload = await apiPost<ApiResponse<ItemPayload<WorkflowDefinition>>>(`/v1/workflows/definitions/${encodeURIComponent(id)}/publish`, {});
	return payload.data.item;
}

export async function startWorkflowInstance(body: WorkflowStartRequest): Promise<WorkflowInstance> {
	const payload = await apiPost<ApiResponse<ItemPayload<WorkflowInstance>>>("/v1/workflows/instances", body);
	return payload.data.item;
}

export async function getWorkflowInstance(id: string): Promise<WorkflowInstance> {
	const payload = await apiGet<ApiResponse<ItemPayload<WorkflowInstance>>>(`/v1/workflows/instances/${encodeURIComponent(id)}`);
	return payload.data.item;
}

export async function approveWorkflowTask(instanceId: string, taskId: string, body: WorkflowTaskActionRequest): Promise<WorkflowInstance> {
	return taskAction("approve", instanceId, taskId, body);
}

export async function rejectWorkflowTask(instanceId: string, taskId: string, body: WorkflowTaskActionRequest): Promise<WorkflowInstance> {
	return taskAction("reject", instanceId, taskId, body);
}

export async function transferWorkflowTask(instanceId: string, taskId: string, body: WorkflowTaskTargetActionRequest): Promise<WorkflowInstance> {
	return targetTaskAction("transfer", instanceId, taskId, body);
}

export async function copyWorkflowTask(instanceId: string, taskId: string, body: WorkflowTaskTargetActionRequest): Promise<WorkflowInstance> {
	return targetTaskAction("copy", instanceId, taskId, body);
}

export async function withdrawWorkflowInstance(instanceId: string, body: WorkflowTaskActionRequest): Promise<WorkflowInstance> {
	const payload = await apiPost<ApiResponse<ItemPayload<WorkflowInstance>>>(`/v1/workflows/instances/${encodeURIComponent(instanceId)}/withdraw`, body);
	return payload.data.item;
}

async function taskAction(action: "approve" | "reject", instanceId: string, taskId: string, body: WorkflowTaskActionRequest): Promise<WorkflowInstance> {
	const payload = await apiPost<ApiResponse<ItemPayload<WorkflowInstance>>>(
		`/v1/workflows/instances/${encodeURIComponent(instanceId)}/tasks/${encodeURIComponent(taskId)}/${action}`,
		body
	);
	return payload.data.item;
}

async function targetTaskAction(action: "transfer" | "copy", instanceId: string, taskId: string, body: WorkflowTaskTargetActionRequest): Promise<WorkflowInstance> {
	const payload = await apiPost<ApiResponse<ItemPayload<WorkflowInstance>>>(
		`/v1/workflows/instances/${encodeURIComponent(instanceId)}/tasks/${encodeURIComponent(taskId)}/${action}`,
		body
	);
	return payload.data.item;
}
