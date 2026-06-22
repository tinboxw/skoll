package object

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
)

const metadataSuffix = ".meta.json"

var ErrObjectExists = errors.New("object already exists")

type LocalStore struct {
	root string
}

type localMetadata struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	MIME         string            `json:"mime"`
	Hash         string            `json:"hash"`
	ETag         string            `json:"etag"`
	LastModified time.Time         `json:"lastModified"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

func NewLocalStore(root string) (*LocalStore, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("local object store root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absRoot, 0o755); err != nil {
		return nil, err
	}
	return &LocalStore{root: filepath.Clean(absRoot)}, nil
}

func (s *LocalStore) Put(ctx context.Context, in domainfile.PutObjectInput) (domainfile.ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return domainfile.ObjectInfo{}, err
	}
	if err := validateRawLocalKey(in.Key); err != nil {
		return domainfile.ObjectInfo{}, err
	}
	in = domainfile.NormalizePutObjectInput(in)
	if err := domainfile.ValidatePutObjectInput(in); err != nil {
		return domainfile.ObjectInfo{}, err
	}
	target, err := s.resolve(in.Key)
	if err != nil {
		return domainfile.ObjectInfo{}, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return domainfile.ObjectInfo{}, err
	}
	if err := ensureObjectAbsent(target); err != nil {
		return domainfile.ObjectInfo{}, err
	}

	tmp, err := os.CreateTemp(filepath.Dir(target), ".upload-*")
	if err != nil {
		return domainfile.ObjectInfo{}, err
	}
	tmpName := tmp.Name()
	written, copyErr := io.Copy(tmp, in.Body)
	closeErr := tmp.Close()
	if copyErr != nil {
		_ = os.Remove(tmpName)
		return domainfile.ObjectInfo{}, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpName)
		return domainfile.ObjectInfo{}, closeErr
	}
	if written != in.Size {
		_ = os.Remove(tmpName)
		return domainfile.ObjectInfo{}, fmt.Errorf("object size mismatch: wrote %d bytes, want %d", written, in.Size)
	}
	if err := os.Rename(tmpName, target); err != nil {
		_ = os.Remove(tmpName)
		return domainfile.ObjectInfo{}, err
	}

	info := domainfile.ObjectInfo{
		Key:          in.Key,
		Size:         in.Size,
		MIME:         in.MIME,
		Hash:         in.Hash,
		ETag:         in.Hash,
		LastModified: time.Now().UTC(),
		Metadata:     in.Metadata,
	}
	if err := s.writeInfo(info); err != nil {
		return domainfile.ObjectInfo{}, err
	}
	return info, nil
}

func (s *LocalStore) Get(ctx context.Context, key string) (domainfile.ObjectStream, error) {
	if err := ctx.Err(); err != nil {
		return domainfile.ObjectStream{}, err
	}
	info, err := s.Stat(ctx, key)
	if err != nil {
		return domainfile.ObjectStream{}, err
	}
	target, err := s.resolve(info.Key)
	if err != nil {
		return domainfile.ObjectStream{}, err
	}
	body, err := os.Open(target)
	if err != nil {
		return domainfile.ObjectStream{}, err
	}
	return domainfile.ObjectStream{Info: info, Body: body}, nil
}

func (s *LocalStore) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	target, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(metadataPath(target)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *LocalStore) Stat(ctx context.Context, key string) (domainfile.ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return domainfile.ObjectInfo{}, err
	}
	target, err := s.resolve(key)
	if err != nil {
		return domainfile.ObjectInfo{}, err
	}
	return s.readInfo(target)
}

func (s *LocalStore) Presign(ctx context.Context, in domainfile.PresignInput) (domainfile.PresignedObject, error) {
	if err := ctx.Err(); err != nil {
		return domainfile.PresignedObject{}, err
	}
	if err := validateRawLocalKey(in.Key); err != nil {
		return domainfile.PresignedObject{}, err
	}
	in = domainfile.NormalizePresignInput(in)
	if err := domainfile.ValidatePresignInput(in); err != nil {
		return domainfile.PresignedObject{}, err
	}
	if _, err := s.resolve(in.Key); err != nil {
		return domainfile.PresignedObject{}, err
	}
	u := url.URL{
		Scheme: "local",
		Host:   "object",
		Path:   "/" + in.Key,
	}
	query := u.Query()
	query.Set("operation", string(in.Operation))
	query.Set("expires", fmt.Sprintf("%d", time.Now().UTC().Add(in.ExpiresIn).Unix()))
	u.RawQuery = query.Encode()
	return domainfile.PresignedObject{
		Key:       in.Key,
		Operation: in.Operation,
		URL:       u.String(),
		ExpiresAt: time.Now().UTC().Add(in.ExpiresIn),
	}, nil
}

func (s *LocalStore) resolve(key string) (string, error) {
	if err := validateRawLocalKey(key); err != nil {
		return "", err
	}
	key = domainfile.NormalizeKey(key)
	if err := domainfile.ValidateKey(key); err != nil {
		return "", err
	}
	target := filepath.Join(s.root, filepath.FromSlash(key))
	cleanTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(s.root, cleanTarget)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("object key escapes local store root")
	}
	return cleanTarget, nil
}

func (s *LocalStore) writeInfo(info domainfile.ObjectInfo) error {
	info = domainfile.NormalizeObjectInfo(info)
	if err := domainfile.ValidateObjectInfo(info); err != nil {
		return err
	}
	target, err := s.resolve(info.Key)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(localMetadata{
		Key:          info.Key,
		Size:         info.Size,
		MIME:         info.MIME,
		Hash:         info.Hash,
		ETag:         info.ETag,
		LastModified: info.LastModified,
		Metadata:     info.Metadata,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metadataPath(target), raw, 0o644)
}

func (s *LocalStore) readInfo(target string) (domainfile.ObjectInfo, error) {
	raw, err := os.ReadFile(metadataPath(target))
	if err != nil {
		return domainfile.ObjectInfo{}, err
	}
	var meta localMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		return domainfile.ObjectInfo{}, err
	}
	stat, err := os.Stat(target)
	if err != nil {
		return domainfile.ObjectInfo{}, err
	}
	info := domainfile.ObjectInfo{
		Key:          meta.Key,
		Size:         stat.Size(),
		MIME:         meta.MIME,
		Hash:         meta.Hash,
		ETag:         meta.ETag,
		LastModified: meta.LastModified,
		Metadata:     meta.Metadata,
	}
	if err := domainfile.ValidateObjectInfo(info); err != nil {
		return domainfile.ObjectInfo{}, err
	}
	return domainfile.NormalizeObjectInfo(info), nil
}

func metadataPath(target string) string {
	return target + metadataSuffix
}

func validateRawLocalKey(key string) error {
	raw := strings.TrimSpace(key)
	if raw == "" {
		return fmt.Errorf("object key is required")
	}
	if filepath.IsAbs(raw) || strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, `\`) {
		return fmt.Errorf("object key must be relative")
	}
	if strings.Contains(raw, `\`) {
		return fmt.Errorf("object key must use slash separators")
	}
	return nil
}

func ensureObjectAbsent(target string) error {
	if _, err := os.Stat(target); err == nil {
		return ErrObjectExists
	} else if !os.IsNotExist(err) {
		return err
	}
	if _, err := os.Stat(metadataPath(target)); err == nil {
		return ErrObjectExists
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}
