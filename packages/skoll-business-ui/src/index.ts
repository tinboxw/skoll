export * from "./core";

export {
	DocumentApprovalPanel as BusinessWorkflowPanel,
	DocumentAttachments as BusinessFilePanel,
	DocumentComments as BusinessComments,
	DocumentDetail as BusinessDocumentDetail,
	DocumentFieldInput as BusinessFieldInput,
	DocumentForm as BusinessDocumentForm,
	DocumentList as BusinessDocumentList,
	DocumentTimeline as BusinessTimeline
} from "@skoll/document-ui";
export type {
	DocumentActionRequest as BusinessWorkflowActionRequest,
	DocumentAttachment as BusinessAttachment,
	DocumentComment as BusinessComment,
	DocumentDraft as BusinessDocumentDraft,
	DocumentFormErrors as BusinessFormErrors,
	DocumentLine as BusinessLine,
	DocumentLineSchema as BusinessLineSchema,
	DocumentRecord as BusinessDocumentRecord,
	DocumentSchema as BusinessDocumentSchema,
	DocumentTimelineEvent as BusinessTimelineEvent,
	WorkflowInstance as BusinessWorkflowInstance
} from "@skoll/document-ui";
