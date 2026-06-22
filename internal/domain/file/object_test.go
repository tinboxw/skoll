package file

import (
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestNewFileObjectNormalizesAndDefaults(t *testing.T) {
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)

	obj, err := NewFileObject(FileObjectInput{
		ID:            shared.ID("file-1"),
		Key:           " /Uploads/2026/Invoice.PDF ",
		Name:          " invoice.pdf ",
		Size:          12,
		MIME:          " Application/PDF ",
		Hash:          " ABCDEF123456 ",
		Owner:         OwnerRef{Type: " User ", ID: shared.ID("u-1")},
		StorageDriver: " Local ",
		Source:        SourceRef{Module: " System "},
		CreatedAt:     now,
	})
	if err != nil {
		t.Fatalf("NewFileObject() error = %v", err)
	}

	if obj.Key != "uploads/2026/invoice.pdf" {
		t.Fatalf("key = %q", obj.Key)
	}
	if obj.Name != "invoice.pdf" || obj.MIME != "application/pdf" || obj.Hash != "abcdef123456" {
		t.Fatalf("normalized fields = %#v", obj)
	}
	if obj.Visibility != VisibilityPrivate {
		t.Fatalf("visibility = %q", obj.Visibility)
	}
	if obj.Status != StatusPending {
		t.Fatalf("status = %q", obj.Status)
	}
	if obj.Owner.Type != "user" || obj.Owner.ID != "u-1" {
		t.Fatalf("owner = %#v", obj.Owner)
	}
	if obj.StorageDriver != "local" || obj.Source.Module != "system" {
		t.Fatalf("storage/source = %q/%#v", obj.StorageDriver, obj.Source)
	}
	if !obj.Meta.CreatedAt.Equal(now) || !obj.Meta.UpdatedAt.Equal(now) {
		t.Fatalf("meta = %#v", obj.Meta)
	}
}

func TestNewFileObjectRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*FileObjectInput)
	}{
		{name: "missing id", mutate: func(in *FileObjectInput) { in.ID = "" }},
		{name: "relative key", mutate: func(in *FileObjectInput) { in.Key = "uploads/../secret.txt" }},
		{name: "missing name", mutate: func(in *FileObjectInput) { in.Name = " " }},
		{name: "negative size", mutate: func(in *FileObjectInput) { in.Size = -1 }},
		{name: "invalid mime", mutate: func(in *FileObjectInput) { in.MIME = "text plain" }},
		{name: "missing hash", mutate: func(in *FileObjectInput) { in.Hash = " " }},
		{name: "missing owner", mutate: func(in *FileObjectInput) { in.Owner.ID = "" }},
		{name: "invalid visibility", mutate: func(in *FileObjectInput) { in.Visibility = Visibility("shared") }},
		{name: "invalid storage", mutate: func(in *FileObjectInput) { in.StorageDriver = "1local" }},
		{name: "invalid status", mutate: func(in *FileObjectInput) { in.Status = Status("ready") }},
		{name: "missing source module", mutate: func(in *FileObjectInput) { in.Source.Module = " " }},
		{name: "updated before created", mutate: func(in *FileObjectInput) { in.UpdatedAt = in.CreatedAt.Add(-time.Second) }},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewFileObject(validFileObjectInput(tt.mutate))
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestFileObjectStatusTransitions(t *testing.T) {
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	obj, err := NewFileObject(validFileObjectInput(nil))
	if err != nil {
		t.Fatalf("NewFileObject() error = %v", err)
	}

	later := now.Add(time.Hour)
	if err := obj.MarkAvailable("0123456789abcdef", later); err != nil {
		t.Fatalf("MarkAvailable() error = %v", err)
	}
	if obj.Status != StatusAvailable || obj.Hash != "0123456789abcdef" || !obj.Meta.UpdatedAt.Equal(later) {
		t.Fatalf("available object = %#v", obj)
	}

	obj.MarkFailed(later.Add(time.Hour))
	if obj.Status != StatusFailed {
		t.Fatalf("status = %q", obj.Status)
	}

	obj.MarkDeleted(later.Add(2 * time.Hour))
	if obj.Status != StatusDeleted {
		t.Fatalf("status = %q", obj.Status)
	}
}

func TestVisibilityValidate(t *testing.T) {
	for _, visibility := range []Visibility{VisibilityPrivate, VisibilityPublic, VisibilityPluginAsset} {
		if err := ValidateVisibility(visibility); err != nil {
			t.Fatalf("ValidateVisibility(%q) error = %v", visibility, err)
		}
	}
	if err := ValidateVisibility(Visibility("internal")); err == nil {
		t.Fatal("expected invalid visibility error")
	}
}

func validFileObjectInput(mutate func(*FileObjectInput)) FileObjectInput {
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	in := FileObjectInput{
		ID:            shared.ID("file-1"),
		Key:           "uploads/2026/invoice.pdf",
		Name:          "invoice.pdf",
		Size:          128,
		MIME:          "application/pdf",
		Hash:          "abcdef123456",
		Owner:         OwnerRef{Type: "user", ID: shared.ID("u-1")},
		Visibility:    VisibilityPrivate,
		StorageDriver: "local",
		Status:        StatusPending,
		Source:        SourceRef{Module: "system"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if mutate != nil {
		mutate(&in)
	}
	return in
}
