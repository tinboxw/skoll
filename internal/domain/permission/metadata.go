package permission

import (
	"fmt"
	"regexp"
	"strings"
)

type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

var metadataKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_.\-]{0,63}$`)

func NormalizeMetadata(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return map[string]string{}
	}
	metadata := make(map[string]string, len(raw))
	for key, value := range raw {
		normalizedKey := strings.TrimSpace(strings.ToLower(key))
		if normalizedKey == "" {
			continue
		}
		metadata[normalizedKey] = strings.TrimSpace(value)
	}
	return metadata
}

func ValidateRisk(risk RiskLevel) error {
	switch risk {
	case RiskLevelLow, RiskLevelMedium, RiskLevelHigh, RiskLevelCritical:
		return nil
	default:
		return fmt.Errorf("permission risk is invalid")
	}
}

func ValidateMetadata(metadata map[string]string) error {
	for key, value := range metadata {
		if !metadataKeyPattern.MatchString(key) {
			return fmt.Errorf("permission metadata key must match %s", metadataKeyPattern.String())
		}
		if len([]rune(value)) > 512 {
			return fmt.Errorf("permission metadata value is too long")
		}
	}
	return nil
}
