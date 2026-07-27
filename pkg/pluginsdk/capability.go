package pluginsdk

import (
	"fmt"
	"strings"
)

type HostCapability string

const (
	HostCapabilityTransactionsWithin          HostCapability = "transactions.within"
	HostCapabilityScopesResolve               HostCapability = "scopes.resolve"
	HostCapabilityDatastoreQuery              HostCapability = "datastore.query"
	HostCapabilityDatastoreMutate             HostCapability = "datastore.mutate"
	HostCapabilityDatastoreAggregate          HostCapability = "datastore.aggregate"
	HostCapabilityEventsPublish               HostCapability = "events.publish"
	HostCapabilityDocumentNumbersPreview      HostCapability = "document-numbers.preview"
	HostCapabilityDocumentNumbersIssue        HostCapability = "document-numbers.issue"
	HostCapabilityDocumentsSubmit             HostCapability = "documents.submit"
	HostCapabilityDocumentsAct                HostCapability = "documents.act"
	HostCapabilityDocumentsGet                HostCapability = "documents.get"
	HostCapabilityDocumentsAddAttachment      HostCapability = "documents.add-attachment"
	HostCapabilityDocumentsRemoveAttachment   HostCapability = "documents.remove-attachment"
	HostCapabilityDocumentsListAttachments    HostCapability = "documents.list-attachments"
	HostCapabilityDocumentsAddComment         HostCapability = "documents.add-comment"
	HostCapabilityDocumentsListComments       HostCapability = "documents.list-comments"
	HostCapabilityDocumentsTimeline           HostCapability = "documents.timeline"
	HostCapabilityDocumentsSearch             HostCapability = "documents.search"
	HostCapabilityDocumentsPrint              HostCapability = "documents.print"
	HostCapabilityDocumentsExport             HostCapability = "documents.export"
	HostCapabilityFilesStore                  HostCapability = "files.store"
	HostCapabilityFilesList                   HostCapability = "files.list"
	HostCapabilityFilesGet                    HostCapability = "files.get"
	HostCapabilityFilesDownload               HostCapability = "files.download"
	HostCapabilityFilesDelete                 HostCapability = "files.delete"
	HostCapabilityAuditRecord                 HostCapability = "audit.record"
	HostCapabilityConfigGet                   HostCapability = "config.get"
	HostCapabilityConfigReplace               HostCapability = "config.replace"
	HostCapabilitySecretsGet                  HostCapability = "secrets.get"
	HostCapabilitySecretsSet                  HostCapability = "secrets.set"
	HostCapabilityWorkflowsCreateDefinition   HostCapability = "workflows.create-definition"
	HostCapabilityWorkflowsGetDefinition      HostCapability = "workflows.get-definition"
	HostCapabilityWorkflowsPublishDefinition  HostCapability = "workflows.publish-definition"
	HostCapabilityWorkflowsStart              HostCapability = "workflows.start"
	HostCapabilityWorkflowsGetInstance        HostCapability = "workflows.get-instance"
	HostCapabilityWorkflowsApprove            HostCapability = "workflows.approve"
	HostCapabilityWorkflowsReject             HostCapability = "workflows.reject"
	HostCapabilityWorkflowsWithdraw           HostCapability = "workflows.withdraw"
	HostCapabilityWorkflowsCancel             HostCapability = "workflows.cancel"
	HostCapabilityWorkflowsDelegate           HostCapability = "workflows.delegate"
	HostCapabilityWorkflowsCopy               HostCapability = "workflows.copy"
	HostCapabilityWorkflowsCreateSubstitution HostCapability = "workflows.create-substitution"
	HostCapabilityWorkflowsRevokeSubstitution HostCapability = "workflows.revoke-substitution"
	HostCapabilityJobsSchedule                HostCapability = "jobs.schedule"
	HostCapabilityJobsLeaseDue                HostCapability = "jobs.lease-due"
	HostCapabilityJobsComplete                HostCapability = "jobs.complete"
	HostCapabilityJobsFail                    HostCapability = "jobs.fail"
	HostCapabilityJobsGet                     HostCapability = "jobs.get"
	HostCapabilityJobsList                    HostCapability = "jobs.list"
)

var currentHostCapabilities = map[HostCapability]struct{}{
	HostCapabilityTransactionsWithin: {}, HostCapabilityScopesResolve: {},
	HostCapabilityDatastoreQuery: {}, HostCapabilityDatastoreMutate: {}, HostCapabilityDatastoreAggregate: {},
	HostCapabilityEventsPublish:          {},
	HostCapabilityDocumentNumbersPreview: {}, HostCapabilityDocumentNumbersIssue: {},
	HostCapabilityDocumentsSubmit: {}, HostCapabilityDocumentsAct: {}, HostCapabilityDocumentsGet: {},
	HostCapabilityDocumentsAddAttachment: {}, HostCapabilityDocumentsRemoveAttachment: {}, HostCapabilityDocumentsListAttachments: {},
	HostCapabilityDocumentsAddComment: {}, HostCapabilityDocumentsListComments: {}, HostCapabilityDocumentsTimeline: {},
	HostCapabilityDocumentsSearch: {}, HostCapabilityDocumentsPrint: {}, HostCapabilityDocumentsExport: {},
	HostCapabilityFilesStore: {}, HostCapabilityFilesList: {}, HostCapabilityFilesGet: {},
	HostCapabilityFilesDownload: {}, HostCapabilityFilesDelete: {},
	HostCapabilityAuditRecord: {}, HostCapabilityConfigGet: {}, HostCapabilityConfigReplace: {},
	HostCapabilitySecretsGet: {}, HostCapabilitySecretsSet: {},
	HostCapabilityWorkflowsCreateDefinition: {}, HostCapabilityWorkflowsGetDefinition: {}, HostCapabilityWorkflowsPublishDefinition: {},
	HostCapabilityWorkflowsStart: {}, HostCapabilityWorkflowsGetInstance: {}, HostCapabilityWorkflowsApprove: {},
	HostCapabilityWorkflowsReject: {}, HostCapabilityWorkflowsWithdraw: {}, HostCapabilityWorkflowsCancel: {},
	HostCapabilityWorkflowsDelegate: {}, HostCapabilityWorkflowsCopy: {}, HostCapabilityWorkflowsCreateSubstitution: {},
	HostCapabilityWorkflowsRevokeSubstitution: {},
	HostCapabilityJobsSchedule:                {}, HostCapabilityJobsLeaseDue: {}, HostCapabilityJobsComplete: {},
	HostCapabilityJobsFail: {}, HostCapabilityJobsGet: {}, HostCapabilityJobsList: {},
}

func ParseHostCapability(value string) (HostCapability, error) {
	capability := HostCapability(strings.TrimSpace(strings.ToLower(value)))
	if err := capability.Validate(); err != nil {
		return "", err
	}
	return capability, nil
}

func (c HostCapability) Validate() error {
	if _, exists := currentHostCapabilities[c]; !exists {
		return fmt.Errorf("plugin host capability %q is not declared by the current SDK", c)
	}
	return nil
}
