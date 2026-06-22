package file

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

func TestObjectStoreInterfaceContract(t *testing.T) {
	var store ObjectStore = fakeObjectStore{}
	ctx := context.Background()

	info, err := store.Put(ctx, PutObjectInput{
		Key:  "uploads/a.txt",
		Size: 1,
		MIME: "text/plain",
		Hash: "abcdef123456",
		Body: strings.NewReader("a"),
	})
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if info.Key != "uploads/a.txt" {
		t.Fatalf("info key = %q", info.Key)
	}

	stream, err := store.Get(ctx, "uploads/a.txt")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer stream.Body.Close()
	body, err := io.ReadAll(stream.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(body) != "a" {
		t.Fatalf("body = %q", body)
	}

	if _, err := store.Stat(ctx, "uploads/a.txt"); err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if _, err := store.Presign(ctx, PresignInput{Key: "uploads/a.txt", Operation: PresignOperationGet, ExpiresIn: time.Minute}); err != nil {
		t.Fatalf("Presign() error = %v", err)
	}
	if err := store.Delete(ctx, "uploads/a.txt"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestValidatePutObjectInput(t *testing.T) {
	input := PutObjectInput{
		Key:      " Uploads/A.TXT ",
		Size:     1,
		MIME:     " Text/Plain ",
		Hash:     " ABCDEF123456 ",
		Body:     bytes.NewReader([]byte("a")),
		Metadata: map[string]string{" Trace-ID ": " req-1 "},
	}
	if err := ValidatePutObjectInput(input); err != nil {
		t.Fatalf("ValidatePutObjectInput() error = %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*PutObjectInput)
	}{
		{name: "bad key", mutate: func(in *PutObjectInput) { in.Key = "../secret.txt" }},
		{name: "bad mime", mutate: func(in *PutObjectInput) { in.MIME = "text plain" }},
		{name: "bad hash", mutate: func(in *PutObjectInput) { in.Hash = "short" }},
		{name: "missing body", mutate: func(in *PutObjectInput) { in.Body = nil }},
		{name: "bad metadata", mutate: func(in *PutObjectInput) { in.Metadata = map[string]string{"1bad": "x"} }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			in := input
			tt.mutate(&in)
			if err := ValidatePutObjectInput(in); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestValidatePresignInput(t *testing.T) {
	if err := ValidatePresignInput(PresignInput{
		Key:       "uploads/a.txt",
		Operation: PresignOperationPut,
		ExpiresIn: 15 * time.Minute,
		MIME:      "text/plain",
	}); err != nil {
		t.Fatalf("ValidatePresignInput() error = %v", err)
	}

	cases := []struct {
		name string
		in   PresignInput
	}{
		{name: "bad operation", in: PresignInput{Key: "uploads/a.txt", Operation: "list", ExpiresIn: time.Minute}},
		{name: "zero expiry", in: PresignInput{Key: "uploads/a.txt", Operation: PresignOperationGet}},
		{name: "too long expiry", in: PresignInput{Key: "uploads/a.txt", Operation: PresignOperationGet, ExpiresIn: 25 * time.Hour}},
		{name: "bad mime", in: PresignInput{Key: "uploads/a.txt", Operation: PresignOperationPut, ExpiresIn: time.Minute, MIME: "bad"}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidatePresignInput(tt.in); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestValidatePresignedObject(t *testing.T) {
	err := ValidatePresignedObject(PresignedObject{
		Key:       "uploads/a.txt",
		Operation: PresignOperationGet,
		URL:       "https://storage.example.test/uploads/a.txt?signature=abc",
		ExpiresAt: time.Now().UTC().Add(time.Minute),
		Headers:   map[string]string{"content-type": "text/plain"},
	})
	if err != nil {
		t.Fatalf("ValidatePresignedObject() error = %v", err)
	}
}

type fakeObjectStore struct{}

func (fakeObjectStore) Put(context.Context, PutObjectInput) (ObjectInfo, error) {
	return ObjectInfo{
		Key:          "uploads/a.txt",
		Size:         1,
		MIME:         "text/plain",
		Hash:         "abcdef123456",
		LastModified: time.Now().UTC(),
	}, nil
}

func (fakeObjectStore) Get(context.Context, string) (ObjectStream, error) {
	return ObjectStream{
		Info: ObjectInfo{Key: "uploads/a.txt", Size: 1, MIME: "text/plain", Hash: "abcdef123456", LastModified: time.Now().UTC()},
		Body: io.NopCloser(strings.NewReader("a")),
	}, nil
}

func (fakeObjectStore) Delete(context.Context, string) error {
	return nil
}

func (fakeObjectStore) Stat(context.Context, string) (ObjectInfo, error) {
	return ObjectInfo{Key: "uploads/a.txt", Size: 1, MIME: "text/plain", Hash: "abcdef123456", LastModified: time.Now().UTC()}, nil
}

func (fakeObjectStore) Presign(context.Context, PresignInput) (PresignedObject, error) {
	return PresignedObject{
		Key:       "uploads/a.txt",
		Operation: PresignOperationGet,
		URL:       "https://storage.example.test/uploads/a.txt?signature=abc",
		ExpiresAt: time.Now().UTC().Add(time.Minute),
	}, nil
}
