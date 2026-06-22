package file

import (
	"io"
	"time"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

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

type Options struct {
	Now   func() time.Time
	NewID func() shared.ID
}
