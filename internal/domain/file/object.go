package file

import (
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type Visibility string

const (
	VisibilityPrivate     Visibility = "private"
	VisibilityPublic      Visibility = "public"
	VisibilityPluginAsset Visibility = "plugin_asset"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusAvailable Status = "available"
	StatusFailed    Status = "failed"
	StatusDeleted   Status = "deleted"
)

type OwnerRef struct {
	Type string
	ID   shared.ID
}

type SourceRef struct {
	Module   string
	PluginID string
}

type FileObjectInput struct {
	ID            shared.ID
	Key           string
	Name          string
	Size          int64
	MIME          string
	Hash          string
	Owner         OwnerRef
	Visibility    Visibility
	StorageDriver string
	Status        Status
	Source        SourceRef
	Metadata      map[string]string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type FileObject struct {
	ID            shared.ID
	Key           string
	Name          string
	Size          int64
	MIME          string
	Hash          string
	Owner         OwnerRef
	Visibility    Visibility
	StorageDriver string
	Status        Status
	Source        SourceRef
	Metadata      map[string]string
	Meta          shared.AuditMeta
}

func NewFileObject(in FileObjectInput) (*FileObject, error) {
	in = NormalizeFileObjectInput(in)
	if err := ValidateFileObjectInput(in); err != nil {
		return nil, err
	}
	obj := &FileObject{
		ID:            in.ID,
		Key:           in.Key,
		Name:          in.Name,
		Size:          in.Size,
		MIME:          in.MIME,
		Hash:          in.Hash,
		Owner:         in.Owner,
		Visibility:    in.Visibility,
		StorageDriver: in.StorageDriver,
		Status:        in.Status,
		Source:        in.Source,
		Metadata:      in.Metadata,
	}
	obj.Meta.CreatedAt = in.CreatedAt
	obj.Meta.UpdatedAt = in.UpdatedAt
	return obj, nil
}

func (o *FileObject) MarkAvailable(hash string, now time.Time) error {
	hash = NormalizeHash(hash)
	if err := ValidateHash(hash); err != nil {
		return err
	}
	o.Hash = hash
	o.Status = StatusAvailable
	o.Meta.Touch(now)
	return nil
}

func (o *FileObject) MarkFailed(now time.Time) {
	o.Status = StatusFailed
	o.Meta.Touch(now)
}

func (o *FileObject) MarkDeleted(now time.Time) {
	o.Status = StatusDeleted
	o.Meta.Touch(now)
}

func NormalizeFileObjectInput(in FileObjectInput) FileObjectInput {
	in.Key = NormalizeKey(in.Key)
	in.Name = strings.TrimSpace(in.Name)
	in.MIME = NormalizeMIME(in.MIME)
	in.Hash = NormalizeHash(in.Hash)
	in.Owner = NormalizeOwner(in.Owner)
	in.StorageDriver = NormalizeIdentifier(in.StorageDriver)
	in.Source = NormalizeSource(in.Source)
	in.Metadata = NormalizeObjectMetadata(in.Metadata)
	if in.Status == "" {
		in.Status = StatusPending
	}
	if in.Visibility == "" {
		in.Visibility = VisibilityPrivate
	}
	if in.UpdatedAt.IsZero() {
		in.UpdatedAt = in.CreatedAt
	}
	return in
}
