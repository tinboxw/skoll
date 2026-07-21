package plugin

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	ErrMarketplaceIndexInvalid     = errors.New("marketplace index is invalid")
	ErrMarketplaceDuplicateRelease = errors.New("marketplace index contains duplicate plugin release")
	ErrMarketplaceVersionConflict  = errors.New("marketplace index contains plugin version conflict")
)

var (
	marketplaceIDPattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{1,127}$`)
	marketplaceSemverPattern = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)
	marketplaceSHA256Pattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
)

type MarketplaceIndex struct {
	SchemaVersion string                     `json:"schema_version"`
	GeneratedAt   string                     `json:"generated_at"`
	Publisher     *MarketplacePublisher      `json:"publisher,omitempty"`
	Plugins       []MarketplacePluginRelease `json:"plugins"`
}

type MarketplacePublisher struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Homepage string `json:"homepage,omitempty"`
	Contact  string `json:"contact,omitempty"`
}

type MarketplacePluginRelease struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Version     string                `json:"version"`
	Manifest    *MarketplaceManifest  `json:"manifest,omitempty"`
	Risk        MarketplaceRiskReport `json:"risk"`
	Signature   MarketplaceSignature  `json:"signature"`
	Source      MarketplaceSource     `json:"source"`
	Changelog   MarketplaceChangelog  `json:"changelog"`
	PublishedAt string                `json:"published_at,omitempty"`
}

type MarketplaceManifest struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type MarketplaceRiskReport struct {
	Level       string                 `json:"level"`
	Summary     string                 `json:"summary"`
	Permissions MarketplaceRiskSection `json:"permissions"`
	Migrations  MarketplaceRiskSection `json:"migrations"`
	Network     MarketplaceRiskSection `json:"network"`
	Assets      MarketplaceRiskSection `json:"assets"`
}

type MarketplaceRiskSection struct {
	Level string   `json:"level"`
	Items []string `json:"items"`
}

type MarketplaceSignature struct {
	Required    bool   `json:"required"`
	Algorithm   string `json:"algorithm"`
	PublicKeyID string `json:"public_key_id"`
	SignedAt    string `json:"signed_at,omitempty"`
	Digest      string `json:"digest"`
	Value       string `json:"value"`
}

type MarketplaceSource struct {
	Type       string `json:"type"`
	URL        string `json:"url"`
	Digest     string `json:"digest"`
	SizeBytes  int64  `json:"size_bytes"`
	Homepage   string `json:"homepage,omitempty"`
	Repository string `json:"repository,omitempty"`
}

type MarketplaceChangelog struct {
	Summary  string   `json:"summary"`
	URL      string   `json:"url,omitempty"`
	Items    []string `json:"items"`
	Breaking bool     `json:"breaking,omitempty"`
}

func ValidateMarketplaceIndex(index MarketplaceIndex) error {
	if strings.TrimSpace(index.SchemaVersion) != "v1" {
		return fmt.Errorf("%w: schema_version must be v1", ErrMarketplaceIndexInvalid)
	}
	if strings.TrimSpace(index.GeneratedAt) == "" {
		return fmt.Errorf("%w: generated_at is required", ErrMarketplaceIndexInvalid)
	}
	if len(index.Plugins) == 0 {
		return fmt.Errorf("%w: plugins is required", ErrMarketplaceIndexInvalid)
	}
	if index.Publisher != nil {
		if err := validateMarketplacePublisher(*index.Publisher); err != nil {
			return err
		}
	}

	seenRaw := map[string]struct{}{}
	seenNormalized := map[string]string{}
	for i, release := range index.Plugins {
		if err := validateMarketplaceRelease(release); err != nil {
			return fmt.Errorf("plugins[%d]: %w", i, err)
		}

		id := strings.TrimSpace(strings.ToLower(release.ID))
		version := strings.TrimSpace(release.Version)
		rawKey := id + "@" + version
		if _, ok := seenRaw[rawKey]; ok {
			return fmt.Errorf("%w: %s", ErrMarketplaceDuplicateRelease, rawKey)
		}
		seenRaw[rawKey] = struct{}{}

		normalizedVersion := normalizeMarketplaceVersion(version)
		normalizedKey := id + "@" + normalizedVersion
		if existing, ok := seenNormalized[normalizedKey]; ok && existing != version {
			return fmt.Errorf("%w: %s conflicts with %s", ErrMarketplaceVersionConflict, version, existing)
		}
		seenNormalized[normalizedKey] = version
	}
	return nil
}

func validateMarketplacePublisher(p MarketplacePublisher) error {
	if !marketplaceIDPattern.MatchString(strings.TrimSpace(p.ID)) {
		return fmt.Errorf("%w: publisher.id is invalid", ErrMarketplaceIndexInvalid)
	}
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("%w: publisher.name is required", ErrMarketplaceIndexInvalid)
	}
	if strings.TrimSpace(p.Homepage) != "" && !isHTTPSMarketplaceURL(p.Homepage) {
		return fmt.Errorf("%w: publisher.homepage must be http or https", ErrMarketplaceIndexInvalid)
	}
	return nil
}

func validateMarketplaceRelease(release MarketplacePluginRelease) error {
	if !marketplaceIDPattern.MatchString(strings.TrimSpace(release.ID)) {
		return fmt.Errorf("%w: id is invalid", ErrMarketplaceIndexInvalid)
	}
	if strings.TrimSpace(release.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrMarketplaceIndexInvalid)
	}
	if !marketplaceSemverPattern.MatchString(strings.TrimSpace(release.Version)) {
		return fmt.Errorf("%w: version is invalid", ErrMarketplaceIndexInvalid)
	}
	if release.Manifest != nil {
		if strings.TrimSpace(release.Manifest.Path) == "" || !marketplaceSHA256Pattern.MatchString(release.Manifest.Digest) {
			return fmt.Errorf("%w: manifest path and digest are required", ErrMarketplaceIndexInvalid)
		}
	}
	if err := validateMarketplaceRisk(release.Risk); err != nil {
		return err
	}
	if err := validateMarketplaceSignature(release.Signature); err != nil {
		return err
	}
	if err := validateMarketplaceSource(release.Source); err != nil {
		return err
	}
	if err := validateMarketplaceChangelog(release.Changelog); err != nil {
		return err
	}
	return nil
}

func normalizeMarketplaceVersion(raw string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(raw)), "v")
}

func validateMarketplaceRisk(risk MarketplaceRiskReport) error {
	if !isMarketplaceRiskLevel(risk.Level) {
		return fmt.Errorf("%w: risk.level is invalid", ErrMarketplaceIndexInvalid)
	}
	if strings.TrimSpace(risk.Summary) == "" {
		return fmt.Errorf("%w: risk.summary is required", ErrMarketplaceIndexInvalid)
	}
	for name, section := range map[string]MarketplaceRiskSection{
		"permissions": risk.Permissions,
		"migrations":  risk.Migrations,
		"network":     risk.Network,
		"assets":      risk.Assets,
	} {
		if !isMarketplaceRiskLevel(section.Level) {
			return fmt.Errorf("%w: risk.%s.level is invalid", ErrMarketplaceIndexInvalid, name)
		}
		for _, item := range section.Items {
			if strings.TrimSpace(item) == "" {
				return fmt.Errorf("%w: risk.%s.items contains blank item", ErrMarketplaceIndexInvalid, name)
			}
		}
	}
	return nil
}

func validateMarketplaceSignature(sig MarketplaceSignature) error {
	if !sig.Required {
		return fmt.Errorf("%w: signature.required must be true", ErrMarketplaceIndexInvalid)
	}
	if sig.Algorithm != string(SigAlgoRSASHA256) {
		return fmt.Errorf("%w: signature.algorithm is unsupported", ErrMarketplaceIndexInvalid)
	}
	if strings.TrimSpace(sig.PublicKeyID) == "" {
		return fmt.Errorf("%w: signature.public_key_id is required", ErrMarketplaceIndexInvalid)
	}
	if !marketplaceSHA256Pattern.MatchString(sig.Digest) {
		return fmt.Errorf("%w: signature.digest is invalid", ErrMarketplaceIndexInvalid)
	}
	if strings.TrimSpace(sig.Value) == "" {
		return fmt.Errorf("%w: signature.value is required", ErrMarketplaceIndexInvalid)
	}
	return nil
}

func validateMarketplaceSource(source MarketplaceSource) error {
	switch strings.TrimSpace(source.Type) {
	case "zip", "oci", "git_release":
	default:
		return fmt.Errorf("%w: source.type is invalid", ErrMarketplaceIndexInvalid)
	}
	if !isHTTPSMarketplaceURL(source.URL) {
		return fmt.Errorf("%w: source.url must be http or https", ErrMarketplaceIndexInvalid)
	}
	if !marketplaceSHA256Pattern.MatchString(source.Digest) {
		return fmt.Errorf("%w: source.digest is invalid", ErrMarketplaceIndexInvalid)
	}
	if source.SizeBytes <= 0 {
		return fmt.Errorf("%w: source.size_bytes must be positive", ErrMarketplaceIndexInvalid)
	}
	if strings.TrimSpace(source.Homepage) != "" && !isHTTPSMarketplaceURL(source.Homepage) {
		return fmt.Errorf("%w: source.homepage must be http or https", ErrMarketplaceIndexInvalid)
	}
	if strings.TrimSpace(source.Repository) != "" && !isHTTPSMarketplaceURL(source.Repository) {
		return fmt.Errorf("%w: source.repository must be http or https", ErrMarketplaceIndexInvalid)
	}
	return nil
}

func validateMarketplaceChangelog(changelog MarketplaceChangelog) error {
	if strings.TrimSpace(changelog.Summary) == "" {
		return fmt.Errorf("%w: changelog.summary is required", ErrMarketplaceIndexInvalid)
	}
	if len(changelog.Items) == 0 {
		return fmt.Errorf("%w: changelog.items is required", ErrMarketplaceIndexInvalid)
	}
	for _, item := range changelog.Items {
		if strings.TrimSpace(item) == "" {
			return fmt.Errorf("%w: changelog.items contains blank item", ErrMarketplaceIndexInvalid)
		}
	}
	if strings.TrimSpace(changelog.URL) != "" && !isHTTPSMarketplaceURL(changelog.URL) {
		return fmt.Errorf("%w: changelog.url must be http or https", ErrMarketplaceIndexInvalid)
	}
	return nil
}

func isMarketplaceRiskLevel(level string) bool {
	switch strings.TrimSpace(level) {
	case "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

func isHTTPSMarketplaceURL(raw string) bool {
	u, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}
