package file

import (
	"context"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
)

type Service interface {
	Upload(ctx context.Context, in UploadInput) (*domainfile.FileObject, error)
}
