export type DocumentLocale = "zh-CN" | "en-US";

export type WorkflowActor = {
	id: string;
	name?: string;
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
	tasks: WorkflowTask[];
	timeline: WorkflowAction[];
	createdAt: string;
	updatedAt: string;
};

export type DocumentFieldType =
	| "string"
	| "text"
	| "integer"
	| "decimal"
	| "money"
	| "quantity"
	| "boolean"
	| "date"
	| "datetime"
	| "reference"
	| "json";

export type DocumentValidationRuleType = "min" | "max" | "min_length" | "max_length" | "pattern";

export type DocumentValidationRule = {
	type: DocumentValidationRuleType;
	value: string;
	message?: string;
};

export type DocumentFieldSchema = {
	key: string;
	label: string;
	type: DocumentFieldType;
	required: boolean;
	sensitive?: boolean;
	referenceType?: string;
	rules?: DocumentValidationRule[];
};

export type DocumentLineSchema = {
	key: string;
	name: string;
	minItems: number;
	maxItems: number;
	fields: DocumentFieldSchema[];
};

export type DocumentStateSchema = {
	key: string;
	name: string;
	terminal: boolean;
};

export type DocumentActionSchema = {
	key: string;
	name: string;
	from: string[];
	to: string;
	requiresComment: boolean;
};

export type DocumentSchema = {
	key: string;
	name: string;
	version: number;
	initialState: string;
	header: DocumentFieldSchema[];
	lines: DocumentLineSchema[];
	states: DocumentStateSchema[];
	actions: DocumentActionSchema[];
};

export type DocumentReference = {
	type: string;
	id: string;
	label?: string;
};

export type DocumentValue = {
	type: DocumentFieldType;
	value?: string;
	currency?: string;
	unit?: string;
	reference?: DocumentReference;
};

export type DocumentLine = {
	id: string;
	values: Record<string, DocumentValue>;
};

export type DocumentMetadata = {
	createdAt: string;
	updatedAt: string;
	createdBy: string;
	updatedBy: string;
	tags?: string[];
};

export type DocumentDraft = {
	id: string;
	type: string;
	schemaVersion: number;
	number: string;
	title: string;
	header: Record<string, DocumentValue>;
	lines: Record<string, DocumentLine[]>;
	tags?: string[];
};

export type DocumentRecord = DocumentDraft & {
	state: string;
	version: number;
	metadata: DocumentMetadata;
};

export type DocumentSummary = {
	id: string;
	type: string;
	number: string;
	title: string;
	state: string;
	version: number;
	createdAt: string;
	updatedAt: string;
	createdBy: string;
	updatedBy: string;
	tags?: string[];
};

export type DocumentSearchPage = {
	items: DocumentSummary[];
	nextCursor?: string;
	hasMore: boolean;
};

export type DocumentAttachment = {
	id: string;
	documentId: string;
	file: {
		id: string;
		name: string;
		size: number;
		mime: string;
	};
	addedBy: WorkflowActor;
	addedAt: string;
	addedSequence: number;
	removedBy?: WorkflowActor;
	removedAt?: string;
	removedSequence?: number;
};

export type DocumentComment = {
	id: string;
	documentId: string;
	body: string;
	author: WorkflowActor;
	createdAt: string;
	sequence: number;
};

export type DocumentTimelineKind = "action" | "attachment_added" | "attachment_removed" | "comment_added";

export type DocumentTimelineEvent = {
	id: string;
	sequence: number;
	documentId: string;
	kind: DocumentTimelineKind;
	action?: string;
	attachmentId?: string;
	fileId?: string;
	commentId?: string;
	actor: WorkflowActor;
	occurredAt: string;
};

export type DocumentPrintPayload = {
	schema: DocumentSchema;
	document: DocumentRecord;
	workflow: WorkflowInstance;
	redactedFields?: string[];
	generatedAt: string;
};

export type DocumentKitState = "ready" | "loading" | "empty" | "error" | "forbidden";

export type DocumentActionRequest = {
	action: string;
	comment: string;
	taskId?: string;
	target?: WorkflowActor;
};

export type DocumentFormErrors = Record<string, string>;

export type DocumentKitMessages = {
	search: string;
	searchPlaceholder: string;
	state: string;
	allStates: string;
	refresh: string;
	create: string;
	open: string;
	previous: string;
	next: string;
	noData: string;
	requestFailed: string;
	forbidden: string;
	number: string;
	title: string;
	updatedAt: string;
	createdBy: string;
	actions: string;
	documentInfo: string;
	workflow: string;
	attachments: string;
	timeline: string;
	comments: string;
	addLine: string;
	removeLine: string;
	saveDraft: string;
	submit: string;
	cancel: string;
	approve: string;
	reject: string;
	withdraw: string;
	delegate: string;
	comment: string;
	commentRequired: string;
	delegateTarget: string;
	delegateTargetRequired: string;
	print: string;
	export: string;
	redacted: string;
	sensitiveHidden: string;
	generatedAt: string;
	version: string;
	status: string;
	createdAt: string;
	updatedBy: string;
	fileRemoved: string;
	addAttachment: string;
	downloadAttachment: string;
	removeAttachment: string;
	addComment: string;
	noAttachments: string;
	noComments: string;
	noTimeline: string;
	yes: string;
	no: string;
	fieldRequired: (label: string) => string;
	fieldInvalid: (label: string) => string;
	lineMinimum: (name: string, minimum: number) => string;
	lineMaximum: (name: string, maximum: number) => string;
};
