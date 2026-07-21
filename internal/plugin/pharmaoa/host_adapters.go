package pharmaoa

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	filesvc "github.com/tinboxw/skoll/internal/service/file"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

var errPluginAuditQueryUnsupported = fmt.Errorf("plugin audit history is available only through host audit APIs")

type hostAuditAdapter struct {
	audit pluginsdk.AuditService
}

func newHostAuditAdapter(audit pluginsdk.AuditService) *hostAuditAdapter {
	return &hostAuditAdapter{audit: audit}
}

func (a *hostAuditAdapter) Append(ctx context.Context, actorID, action, resource, resourceID string, detail map[string]any) (*domainaudit.Record, error) {
	metadata := make(map[string]any, len(detail)+1)
	for key, value := range detail {
		metadata[key] = value
	}
	if actorID = strings.TrimSpace(actorID); actorID != "" {
		metadata["reportedActorId"] = actorID
	}
	action = pluginAuditName(action, "action")
	resource = pluginAuditName(resource, "resource")
	receipt, err := a.audit.Record(ctx, pluginsdk.AuditEntry{
		Action: action, Resource: resource, ResourceID: strings.TrimSpace(resourceID),
		Risk: pluginsdk.AuditRiskMedium, Detail: metadata,
	})
	if err != nil {
		return nil, err
	}
	return &domainaudit.Record{
		ID: shared.ID(receipt.ID), ActorID: shared.ID(actorID), Action: action, Resource: resource,
		ResourceID: strings.TrimSpace(resourceID), Detail: metadata, OccurredAt: receipt.OccurredAt,
	}, nil
}

func (a *hostAuditAdapter) GetByID(context.Context, string) (*domainaudit.Record, error) {
	return nil, errPluginAuditQueryUnsupported
}

func (a *hostAuditAdapter) ListByActor(context.Context, string, int) ([]*domainaudit.Record, error) {
	return nil, errPluginAuditQueryUnsupported
}

func (a *hostAuditAdapter) ListByTimeRange(context.Context, time.Time, time.Time, int) ([]*domainaudit.Record, error) {
	return nil, errPluginAuditQueryUnsupported
}

func (a *hostAuditAdapter) ClearByTimeRange(context.Context, time.Time, time.Time) (int, error) {
	return 0, errPluginAuditQueryUnsupported
}

func pluginAuditName(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, PluginID+".")
	value = strings.TrimPrefix(value, PluginID+"_")
	if value == "" {
		return fallback
	}
	return value
}

type hostFileAdapter struct {
	files pluginsdk.FileService
}

func newHostFileAdapter(files pluginsdk.FileService) *hostFileAdapter {
	return &hostFileAdapter{files: files}
}

func (a *hostFileAdapter) Get(ctx context.Context, in filesvc.GetInput) (*domainfile.FileObject, filesvc.AccessDecision, error) {
	item, err := a.files.Get(ctx, in.FileID.String())
	if err != nil {
		return nil, filesvc.AccessDecision{Allowed: false, Reason: "host file access denied"}, err
	}
	return pluginDomainFile(item), filesvc.AccessDecision{Allowed: true, Resource: item.ID}, nil
}

func (a *hostFileAdapter) Upload(ctx context.Context, in filesvc.UploadInput) (*domainfile.FileObject, error) {
	if in.Body == nil {
		return nil, fmt.Errorf("plugin file content is required")
	}
	content, err := io.ReadAll(io.LimitReader(in.Body, domainfile.MaxFileSizeBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > domainfile.MaxFileSizeBytes {
		return nil, domainfile.ErrFileTooLarge
	}
	visibility, err := pluginFileVisibility(in.Visibility)
	if err != nil {
		return nil, err
	}
	item, err := a.files.Store(ctx, pluginsdk.FileWrite{
		Key: in.Key, Name: in.Name, Content: content, Visibility: visibility, Metadata: in.Metadata,
	})
	if err != nil {
		return nil, err
	}
	return pluginDomainFile(item), nil
}

func pluginFileVisibility(value domainfile.Visibility) (pluginsdk.FileVisibility, error) {
	switch value {
	case "", domainfile.VisibilityPrivate:
		return pluginsdk.FileVisibilityPrivate, nil
	case domainfile.VisibilityPluginAsset:
		return pluginsdk.FileVisibilityPluginAsset, nil
	default:
		return "", fmt.Errorf("plugin file visibility is invalid")
	}
}

func pluginDomainFile(item pluginsdk.FileObject) *domainfile.FileObject {
	return &domainfile.FileObject{
		ID: shared.ID(item.ID), Key: item.Key, Name: item.Name, Size: item.Size, MIME: item.MIME, Hash: item.Hash,
		Visibility: domainfile.Visibility(item.Visibility), Status: domainfile.Status(item.Status),
		Source: domainfile.SourceRef{Module: "plugin", PluginID: PluginID}, Metadata: item.Metadata,
		Owner: domainfile.OwnerRef{Type: string(domainrbac.SubjectUser)}, StorageDriver: "host",
		Meta: shared.AuditMeta{CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt},
	}
}
