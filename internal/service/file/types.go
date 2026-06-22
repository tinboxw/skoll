package file

import (
	"context"
	"io"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	filerepo "github.com/tinboxw/skoll/internal/repository/file"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
)

const AccessActionRead = "read"
const AccessActionDownload = "download"
const AccessActionDelete = "delete"

type UploadInput struct {
	Key           string
	Name          string
	Size          int64
	MIME          string
	Hash          string
	Body          io.Reader
	Owner         domainfile.OwnerRef
	Visibility    domainfile.Visibility
	StorageDriver string
	Source        domainfile.SourceRef
	Metadata      map[string]string
	Actor         domainaudit.ActorRef
	Trace         domainaudit.TraceContext
	AuditMetadata map[string]any
}

type PermissionChecker interface {
	CheckPermission(ctx context.Context, in rbacsvc.CheckPermissionInput) (bool, error)
}

type AuditEventSink interface {
	AppendEvent(ctx context.Context, event *domainaudit.Event) error
}

type AccessInput struct {
	FileID      shared.ID
	Key         string
	Object      *domainfile.FileObject
	Action      string
	SubjectType domainrbac.SubjectType
	SubjectID   shared.ID
	ActorName   string
	Trace       domainaudit.TraceContext
	Metadata    map[string]any
}

type AccessDecision struct {
	Allowed  bool
	Resource string
	Reason   string
}

type ListInput struct {
	Filter filerepo.ListFilter
	Offset int
	Limit  int
}

type GetInput struct {
	FileID      shared.ID
	SubjectType domainrbac.SubjectType
	SubjectID   shared.ID
	ActorName   string
	Trace       domainaudit.TraceContext
	Metadata    map[string]any
}

type DownloadInput struct {
	FileID      shared.ID
	ExpiresIn   time.Duration
	SubjectType domainrbac.SubjectType
	SubjectID   shared.ID
	ActorName   string
	Trace       domainaudit.TraceContext
	Metadata    map[string]any
}

type DownloadResult struct {
	Object   *domainfile.FileObject
	Presign  domainfile.PresignedObject
	Decision AccessDecision
}

type DeleteInput struct {
	FileID      shared.ID
	SubjectType domainrbac.SubjectType
	SubjectID   shared.ID
	ActorName   string
	Trace       domainaudit.TraceContext
	Metadata    map[string]any
}

type MultipartInitInput struct {
	Key           string
	Name          string
	Size          int64
	MIME          string
	Hash          string
	Owner         domainfile.OwnerRef
	Visibility    domainfile.Visibility
	StorageDriver string
	Source        domainfile.SourceRef
	Metadata      map[string]string
	TTL           time.Duration
	Actor         domainaudit.ActorRef
	Trace         domainaudit.TraceContext
	AuditMetadata map[string]any
}

type MultipartInitResult struct {
	Object *domainfile.FileObject
	Upload domainfile.MultipartUpload
}

type MultipartUploadPartInput struct {
	UploadID      string
	Key           string
	PartNumber    int
	Size          int64
	Hash          string
	Body          io.Reader
	Trace         domainaudit.TraceContext
	AuditMetadata map[string]any
}

type MultipartCompleteInput struct {
	FileID        shared.ID
	UploadID      string
	Key           string
	ExpectedSize  int64
	ExpectedHash  string
	Parts         []domainfile.MultipartPart
	Actor         domainaudit.ActorRef
	Trace         domainaudit.TraceContext
	AuditMetadata map[string]any
}

type MultipartAbortInput struct {
	FileID        shared.ID
	UploadID      string
	Key           string
	Actor         domainaudit.ActorRef
	Trace         domainaudit.TraceContext
	AuditMetadata map[string]any
}

type Options struct {
	Now        func() time.Time
	NewID      func() shared.ID
	NewAuditID func() shared.ID
	Permission PermissionChecker
	Audit      AuditEventSink
	Multipart  domainfile.MultipartStore
}
