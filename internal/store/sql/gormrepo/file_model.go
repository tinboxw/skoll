package gormrepo

import (
	"encoding/json"
	"strings"
	"time"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type FileObjectModel struct {
	ID             string `gorm:"size:64;primaryKey"`
	ObjectKey      string `gorm:"size:256;uniqueIndex"`
	Name           string `gorm:"size:255"`
	SizeBytes      int64
	MIME           string `gorm:"size:128"`
	Hash           string `gorm:"size:256;index:idx_file_objects_hash"`
	OwnerType      string `gorm:"size:64;index:idx_file_objects_owner"`
	OwnerID        string `gorm:"size:64;index:idx_file_objects_owner"`
	Visibility     string `gorm:"size:32;index:idx_file_objects_visibility_status"`
	StorageDriver  string `gorm:"size:64;index:idx_file_objects_storage_status"`
	Status         string `gorm:"size:32;index:idx_file_objects_visibility_status;index:idx_file_objects_storage_status;index:idx_file_objects_status_updated"`
	SourceModule   string `gorm:"size:64;index:idx_file_objects_source"`
	SourcePluginID string `gorm:"size:64;index:idx_file_objects_source"`
	MetadataJSON   string `gorm:"type:text"`
	CreatedAt      time.Time
	UpdatedAt      time.Time `gorm:"index:idx_file_objects_status_updated"`
}

func (FileObjectModel) TableName() string { return "sk_file_objects" }

func FileObjectModelFromDomain(object domainfile.FileObject) FileObjectModel {
	metadata, _ := json.Marshal(domainfile.NormalizeObjectMetadata(object.Metadata))
	return FileObjectModel{
		ID:             object.ID.String(),
		ObjectKey:      strings.TrimSpace(object.Key),
		Name:           strings.TrimSpace(object.Name),
		SizeBytes:      object.Size,
		MIME:           strings.TrimSpace(object.MIME),
		Hash:           strings.TrimSpace(object.Hash),
		OwnerType:      strings.TrimSpace(object.Owner.Type),
		OwnerID:        object.Owner.ID.String(),
		Visibility:     string(object.Visibility),
		StorageDriver:  strings.TrimSpace(object.StorageDriver),
		Status:         string(object.Status),
		SourceModule:   strings.TrimSpace(object.Source.Module),
		SourcePluginID: strings.TrimSpace(object.Source.PluginID),
		MetadataJSON:   string(metadata),
		CreatedAt:      object.Meta.CreatedAt,
		UpdatedAt:      object.Meta.UpdatedAt,
	}
}

func (m FileObjectModel) ToDomain() (*domainfile.FileObject, error) {
	metadata := map[string]string{}
	if strings.TrimSpace(m.MetadataJSON) != "" {
		if err := json.Unmarshal([]byte(m.MetadataJSON), &metadata); err != nil {
			return nil, err
		}
	}
	return domainfile.NewFileObject(domainfile.FileObjectInput{
		ID:            shared.ID(strings.TrimSpace(m.ID)),
		Key:           strings.TrimSpace(m.ObjectKey),
		Name:          strings.TrimSpace(m.Name),
		Size:          m.SizeBytes,
		MIME:          strings.TrimSpace(m.MIME),
		Hash:          strings.TrimSpace(m.Hash),
		Owner:         domainfile.OwnerRef{Type: strings.TrimSpace(m.OwnerType), ID: shared.ID(strings.TrimSpace(m.OwnerID))},
		Visibility:    domainfile.Visibility(strings.TrimSpace(m.Visibility)),
		StorageDriver: strings.TrimSpace(m.StorageDriver),
		Status:        domainfile.Status(strings.TrimSpace(m.Status)),
		Source:        domainfile.SourceRef{Module: strings.TrimSpace(m.SourceModule), PluginID: strings.TrimSpace(m.SourcePluginID)},
		Metadata:      metadata,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	})
}
