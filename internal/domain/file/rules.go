package file

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	keyPattern        = regexp.MustCompile(`^[a-z0-9][a-z0-9/_\-.]{1,255}$`)
	identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9_.\-]{0,63}$`)
	mimePattern       = regexp.MustCompile(`^[a-z0-9][a-z0-9!#$&^_.+\-]{0,126}/[a-z0-9][a-z0-9!#$&^_.+\-]{0,126}$`)
)

const MaxFileSizeBytes int64 = 100 * 1024 * 1024

var ErrFileTooLarge = errors.New("file size exceeds limit")

func ValidateFileObjectInput(in FileObjectInput) error {
	if in.ID.IsZero() {
		return fmt.Errorf("file object id is required")
	}
	if err := ValidateKey(in.Key); err != nil {
		return err
	}
	if err := ValidateName(in.Name); err != nil {
		return err
	}
	if err := ValidateSize(in.Size); err != nil {
		return err
	}
	if err := ValidateMIME(in.MIME); err != nil {
		return err
	}
	if err := ValidateHash(in.Hash); err != nil {
		return err
	}
	if err := ValidateOwner(in.Owner); err != nil {
		return err
	}
	if err := ValidateVisibility(in.Visibility); err != nil {
		return err
	}
	if err := ValidateIdentifier(in.StorageDriver, "storage driver"); err != nil {
		return err
	}
	if err := ValidateStatus(in.Status); err != nil {
		return err
	}
	if err := ValidateSource(in.Source); err != nil {
		return err
	}
	if err := ValidateObjectMetadata(in.Metadata); err != nil {
		return err
	}
	return ValidateTimestamps(in.CreatedAt, in.UpdatedAt)
}

func NormalizeKey(key string) string {
	key = strings.TrimSpace(strings.ReplaceAll(key, "\\", "/"))
	key = strings.Trim(key, "/")
	for strings.Contains(key, "//") {
		key = strings.ReplaceAll(key, "//", "/")
	}
	return strings.ToLower(key)
}

func ValidateKey(key string) error {
	if !keyPattern.MatchString(key) {
		return fmt.Errorf("file key must match %s", keyPattern.String())
	}
	if strings.Contains(key, "..") || strings.Contains(key, "/./") || strings.HasPrefix(key, "./") {
		return fmt.Errorf("file key must not contain relative path segments")
	}
	if strings.Contains(key, "//") || strings.HasPrefix(key, "/") || strings.HasSuffix(key, "/") {
		return fmt.Errorf("file key must not contain empty path segments")
	}
	return nil
}

func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("file name is required")
	}
	if len([]rune(name)) > 255 {
		return fmt.Errorf("file name is too long")
	}
	if strings.ContainsAny(name, "\r\n") {
		return fmt.Errorf("file name must not contain line breaks")
	}
	return nil
}

func ValidateSize(size int64) error {
	if size < 0 {
		return fmt.Errorf("file size must be non-negative")
	}
	if size > MaxFileSizeBytes {
		return ErrFileTooLarge
	}
	return nil
}

func NormalizeMIME(mime string) string {
	return strings.ToLower(strings.TrimSpace(mime))
}

func ValidateMIME(mime string) error {
	if !mimePattern.MatchString(mime) {
		return fmt.Errorf("file mime type is invalid")
	}
	return nil
}

func NormalizeHash(hash string) string {
	return strings.ToLower(strings.TrimSpace(hash))
}

func ValidateHash(hash string) error {
	if len(hash) < 8 || len(hash) > 256 {
		return fmt.Errorf("file hash length is invalid")
	}
	if strings.ContainsAny(hash, " \t\r\n") {
		return fmt.Errorf("file hash must not contain whitespace")
	}
	return nil
}

func NormalizeOwner(owner OwnerRef) OwnerRef {
	return OwnerRef{
		Type: NormalizeIdentifier(owner.Type),
		ID:   owner.ID,
	}
}

func ValidateOwner(owner OwnerRef) error {
	if err := ValidateIdentifier(owner.Type, "owner type"); err != nil {
		return err
	}
	if owner.ID.IsZero() {
		return fmt.Errorf("file owner id is required")
	}
	return nil
}

func ValidateVisibility(visibility Visibility) error {
	switch visibility {
	case VisibilityPrivate, VisibilityPublic, VisibilityPluginAsset:
		return nil
	default:
		return fmt.Errorf("file visibility is invalid")
	}
}

func NormalizeIdentifier(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func ValidateIdentifier(value, label string) error {
	if !identifierPattern.MatchString(value) {
		return fmt.Errorf("file %s must match %s", label, identifierPattern.String())
	}
	return nil
}

func ValidateStatus(status Status) error {
	switch status {
	case StatusPending, StatusAvailable, StatusFailed, StatusDeleted:
		return nil
	default:
		return fmt.Errorf("file status is invalid")
	}
}

func NormalizeSource(source SourceRef) SourceRef {
	return SourceRef{
		Module:   NormalizeIdentifier(source.Module),
		PluginID: NormalizeIdentifier(source.PluginID),
	}
}

func ValidateSource(source SourceRef) error {
	if err := ValidateIdentifier(source.Module, "source module"); err != nil {
		return err
	}
	if source.PluginID == "" {
		return nil
	}
	return ValidateIdentifier(source.PluginID, "source plugin id")
}

func ValidateTimestamps(createdAt, updatedAt time.Time) error {
	if createdAt.IsZero() {
		return fmt.Errorf("file object created time is required")
	}
	if updatedAt.IsZero() {
		return fmt.Errorf("file object updated time is required")
	}
	if updatedAt.Before(createdAt) {
		return fmt.Errorf("file object updated time must not be before created time")
	}
	return nil
}
