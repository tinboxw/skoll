package file

import (
	"context"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
)

type Service interface {
	Upload(ctx context.Context, in UploadInput) (*domainfile.FileObject, error)
	List(ctx context.Context, in ListInput) ([]domainfile.FileObject, error)
	Get(ctx context.Context, in GetInput) (*domainfile.FileObject, AccessDecision, error)
	Download(ctx context.Context, in DownloadInput) (*DownloadResult, error)
	Delete(ctx context.Context, in DeleteInput) (AccessDecision, error)
	AuthorizeAccess(ctx context.Context, in AccessInput) (AccessDecision, error)
}
