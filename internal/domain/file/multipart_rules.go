package file

import (
	"fmt"
	"regexp"
	"time"
)

const (
	minMultipartPartNumber = 1
	maxMultipartPartNumber = 10000
)

var uploadIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:\-]{0,127}$`)

func ValidateMultipartInitInput(in MultipartInitInput) error {
	in = NormalizeMultipartInitInput(in)
	if err := ValidateKey(in.Key); err != nil {
		return err
	}
	if err := ValidateSize(in.Size); err != nil {
		return err
	}
	if err := ValidateMIME(in.MIME); err != nil {
		return err
	}
	if err := ValidateHash(in.ExpectedHash); err != nil {
		return err
	}
	return ValidateObjectMetadata(in.Metadata)
}

func ValidateMultipartUpload(upload MultipartUpload) error {
	upload = NormalizeMultipartUpload(upload)
	if err := ValidateUploadID(upload.UploadID); err != nil {
		return err
	}
	if err := ValidateKey(upload.Key); err != nil {
		return err
	}
	if upload.ExpiresAt.IsZero() {
		return fmt.Errorf("multipart upload expiry time is required")
	}
	return ValidateObjectMetadata(upload.Metadata)
}

func ValidateMultipartUploadPartInput(in MultipartUploadPartInput) error {
	in = NormalizeMultipartUploadPartInput(in)
	if err := ValidateUploadID(in.UploadID); err != nil {
		return err
	}
	if err := ValidateKey(in.Key); err != nil {
		return err
	}
	if err := ValidatePartNumber(in.PartNumber); err != nil {
		return err
	}
	if in.Size <= 0 {
		return fmt.Errorf("multipart part size must be positive")
	}
	if err := ValidateHash(in.Hash); err != nil {
		return err
	}
	if in.Body == nil {
		return fmt.Errorf("multipart part body is required")
	}
	return nil
}

func ValidateMultipartPart(part MultipartPart) error {
	part = NormalizeMultipartPart(part)
	if err := ValidatePartNumber(part.PartNumber); err != nil {
		return err
	}
	if part.Size <= 0 {
		return fmt.Errorf("multipart part size must be positive")
	}
	if err := ValidateHash(part.Hash); err != nil {
		return err
	}
	if part.ETag != "" {
		if err := ValidateHash(part.ETag); err != nil {
			return err
		}
	}
	return nil
}

func ValidateMultipartCompleteInput(in MultipartCompleteInput) error {
	in = NormalizeMultipartCompleteInput(in)
	if err := ValidateUploadID(in.UploadID); err != nil {
		return err
	}
	if err := ValidateKey(in.Key); err != nil {
		return err
	}
	if in.ExpectedSize <= 0 {
		return fmt.Errorf("multipart expected size must be positive")
	}
	if err := ValidateHash(in.ExpectedHash); err != nil {
		return err
	}
	if len(in.Parts) == 0 {
		return fmt.Errorf("multipart complete requires at least one part")
	}
	seen := make(map[int]struct{}, len(in.Parts))
	var total int64
	for _, part := range in.Parts {
		if err := ValidateMultipartPart(part); err != nil {
			return err
		}
		if _, ok := seen[part.PartNumber]; ok {
			return fmt.Errorf("multipart part number is duplicated")
		}
		seen[part.PartNumber] = struct{}{}
		total += part.Size
	}
	if total != in.ExpectedSize {
		return fmt.Errorf("multipart part size total does not match expected size")
	}
	return nil
}

func ValidateMultipartAbortInput(in MultipartAbortInput) error {
	in = NormalizeMultipartAbortInput(in)
	if err := ValidateUploadID(in.UploadID); err != nil {
		return err
	}
	return ValidateKey(in.Key)
}

func ValidateUploadID(uploadID string) error {
	if !uploadIDPattern.MatchString(uploadID) {
		return fmt.Errorf("multipart upload id must match %s", uploadIDPattern.String())
	}
	return nil
}

func ValidatePartNumber(partNumber int) error {
	if partNumber < minMultipartPartNumber || partNumber > maxMultipartPartNumber {
		return fmt.Errorf("multipart part number must be between %d and %d", minMultipartPartNumber, maxMultipartPartNumber)
	}
	return nil
}

func MultipartExpiry(now time.Time, ttl time.Duration) (time.Time, error) {
	if now.IsZero() {
		return time.Time{}, fmt.Errorf("multipart expiry base time is required")
	}
	if ttl <= 0 {
		return time.Time{}, fmt.Errorf("multipart expiry ttl must be positive")
	}
	return now.Add(ttl), nil
}
