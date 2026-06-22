package file

import (
	"context"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
)

type Service interface {
	Upload(ctx context.Context, in UploadInput) (*domainfile.FileObject, error)
	InitMultipart(ctx context.Context, in MultipartInitInput) (*MultipartInitResult, error)
	UploadMultipartPart(ctx context.Context, in MultipartUploadPartInput) (domainfile.MultipartPart, error)
	CompleteMultipart(ctx context.Context, in MultipartCompleteInput) (*domainfile.FileObject, error)
	AbortMultipart(ctx context.Context, in MultipartAbortInput) error
	List(ctx context.Context, in ListInput) ([]domainfile.FileObject, error)
	Get(ctx context.Context, in GetInput) (*domainfile.FileObject, AccessDecision, error)
	Download(ctx context.Context, in DownloadInput) (*DownloadResult, error)
	Delete(ctx context.Context, in DeleteInput) (AccessDecision, error)
	AuthorizeAccess(ctx context.Context, in AccessInput) (AccessDecision, error)
}
