package file

import (
	"context"
	"time"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type ListFilter struct {
	OwnerType      string
	OwnerID        shared.ID
	Visibility     domainfile.Visibility
	StorageDriver  string
	Status         domainfile.Status
	SourceModule   string
	SourcePluginID string
}

type FileRepository interface {
	Upsert(ctx context.Context, object *domainfile.FileObject) error
	Get(ctx context.Context, id shared.ID) (*domainfile.FileObject, error)
	GetByKey(ctx context.Context, key string) (*domainfile.FileObject, error)
	List(ctx context.Context, filter ListFilter, offset, limit int) ([]domainfile.FileObject, error)
	SetStatus(ctx context.Context, id shared.ID, status domainfile.Status, now time.Time) error
	Delete(ctx context.Context, id shared.ID) error
}
