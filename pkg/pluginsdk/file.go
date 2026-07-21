package pluginsdk

import (
	"context"
	"time"
)

type FileVisibility string

const (
	FileVisibilityPrivate     FileVisibility = "private"
	FileVisibilityPluginAsset FileVisibility = "plugin_asset"
)

type FileWrite struct {
	Key        string
	Name       string
	Content    []byte
	Visibility FileVisibility
	Metadata   map[string]string
}

type FileQuery struct {
	Visibility FileVisibility
	Offset     int
	Limit      int
}

type FileObject struct {
	ID         string
	Key        string
	Name       string
	Size       int64
	MIME       string
	Hash       string
	Visibility FileVisibility
	Status     string
	Metadata   map[string]string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type FileDownload struct {
	ID        string
	URL       string
	Method    string
	ExpiresAt time.Time
	Headers   map[string]string
}

type FileService interface {
	Store(ctx context.Context, input FileWrite) (FileObject, error)
	List(ctx context.Context, query FileQuery) ([]FileObject, error)
	Get(ctx context.Context, id string) (FileObject, error)
	Download(ctx context.Context, id string) (FileDownload, error)
	Delete(ctx context.Context, id string) error
}
