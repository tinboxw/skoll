package file

import (
	"context"
	"io"
	"strings"
	"time"
)

type MultipartStore interface {
	Init(ctx context.Context, in MultipartInitInput) (MultipartUpload, error)
	UploadPart(ctx context.Context, in MultipartUploadPartInput) (MultipartPart, error)
	Complete(ctx context.Context, in MultipartCompleteInput) (ObjectInfo, error)
	Abort(ctx context.Context, in MultipartAbortInput) error
}

type MultipartInitInput struct {
	Key          string
	Size         int64
	MIME         string
	ExpectedHash string
	Metadata     map[string]string
}

type MultipartUpload struct {
	UploadID  string
	Key       string
	ExpiresAt time.Time
	Metadata  map[string]string
}

type MultipartUploadPartInput struct {
	UploadID   string
	Key        string
	PartNumber int
	Size       int64
	Hash       string
	Body       io.Reader
}

type MultipartPart struct {
	PartNumber int
	Size       int64
	Hash       string
	ETag       string
}

type MultipartCompleteInput struct {
	UploadID     string
	Key          string
	ExpectedSize int64
	ExpectedHash string
	Parts        []MultipartPart
}

type MultipartAbortInput struct {
	UploadID string
	Key      string
}

func NormalizeMultipartInitInput(in MultipartInitInput) MultipartInitInput {
	in.Key = NormalizeKey(in.Key)
	in.MIME = NormalizeMIME(in.MIME)
	in.ExpectedHash = NormalizeHash(in.ExpectedHash)
	in.Metadata = NormalizeObjectMetadata(in.Metadata)
	return in
}

func NormalizeMultipartUpload(upload MultipartUpload) MultipartUpload {
	upload.UploadID = NormalizeUploadID(upload.UploadID)
	upload.Key = NormalizeKey(upload.Key)
	upload.Metadata = NormalizeObjectMetadata(upload.Metadata)
	return upload
}

func NormalizeMultipartUploadPartInput(in MultipartUploadPartInput) MultipartUploadPartInput {
	in.UploadID = NormalizeUploadID(in.UploadID)
	in.Key = NormalizeKey(in.Key)
	in.Hash = NormalizeHash(in.Hash)
	return in
}

func NormalizeMultipartPart(part MultipartPart) MultipartPart {
	part.Hash = NormalizeHash(part.Hash)
	part.ETag = NormalizeHash(part.ETag)
	return part
}

func NormalizeMultipartCompleteInput(in MultipartCompleteInput) MultipartCompleteInput {
	in.UploadID = NormalizeUploadID(in.UploadID)
	in.Key = NormalizeKey(in.Key)
	in.ExpectedHash = NormalizeHash(in.ExpectedHash)
	if len(in.Parts) > 0 {
		parts := make([]MultipartPart, len(in.Parts))
		for i, part := range in.Parts {
			parts[i] = NormalizeMultipartPart(part)
		}
		in.Parts = parts
	}
	return in
}

func NormalizeMultipartAbortInput(in MultipartAbortInput) MultipartAbortInput {
	in.UploadID = NormalizeUploadID(in.UploadID)
	in.Key = NormalizeKey(in.Key)
	return in
}

func NormalizeUploadID(uploadID string) string {
	return strings.TrimSpace(uploadID)
}
