package file

import (
	"context"
	"io"
	"time"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
)

const AccessActionDownload = "download"

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
}

type PermissionChecker interface {
	CheckPermission(ctx context.Context, in rbacsvc.CheckPermissionInput) (bool, error)
}

type AccessInput struct {
	FileID      shared.ID
	Key         string
	Object      *domainfile.FileObject
	Action      string
	SubjectType domainrbac.SubjectType
	SubjectID   shared.ID
}

type AccessDecision struct {
	Allowed  bool
	Resource string
	Reason   string
}

type Options struct {
	Now        func() time.Time
	NewID      func() shared.ID
	Permission PermissionChecker
}
