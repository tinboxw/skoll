package file

import (
	"context"
	"fmt"
	"time"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	"github.com/tinboxw/skoll/internal/domain/shared"
	filerepo "github.com/tinboxw/skoll/internal/repository/file"
)

type serviceImpl struct {
	repo    filerepo.FileRepository
	objects domainfile.ObjectStore
	now     func() time.Time
	newID   func() shared.ID
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
	return &serviceImpl{repo: repo, objects: objects, now: now, newID: newID}
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
