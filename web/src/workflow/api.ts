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
	escalation?: WorkflowEscalationRule;
	signature?: WorkflowSignaturePolicy;
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

export type WorkflowEscalationRule = {
	afterSeconds: number;
	target: WorkflowActor;
};

export type WorkflowSignaturePolicy = {
	meaning: string;
	requireEvidence: boolean;
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
	originalAssignee: WorkflowActor;
	assignment: "direct" | "delegated" | "substituted" | "escalated";
	authorizedBy?: WorkflowActor;
	authorizationId?: string;
	status: "pending" | "approved" | "rejected" | "delegated" | "copied" | "canceled";
	createdAt: string;
	completedAt?: string;
};

export type WorkflowAction = {
	id: string;
	type: "start" | "approve" | "reject" | "withdraw" | "delegate" | "substitute" | "escalate" | "copy" | "cancel";
	instanceId: string;
	taskId?: string;
	nodeId?: string;
	actor: WorkflowActor;
	target?: WorkflowActor;
	comment?: string;
	receiptId?: string;
	createdAt: string;
};

export type WorkflowEvidenceReference = {
	fileId: string;
	name: string;
	hash: string;
	size: number;
	mime: string;
};

export type WorkflowSignatureReceipt = {
	id: string;
	actionId: string;
	instanceId: string;
	definitionId: string;
	definitionKey: string;
	businessType: string;
	businessId: string;
	taskId: string;
	nodeId: string;
	action: "approve" | "reject";
	actor: WorkflowActor;
	meaning: string;
	verificationId: string;
	verificationMethod: "password";
	verificationAt: string;
	audience: string;
	evidence: WorkflowEvidenceReference[];
	commentDigest: string;
	evidenceDigest: string;
	auditCorrelationId: string;
	signedAt: string;
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
	receipts: WorkflowSignatureReceipt[];
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
	signature?: WorkflowDecisionSignatureRequest;
};

export type WorkflowDecisionSignatureRequest = {
	proof: string;
	meaning: string;
	evidenceIds: string[];
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

export async function getWorkflowDefinition(id: string): Promise<WorkflowDefinition> {
	const payload = await apiGet<ApiResponse<ItemPayload<WorkflowDefinition>>>(`/v1/workflows/definitions/${encodeURIComponent(id)}`);
	return payload.data.item;
}

export async function requestWorkflowReverification(password: string): Promise<{
	proof: string;
	verificationId: string;
	verifiedAt: string;
	expiresAt: string;
}> {
	const payload = await apiPost<ApiResponse<{
		proof: string;
		verificationId: string;
		verifiedAt: string;
		expiresAt: string;
	}>>("/v1/auth/reverify", { password, audience: "core" });
	return payload.data;
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

export async function delegateWorkflowTask(instanceId: string, taskId: string, body: WorkflowTaskTargetActionRequest): Promise<WorkflowInstance> {
	return targetTaskAction("delegate", instanceId, taskId, body);
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

async function targetTaskAction(action: "delegate" | "copy", instanceId: string, taskId: string, body: WorkflowTaskTargetActionRequest): Promise<WorkflowInstance> {
	const payload = await apiPost<ApiResponse<ItemPayload<WorkflowInstance>>>(
		`/v1/workflows/instances/${encodeURIComponent(instanceId)}/tasks/${encodeURIComponent(taskId)}/${action}`,
		body
	);
	return payload.data.item;
}
