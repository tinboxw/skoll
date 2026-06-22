package file

import (
	"context"
	"io"
	"time"
)

type ObjectStore interface {
	Put(ctx context.Context, in PutObjectInput) (ObjectInfo, error)
	Get(ctx context.Context, key string) (ObjectStream, error)
	Delete(ctx context.Context, key string) error
	Stat(ctx context.Context, key string) (ObjectInfo, error)
	Presign(ctx context.Context, in PresignInput) (PresignedObject, error)
}

type ObjectInfo struct {
	Key          string
	Size         int64
	MIME         string
	Hash         string
	ETag         string
	LastModified time.Time
	Metadata     map[string]string
}

type ObjectStream struct {
	Info ObjectInfo
	Body io.ReadCloser
}

type PutObjectInput struct {
	Key      string
	Size     int64
	MIME     string
	Hash     string
	Body     io.Reader
	Metadata map[string]string
}

type PresignOperation string

const (
	PresignOperationGet    PresignOperation = "get"
	PresignOperationPut    PresignOperation = "put"
	PresignOperationDelete PresignOperation = "delete"
)

type PresignInput struct {
	Key       string
	Operation PresignOperation
	ExpiresIn time.Duration
	MIME      string
}

type PresignedObject struct {
	Key       string
	Operation PresignOperation
	URL       string
	ExpiresAt time.Time
	Headers   map[string]string
}

func NormalizeObjectInfo(info ObjectInfo) ObjectInfo {
	info.Key = NormalizeKey(info.Key)
	info.MIME = NormalizeMIME(info.MIME)
	info.Hash = NormalizeHash(info.Hash)
	info.ETag = NormalizeHash(info.ETag)
	info.Metadata = NormalizeObjectMetadata(info.Metadata)
	return info
}

func NormalizePutObjectInput(in PutObjectInput) PutObjectInput {
	in.Key = NormalizeKey(in.Key)
	in.MIME = NormalizeMIME(in.MIME)
	in.Hash = NormalizeHash(in.Hash)
	in.Metadata = NormalizeObjectMetadata(in.Metadata)
	return in
}

func NormalizePresignInput(in PresignInput) PresignInput {
	in.Key = NormalizeKey(in.Key)
	in.MIME = NormalizeMIME(in.MIME)
	return in
}
