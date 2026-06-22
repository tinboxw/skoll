package file

import (
	"context"
	"fmt"
	"strings"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	filerepo "github.com/tinboxw/skoll/internal/repository/file"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
)

type serviceImpl struct {
	repo       filerepo.FileRepository
	objects    domainfile.ObjectStore
	now        func() time.Time
	newID      func() shared.ID
	newAuditID func() shared.ID
	permission PermissionChecker
	audit      AuditEventSink
}

func NewService(repo filerepo.FileRepository, objects domainfile.ObjectStore, opts Options) Service {
	now := opts.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	newID := opts.NewID
	if newID == nil {
		newID = func() shared.ID { return shared.ID(fmt.Sprintf("file-%d", now().UnixNano())) }
	}
	newAuditID := opts.NewAuditID
	if newAuditID == nil {
		newAuditID = func() shared.ID { return shared.ID(fmt.Sprintf("file-audit-%d", now().UnixNano())) }
	}
	return &serviceImpl{repo: repo, objects: objects, now: now, newID: newID, newAuditID: newAuditID, permission: opts.Permission, audit: opts.Audit}
}

func (s *serviceImpl) Upload(ctx context.Context, in UploadInput) (*domainfile.FileObject, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("file repository is not configured")
	}
	if s.objects == nil {
		return nil, fmt.Errorf("object store is not configured")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	now := s.now()
	object, err := domainfile.NewFileObject(domainfile.FileObjectInput{
		ID:            s.newID(),
		Key:           in.Key,
		Name:          in.Name,
		Size:          in.Size,
		MIME:          in.MIME,
		Hash:          in.Hash,
		Owner:         in.Owner,
		Visibility:    in.Visibility,
		StorageDriver: in.StorageDriver,
		Status:        domainfile.StatusPending,
		Source:        in.Source,
		Metadata:      in.Metadata,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		return nil, err
	}

	info, err := s.objects.Put(ctx, domainfile.PutObjectInput{
		Key:      object.Key,
		Size:     object.Size,
		MIME:     object.MIME,
		Hash:     object.Hash,
		Body:     in.Body,
		Metadata: object.Metadata,
	})
	if err != nil {
		s.appendFileAudit(ctx, fileAuditInput{
			Action:   "upload",
			Actor:    uploadAuditActor(in, *object),
			Resource: auditResourceFromObject(*object),
			Result:   domainaudit.EventResultFailure,
			Risk:     domainaudit.EventRiskMedium,
			Trace:    in.Trace,
			Metadata: mergeAuditMetadata(in.AuditMetadata, map[string]any{"reason": "object_write_failed"}),
			Object:   object,
		})
		return nil, err
	}

	if err := object.MarkAvailable(info.Hash, now); err != nil {
		_ = s.objects.Delete(ctx, info.Key)
		s.appendFileAudit(ctx, fileAuditInput{
			Action:   "upload",
			Actor:    uploadAuditActor(in, *object),
			Resource: auditResourceFromObject(*object),
			Result:   domainaudit.EventResultFailure,
			Risk:     domainaudit.EventRiskMedium,
			Trace:    in.Trace,
			Metadata: mergeAuditMetadata(in.AuditMetadata, map[string]any{"reason": "mark_available_failed"}),
			Object:   object,
		})
		return nil, err
	}
	if err := s.repo.Upsert(ctx, object); err != nil {
		if cleanupErr := s.objects.Delete(ctx, info.Key); cleanupErr != nil {
			s.appendFileAudit(ctx, fileAuditInput{
				Action:   "upload",
				Actor:    uploadAuditActor(in, *object),
				Resource: auditResourceFromObject(*object),
				Result:   domainaudit.EventResultFailure,
				Risk:     domainaudit.EventRiskHigh,
				Trace:    in.Trace,
				Metadata: mergeAuditMetadata(in.AuditMetadata, map[string]any{"reason": "metadata_write_failed", "cleanup": "failed"}),
				Object:   object,
			})
			return nil, fmt.Errorf("file metadata write failed: %w; object cleanup failed: %v", err, cleanupErr)
		}
		s.appendFileAudit(ctx, fileAuditInput{
			Action:   "upload",
			Actor:    uploadAuditActor(in, *object),
			Resource: auditResourceFromObject(*object),
			Result:   domainaudit.EventResultFailure,
			Risk:     domainaudit.EventRiskMedium,
			Trace:    in.Trace,
			Metadata: mergeAuditMetadata(in.AuditMetadata, map[string]any{"reason": "metadata_write_failed", "cleanup": "deleted"}),
			Object:   object,
		})
		return nil, err
	}
	s.appendFileAudit(ctx, fileAuditInput{
		Action:   "upload",
		Actor:    uploadAuditActor(in, *object),
		Resource: auditResourceFromObject(*object),
		Result:   domainaudit.EventResultSuccess,
		Risk:     domainaudit.EventRiskLow,
		Trace:    in.Trace,
		Metadata: in.AuditMetadata,
		Object:   object,
	})
	return object, nil
}

func (s *serviceImpl) AuthorizeAccess(ctx context.Context, in AccessInput) (AccessDecision, error) {
	if s == nil || s.repo == nil {
		return AccessDecision{Reason: "repository_not_configured"}, fmt.Errorf("file repository is not configured")
	}
	if err := ctx.Err(); err != nil {
		return AccessDecision{Reason: "context_canceled"}, err
	}
	action := normalizeAccessAction(in.Action)
	object, err := s.resolveAccessObject(ctx, in)
	if err != nil {
		return AccessDecision{Reason: "lookup_failed"}, err
	}
	if object == nil {
		return AccessDecision{Reason: "not_found"}, nil
	}
	resource := accessResource(*object)
	if object.Status != domainfile.StatusAvailable {
		decision := AccessDecision{Resource: resource, Reason: "not_available"}
		s.appendAccessAudit(ctx, in, *object, decision, action)
		return decision, nil
	}
	if object.Visibility == domainfile.VisibilityPublic && action == AccessActionDownload {
		decision := AccessDecision{Allowed: true, Resource: resource, Reason: "public"}
		s.appendAccessAudit(ctx, in, *object, decision, action)
		return decision, nil
	}
	if accessSubjectOwnsObject(in, *object) {
		decision := AccessDecision{Allowed: true, Resource: resource, Reason: "owner"}
		s.appendAccessAudit(ctx, in, *object, decision, action)
		return decision, nil
	}
	if s.permission == nil || in.SubjectID.IsZero() {
		decision := AccessDecision{Resource: resource, Reason: "permission_required"}
		s.appendAccessAudit(ctx, in, *object, decision, action)
		return decision, nil
	}
	subjectType := in.SubjectType
	if subjectType == "" {
		subjectType = domainrbac.SubjectUser
	}
	allowed, err := s.permission.CheckPermission(ctx, rbacsvc.CheckPermissionInput{
		SubjectType: subjectType,
		SubjectID:   in.SubjectID.String(),
		Resource:    resource,
		Action:      action,
	})
	if err != nil {
		decision := AccessDecision{Resource: resource, Reason: "permission_check_failed"}
		s.appendAccessAudit(ctx, in, *object, decision, action)
		return decision, err
	}
	if !allowed {
		decision := AccessDecision{Resource: resource, Reason: "permission_denied"}
		s.appendAccessAudit(ctx, in, *object, decision, action)
		return decision, nil
	}
	decision := AccessDecision{Allowed: true, Resource: resource, Reason: "permission"}
	s.appendAccessAudit(ctx, in, *object, decision, action)
	return decision, nil
}

func (s *serviceImpl) resolveAccessObject(ctx context.Context, in AccessInput) (*domainfile.FileObject, error) {
	if in.Object != nil {
		object := *in.Object
		return &object, nil
	}
	if !in.FileID.IsZero() {
		return s.repo.Get(ctx, in.FileID)
	}
	key := strings.TrimSpace(in.Key)
	if key != "" {
		return s.repo.GetByKey(ctx, key)
	}
	return nil, fmt.Errorf("file id or key is required")
}

func normalizeAccessAction(action string) string {
	action = strings.ToLower(strings.TrimSpace(action))
	if action == "" {
		return AccessActionDownload
	}
	return action
}

func accessSubjectOwnsObject(in AccessInput, object domainfile.FileObject) bool {
	if in.SubjectID.IsZero() || object.Owner.ID.IsZero() || in.SubjectID != object.Owner.ID {
		return false
	}
	subjectType := in.SubjectType
	if subjectType == "" {
		subjectType = domainrbac.SubjectUser
	}
	return string(subjectType) == object.Owner.Type
}

func accessResource(object domainfile.FileObject) string {
	if object.Visibility == domainfile.VisibilityPluginAsset && object.Source.PluginID != "" {
		return "plugin:" + object.Source.PluginID + ":asset"
	}
	return "file:" + string(object.Visibility)
}

type fileAuditInput struct {
	Action   string
	Actor    domainaudit.ActorRef
	Resource domainaudit.ResourceRef
	Result   domainaudit.EventResult
	Risk     domainaudit.EventRisk
	Trace    domainaudit.TraceContext
	Metadata map[string]any
	Object   *domainfile.FileObject
}

func (s *serviceImpl) appendAccessAudit(ctx context.Context, in AccessInput, object domainfile.FileObject, decision AccessDecision, action string) {
	auditAction := action
	result := domainaudit.EventResultSuccess
	risk := domainaudit.EventRiskLow
	metadata := mergeAuditMetadata(in.Metadata, map[string]any{
		"accessResource":  decision.Resource,
		"decisionReason":  decision.Reason,
		"requestedAction": action,
	})
	if !decision.Allowed {
		auditAction = "forbidden"
		result = domainaudit.EventResultDenied
		risk = domainaudit.EventRiskHigh
	}
	s.appendFileAudit(ctx, fileAuditInput{
		Action:   auditAction,
		Actor:    accessAuditActor(in),
		Resource: auditResourceFromObject(object),
		Result:   result,
		Risk:     risk,
		Trace:    in.Trace,
		Metadata: metadata,
		Object:   &object,
	})
}

func (s *serviceImpl) appendFileAudit(ctx context.Context, in fileAuditInput) {
	if s == nil || s.audit == nil {
		return
	}
	object := in.Object
	sourceData := map[string]any{"kind": "file_object"}
	if object != nil {
		sourceData["fileId"] = object.ID.String()
		sourceData["key"] = object.Key
		sourceData["name"] = object.Name
		sourceData["size"] = object.Size
		sourceData["mime"] = object.MIME
		sourceData["hash"] = object.Hash
		sourceData["visibility"] = string(object.Visibility)
		sourceData["status"] = string(object.Status)
		sourceData["storageDriver"] = object.StorageDriver
		sourceData["sourceModule"] = object.Source.Module
		sourceData["sourcePluginId"] = object.Source.PluginID
	}
	event, err := domainaudit.NewEvent(domainaudit.EventInput{
		ID:         s.newAuditID(),
		Type:       domainaudit.EventTypeOperation,
		Action:     domainaudit.AuditAction("file.object." + normalizeAuditOperation(in.Action)),
		Actor:      in.Actor,
		Resource:   in.Resource,
		Result:     in.Result,
		Trace:      in.Trace,
		Risk:       in.Risk,
		Metadata:   in.Metadata,
		SourceData: sourceData,
		OccurredAt: s.now(),
	})
	if err != nil {
		return
	}
	_ = s.audit.AppendEvent(ctx, event)
}

func uploadAuditActor(in UploadInput, object domainfile.FileObject) domainaudit.ActorRef {
	if strings.TrimSpace(in.Actor.Type) != "" {
		return in.Actor
	}
	return domainaudit.ActorRef{Type: object.Owner.Type, ID: object.Owner.ID}
}

func accessAuditActor(in AccessInput) domainaudit.ActorRef {
	subjectType := in.SubjectType
	if subjectType == "" {
		subjectType = domainrbac.SubjectUser
	}
	if in.SubjectID.IsZero() {
		return domainaudit.ActorRef{Type: "anonymous", Name: strings.TrimSpace(in.ActorName)}
	}
	return domainaudit.ActorRef{Type: string(subjectType), ID: in.SubjectID, Name: strings.TrimSpace(in.ActorName)}
}

func auditResourceFromObject(object domainfile.FileObject) domainaudit.ResourceRef {
	return domainaudit.ResourceRef{Type: "file_object", ID: object.ID.String(), Name: object.Name}
}

func normalizeAuditOperation(action string) string {
	action = strings.ToLower(strings.TrimSpace(action))
	if action == "" {
		return AccessActionDownload
	}
	switch action {
	case "upload", AccessActionDownload, AccessActionDelete, "forbidden":
		return action
	default:
		return strings.ReplaceAll(action, "-", "_")
	}
}

func mergeAuditMetadata(base map[string]any, extra map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(extra))
	for key, value := range base {
		if strings.TrimSpace(key) != "" {
			out[key] = value
		}
	}
	for key, value := range extra {
		if strings.TrimSpace(key) != "" {
			out[key] = value
		}
	}
	return out
}
