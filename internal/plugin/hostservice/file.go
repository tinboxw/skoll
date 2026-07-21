package hostservice

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	filerepo "github.com/tinboxw/skoll/internal/repository/file"
	filesvc "github.com/tinboxw/skoll/internal/service/file"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const pluginFileDownloadTTL = 5 * time.Minute

var blockedFileExtensions = map[string]struct{}{
	".bat": {}, ".cmd": {}, ".com": {}, ".dll": {}, ".exe": {}, ".jar": {}, ".js": {},
	".msi": {}, ".ps1": {}, ".scr": {}, ".sh": {}, ".vbs": {},
}

type fileBackend interface {
	Upload(ctx context.Context, in filesvc.UploadInput) (*domainfile.FileObject, error)
	List(ctx context.Context, in filesvc.ListInput) ([]domainfile.FileObject, error)
	Get(ctx context.Context, in filesvc.GetInput) (*domainfile.FileObject, filesvc.AccessDecision, error)
	Download(ctx context.Context, in filesvc.DownloadInput) (*filesvc.DownloadResult, error)
	Delete(ctx context.Context, in filesvc.DeleteInput) (filesvc.AccessDecision, error)
}

type fileService struct {
	pluginID string
	files    fileBackend
}

func NewFileService(pluginID string, files fileBackend) (pluginsdk.FileService, error) {
	pluginID = strings.ToLower(strings.TrimSpace(pluginID))
	if pluginID == "" {
		return nil, fmt.Errorf("plugin host identity is required")
	}
	if files == nil {
		return nil, fmt.Errorf("plugin host file backend is required")
	}
	return &fileService{pluginID: pluginID, files: files}, nil
}

func (s *fileService) Store(ctx context.Context, input pluginsdk.FileWrite) (pluginsdk.FileObject, error) {
	name := strings.TrimSpace(input.Name)
	if _, blocked := blockedFileExtensions[strings.ToLower(path.Ext(name))]; blocked {
		return pluginsdk.FileObject{}, fmt.Errorf("plugin executable file upload is forbidden")
	}
	if strings.ContainsAny(name, `/\`) {
		return pluginsdk.FileObject{}, fmt.Errorf("plugin file name must not contain path separators")
	}
	if int64(len(input.Content)) > domainfile.MaxFileSizeBytes {
		return pluginsdk.FileObject{}, domainfile.ErrFileTooLarge
	}
	mime := strings.TrimSpace(strings.Split(http.DetectContentType(input.Content), ";")[0])
	if blockedFileMIME(mime) {
		return pluginsdk.FileObject{}, fmt.Errorf("plugin executable file content is forbidden")
	}
	visibility, err := pluginFileVisibility(input.Visibility)
	if err != nil {
		return pluginsdk.FileObject{}, err
	}
	relativeKey := domainfile.NormalizeKey(input.Key)
	if err := domainfile.ValidateKey(relativeKey); err != nil {
		return pluginsdk.FileObject{}, err
	}
	actor := trustedHostActor(ctx, s.pluginID)
	ownerID := actor.id
	hash := sha256.Sum256(input.Content)
	item, err := s.files.Upload(ctx, filesvc.UploadInput{
		Key:           "plugins/" + s.pluginID + "/" + relativeKey,
		Name:          name,
		Size:          int64(len(input.Content)),
		MIME:          mime,
		Hash:          hex.EncodeToString(hash[:]),
		Body:          bytes.NewReader(input.Content),
		Owner:         domainfile.OwnerRef{Type: actor.typ, ID: shared.ID(ownerID)},
		Visibility:    visibility,
		StorageDriver: "local",
		Source:        domainfile.SourceRef{Module: "plugin", PluginID: s.pluginID},
		Metadata:      cloneStringMap(input.Metadata),
		Actor:         domainaudit.ActorRef{Type: actor.typ, ID: shared.ID(actor.id), Name: actor.name},
		AuditMetadata: map[string]any{"pluginId": s.pluginID},
	})
	if err != nil {
		return pluginsdk.FileObject{}, err
	}
	return pluginFileObject(*item), nil
}

func (s *fileService) List(ctx context.Context, query pluginsdk.FileQuery) ([]pluginsdk.FileObject, error) {
	if query.Offset < 0 || query.Limit < 0 {
		return nil, fmt.Errorf("plugin file pagination is invalid")
	}
	limit := query.Limit
	if limit == 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	visibility, err := pluginFileQueryVisibility(query.Visibility)
	if err != nil {
		return nil, err
	}
	items, err := s.files.List(ctx, filesvc.ListInput{Filter: filerepo.ListFilter{
		Visibility: visibility, SourcePluginID: s.pluginID,
	}, Offset: query.Offset, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]pluginsdk.FileObject, 0, len(items))
	for _, item := range items {
		if item.Source.PluginID == s.pluginID {
			out = append(out, pluginFileObject(item))
		}
	}
	return out, nil
}

func (s *fileService) Get(ctx context.Context, id string) (pluginsdk.FileObject, error) {
	object, err := s.authorizedObject(ctx, id, trustedHostActor(ctx, s.pluginID))
	if err != nil {
		return pluginsdk.FileObject{}, err
	}
	return pluginFileObject(*object), nil
}

func (s *fileService) Download(ctx context.Context, id string) (pluginsdk.FileDownload, error) {
	actor := trustedHostActor(ctx, s.pluginID)
	object, err := s.authorizedObject(ctx, id, actor)
	if err != nil {
		return pluginsdk.FileDownload{}, err
	}
	result, err := s.files.Download(ctx, filesvc.DownloadInput{
		FileID: shared.ID(object.ID.String()), ExpiresIn: pluginFileDownloadTTL,
		SubjectType: domainrbac.SubjectType(actor.typ), SubjectID: shared.ID(actor.id), ActorName: actor.name,
		Metadata: map[string]any{"pluginId": s.pluginID},
	})
	if err != nil {
		return pluginsdk.FileDownload{}, err
	}
	if result == nil || !result.Decision.Allowed || result.Object == nil {
		return pluginsdk.FileDownload{}, fmt.Errorf("plugin file access denied")
	}
	return pluginsdk.FileDownload{
		ID: result.Object.ID.String(), URL: result.Presign.URL, Method: strings.ToUpper(string(result.Presign.Operation)),
		ExpiresAt: result.Presign.ExpiresAt, Headers: cloneStringMap(result.Presign.Headers),
	}, nil
}

func (s *fileService) Delete(ctx context.Context, id string) error {
	actor := trustedHostActor(ctx, s.pluginID)
	object, err := s.authorizedObject(ctx, id, actor)
	if err != nil {
		return err
	}
	decision, err := s.files.Delete(ctx, filesvc.DeleteInput{
		FileID: object.ID, SubjectType: domainrbac.SubjectType(actor.typ), SubjectID: shared.ID(actor.id), ActorName: actor.name,
		Metadata: map[string]any{"pluginId": s.pluginID},
	})
	if err != nil {
		return err
	}
	if !decision.Allowed {
		return fmt.Errorf("plugin file access denied")
	}
	return nil
}

func (s *fileService) authorizedObject(ctx context.Context, id string, actor hostActor) (*domainfile.FileObject, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("plugin file id is required")
	}
	object, decision, err := s.files.Get(ctx, filesvc.GetInput{
		FileID: shared.ID(id), SubjectType: domainrbac.SubjectType(actor.typ), SubjectID: shared.ID(actor.id), ActorName: actor.name,
		Metadata: map[string]any{"pluginId": s.pluginID},
	})
	if err != nil {
		return nil, err
	}
	if !decision.Allowed || object == nil || object.Source.PluginID != s.pluginID {
		return nil, fmt.Errorf("plugin file access denied")
	}
	return object, nil
}

func pluginFileVisibility(value pluginsdk.FileVisibility) (domainfile.Visibility, error) {
	if value == "" {
		return domainfile.VisibilityPrivate, nil
	}
	switch value {
	case pluginsdk.FileVisibilityPrivate:
		return domainfile.VisibilityPrivate, nil
	case pluginsdk.FileVisibilityPluginAsset:
		return domainfile.VisibilityPluginAsset, nil
	default:
		return "", fmt.Errorf("plugin file visibility is invalid")
	}
}

func pluginFileQueryVisibility(value pluginsdk.FileVisibility) (domainfile.Visibility, error) {
	if value == "" {
		return "", nil
	}
	return pluginFileVisibility(value)
}

func pluginFileObject(item domainfile.FileObject) pluginsdk.FileObject {
	return pluginsdk.FileObject{
		ID: item.ID.String(), Key: item.Key, Name: item.Name, Size: item.Size, MIME: item.MIME, Hash: item.Hash,
		Visibility: pluginsdk.FileVisibility(item.Visibility), Status: string(item.Status), Metadata: cloneStringMap(item.Metadata),
		CreatedAt: item.Meta.CreatedAt, UpdatedAt: item.Meta.UpdatedAt,
	}
}

func blockedFileMIME(value string) bool {
	value = strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
	return value == "application/x-msdownload" || value == "application/x-executable" || value == "application/x-sh"
}

func cloneStringMap(values map[string]string) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

var _ pluginsdk.FileService = (*fileService)(nil)
