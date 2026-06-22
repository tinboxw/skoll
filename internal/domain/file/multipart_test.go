package file

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

func TestMultipartStoreInterfaceContract(t *testing.T) {
	var store MultipartStore = fakeMultipartStore{}
	ctx := context.Background()

	upload, err := store.Init(ctx, MultipartInitInput{
		Key:          "uploads/large.bin",
		Size:         6,
		MIME:         "application/octet-stream",
		ExpectedHash: "abcdef123456",
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if upload.UploadID == "" {
		t.Fatal("upload id is required")
	}

	part, err := store.UploadPart(ctx, MultipartUploadPartInput{
		UploadID:   upload.UploadID,
		Key:        upload.Key,
		PartNumber: 1,
		Size:       6,
		Hash:       "abcdef123456",
		Body:       strings.NewReader("abcdef"),
	})
	if err != nil {
		t.Fatalf("UploadPart() error = %v", err)
	}

	if _, err := store.Complete(ctx, MultipartCompleteInput{
		UploadID:     upload.UploadID,
		Key:          upload.Key,
		ExpectedSize: 6,
		ExpectedHash: "abcdef123456",
		Parts:        []MultipartPart{part},
	}); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if err := store.Abort(ctx, MultipartAbortInput{UploadID: upload.UploadID, Key: upload.Key}); err != nil {
		t.Fatalf("Abort() error = %v", err)
	}
}

func TestValidateMultipartInitInput(t *testing.T) {
	input := MultipartInitInput{
		Key:          " Uploads/Large.BIN ",
		Size:         6,
		MIME:         " Application/Octet-Stream ",
		ExpectedHash: " ABCDEF123456 ",
		Metadata:     map[string]string{" Trace-ID ": " req-1 "},
	}
	if err := ValidateMultipartInitInput(input); err != nil {
		t.Fatalf("ValidateMultipartInitInput() error = %v", err)
	}

	input.ExpectedHash = "short"
	if err := ValidateMultipartInitInput(input); err == nil {
		t.Fatal("expected invalid hash error")
	}
}

func TestValidateMultipartUploadPartInput(t *testing.T) {
	input := MultipartUploadPartInput{
		UploadID:   "upload-1",
		Key:        "uploads/large.bin",
		PartNumber: 1,
		Size:       6,
		Hash:       "abcdef123456",
		Body:       strings.NewReader("abcdef"),
	}
	if err := ValidateMultipartUploadPartInput(input); err != nil {
		t.Fatalf("ValidateMultipartUploadPartInput() error = %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*MultipartUploadPartInput)
	}{
		{name: "missing upload id", mutate: func(in *MultipartUploadPartInput) { in.UploadID = " " }},
		{name: "bad key", mutate: func(in *MultipartUploadPartInput) { in.Key = "uploads/../secret.bin" }},
		{name: "bad part", mutate: func(in *MultipartUploadPartInput) { in.PartNumber = 0 }},
		{name: "bad size", mutate: func(in *MultipartUploadPartInput) { in.Size = 0 }},
		{name: "bad hash", mutate: func(in *MultipartUploadPartInput) { in.Hash = "short" }},
		{name: "missing body", mutate: func(in *MultipartUploadPartInput) { in.Body = nil }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			in := input
			tt.mutate(&in)
			if err := ValidateMultipartUploadPartInput(in); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestValidateMultipartCompleteInputRequiresHashAndSizeContext(t *testing.T) {
	input := MultipartCompleteInput{
		UploadID:     "upload-1",
		Key:          "uploads/large.bin",
		ExpectedSize: 12,
		ExpectedHash: "abcdef123456",
		Parts: []MultipartPart{
			{PartNumber: 1, Size: 6, Hash: "11111111", ETag: "etag1111"},
			{PartNumber: 2, Size: 6, Hash: "22222222", ETag: "etag2222"},
		},
	}
	if err := ValidateMultipartCompleteInput(input); err != nil {
		t.Fatalf("ValidateMultipartCompleteInput() error = %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*MultipartCompleteInput)
	}{
		{name: "missing expected hash", mutate: func(in *MultipartCompleteInput) { in.ExpectedHash = " " }},
		{name: "no parts", mutate: func(in *MultipartCompleteInput) { in.Parts = nil }},
		{name: "duplicate part", mutate: func(in *MultipartCompleteInput) { in.Parts[1].PartNumber = 1 }},
		{name: "size mismatch", mutate: func(in *MultipartCompleteInput) { in.ExpectedSize = 13 }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			in := input
			in.Parts = append([]MultipartPart(nil), input.Parts...)
			tt.mutate(&in)
			if err := ValidateMultipartCompleteInput(in); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestMultipartExpiry(t *testing.T) {
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	got, err := MultipartExpiry(now, time.Hour)
	if err != nil {
		t.Fatalf("MultipartExpiry() error = %v", err)
	}
	if !got.Equal(now.Add(time.Hour)) {
		t.Fatalf("expiry = %s", got)
	}
}

type fakeMultipartStore struct{}

func (fakeMultipartStore) Init(context.Context, MultipartInitInput) (MultipartUpload, error) {
	return MultipartUpload{
		UploadID:  "upload-1",
		Key:       "uploads/large.bin",
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}, nil
}

func (fakeMultipartStore) UploadPart(context.Context, MultipartUploadPartInput) (MultipartPart, error) {
	return MultipartPart{PartNumber: 1, Size: 6, Hash: "abcdef123456", ETag: "etag123456"}, nil
}

func (fakeMultipartStore) Complete(context.Context, MultipartCompleteInput) (ObjectInfo, error) {
	return ObjectInfo{
		Key:          "uploads/large.bin",
		Size:         6,
		MIME:         "application/octet-stream",
		Hash:         "abcdef123456",
		LastModified: time.Now().UTC(),
	}, nil
}

func (fakeMultipartStore) Abort(context.Context, MultipartAbortInput) error {
	return nil
}

var _ io.Reader = strings.NewReader("")
