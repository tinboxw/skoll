package file

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var metadataKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_.\-]{0,63}$`)

func ValidatePutObjectInput(in PutObjectInput) error {
	in = NormalizePutObjectInput(in)
	if err := ValidateKey(in.Key); err != nil {
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
	if in.Body == nil {
		return fmt.Errorf("object body is required")
	}
	return ValidateObjectMetadata(in.Metadata)
}

func ValidateObjectInfo(info ObjectInfo) error {
	info = NormalizeObjectInfo(info)
	if err := ValidateKey(info.Key); err != nil {
		return err
	}
	if err := ValidateSize(info.Size); err != nil {
		return err
	}
	if err := ValidateMIME(info.MIME); err != nil {
		return err
	}
	if err := ValidateHash(info.Hash); err != nil {
		return err
	}
	if info.ETag != "" {
		if err := ValidateHash(info.ETag); err != nil {
			return err
		}
	}
	if info.LastModified.IsZero() {
		return fmt.Errorf("object last modified time is required")
	}
	return ValidateObjectMetadata(info.Metadata)
}

func ValidateObjectKey(key string) error {
	return ValidateKey(NormalizeKey(key))
}

func ValidatePresignInput(in PresignInput) error {
	in = NormalizePresignInput(in)
	if err := ValidateKey(in.Key); err != nil {
		return err
	}
	if err := ValidatePresignOperation(in.Operation); err != nil {
		return err
	}
	if in.ExpiresIn <= 0 {
		return fmt.Errorf("presign expiry must be positive")
	}
	if in.ExpiresIn > 24*time.Hour {
		return fmt.Errorf("presign expiry must not exceed 24h")
	}
	if in.MIME != "" {
		return ValidateMIME(in.MIME)
	}
	return nil
}

func ValidatePresignOperation(operation PresignOperation) error {
	switch operation {
	case PresignOperationGet, PresignOperationPut, PresignOperationDelete:
		return nil
	default:
		return fmt.Errorf("presign operation is invalid")
	}
}

func ValidatePresignedObject(object PresignedObject) error {
	if err := ValidateObjectKey(object.Key); err != nil {
		return err
	}
	if err := ValidatePresignOperation(object.Operation); err != nil {
		return err
	}
	if strings.TrimSpace(object.URL) == "" {
		return fmt.Errorf("presigned url is required")
	}
	if _, err := url.ParseRequestURI(object.URL); err != nil {
		return fmt.Errorf("presigned url is invalid: %w", err)
	}
	if object.ExpiresAt.IsZero() {
		return fmt.Errorf("presigned expiry time is required")
	}
	return ValidateObjectMetadata(object.Headers)
}

func NormalizeObjectMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	normalized := make(map[string]string, len(metadata))
	for key, value := range metadata {
		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		normalized[normalizedKey] = strings.TrimSpace(value)
	}
	return normalized
}

func ValidateObjectMetadata(metadata map[string]string) error {
	for key, value := range metadata {
		if !metadataKeyPattern.MatchString(key) {
			return fmt.Errorf("object metadata key must match %s", metadataKeyPattern.String())
		}
		if len([]rune(value)) > 512 {
			return fmt.Errorf("object metadata value is too long")
		}
	}
	return nil
}
