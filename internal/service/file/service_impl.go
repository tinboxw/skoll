package file

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	permission PermissionChecker
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
	return &serviceImpl{repo: repo, objects: objects, now: now, newID: newID, permission: opts.Permission}
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
		return nil, err
	}

	if err := object.MarkAvailable(info.Hash, now); err != nil {
		_ = s.objects.Delete(ctx, info.Key)
		return nil, err
	}
	if err := s.repo.Upsert(ctx, object); err != nil {
		if cleanupErr := s.objects.Delete(ctx, info.Key); cleanupErr != nil {
			return nil, fmt.Errorf("file metadata write failed: %w; object cleanup failed: %v", err, cleanupErr)
		}
		return nil, err
	}
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
		return AccessDecision{Resource: resource, Reason: "not_available"}, nil
	}
	if object.Visibility == domainfile.VisibilityPublic {
		return AccessDecision{Allowed: true, Resource: resource, Reason: "public"}, nil
	}
	if accessSubjectOwnsObject(in, *object) {
		return AccessDecision{Allowed: true, Resource: resource, Reason: "owner"}, nil
	}
	if s.permission == nil || in.SubjectID.IsZero() {
		return AccessDecision{Resource: resource, Reason: "permission_required"}, nil
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
		return AccessDecision{Resource: resource, Reason: "permission_check_failed"}, err
	}
	if !allowed {
		return AccessDecision{Resource: resource, Reason: "permission_denied"}, nil
	}
	return AccessDecision{Allowed: true, Resource: resource, Reason: "permission"}, nil
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
