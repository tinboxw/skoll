package pluginmgr

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

func (s *Service) SetMarketplaceTrustRoots(roots []string) []string {
	normalized := normalizeStringList(roots)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.trustRoots = make(map[string]struct{}, len(normalized))
	for _, root := range normalized {
		s.trustRoots[root] = struct{}{}
	}
	return append([]string(nil), normalized...)
}

func (s *Service) ListMarketplaceTrustRoots() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.trustRoots) == 0 {
		return nil
	}
	out := make([]string, 0, len(s.trustRoots))
	for root := range s.trustRoots {
		out = append(out, root)
	}
	sort.Strings(out)
	return out
}

func (s *Service) IngestMarketplaceIndex(source, signedBy, signature string, expiresAt time.Time, packages []MarketplaceIndexPackage, now time.Time) (MarketplaceIndexIngestResult, error) {
	source = strings.TrimSpace(source)
	signedBy = strings.TrimSpace(signedBy)
	signature = strings.TrimSpace(signature)
	if source == "" {
		return MarketplaceIndexIngestResult{}, errors.New("source is required")
	}
	if signedBy == "" {
		return MarketplaceIndexIngestResult{}, errors.New("signed_by is required")
	}
	if signature == "" {
		return MarketplaceIndexIngestResult{}, errors.New("signature is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.trustRoots[signedBy]; !ok {
		return MarketplaceIndexIngestResult{}, ErrMarketplaceTrustRootNotFound
	}
	if !expiresAt.After(now.UTC()) {
		return MarketplaceIndexIngestResult{}, ErrMarketplaceIndexExpired
	}
	want := signMarketplaceIndex(source, signedBy, expiresAt, packages)
	if signature != want {
		return MarketplaceIndexIngestResult{}, ErrMarketplaceIndexSignatureInvalid
	}

	for i := range packages {
		packages[i].Name = strings.TrimSpace(packages[i].Name)
		packages[i].Version = strings.TrimSpace(packages[i].Version)
		packages[i].PackageURL = strings.TrimSpace(packages[i].PackageURL)
		packages[i].PackageHash = strings.TrimSpace(packages[i].PackageHash)
	}

	s.marketIndex[source] = MarketplaceIndexSource{
		Source:         source,
		SignedBy:       signedBy,
		ExpiresAt:      expiresAt.UTC(),
		PackageCount:   len(packages),
		LastIngestedAt: now.UTC(),
	}

	return MarketplaceIndexIngestResult{
		Accepted:     true,
		Source:       source,
		SignedBy:     signedBy,
		PackageCount: len(packages),
		IndexedAt:    now.UTC(),
		Packages:     cloneMarketplacePackages(packages),
	}, nil
}

func (s *Service) ListMarketplaceIndexSources() []MarketplaceIndexSource {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.marketIndex) == 0 {
		return nil
	}
	out := make([]MarketplaceIndexSource, 0, len(s.marketIndex))
	for _, item := range s.marketIndex {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Source < out[j].Source })
	return out
}

func signMarketplaceIndex(source, signedBy string, expiresAt time.Time, packages []MarketplaceIndexPackage) string {
	b := strings.Builder{}
	b.WriteString(strings.TrimSpace(source))
	b.WriteString("|")
	b.WriteString(strings.TrimSpace(signedBy))
	b.WriteString("|")
	b.WriteString(fmt.Sprintf("%d", expiresAt.UTC().Unix()))
	b.WriteString("|")
	b.WriteString(fmt.Sprintf("%d", len(packages)))
	for _, p := range packages {
		b.WriteString("|")
		b.WriteString(strings.TrimSpace(p.Name))
		b.WriteString("@")
		b.WriteString(strings.TrimSpace(p.Version))
		b.WriteString("#")
		b.WriteString(strings.TrimSpace(p.PackageHash))
	}
	sum := sha256.Sum256([]byte(b.String()))
	return "idxsig:" + hex.EncodeToString(sum[:])
}

func cloneMarketplacePackages(in []MarketplaceIndexPackage) []MarketplaceIndexPackage {
	if len(in) == 0 {
		return nil
	}
	out := make([]MarketplaceIndexPackage, len(in))
	for i := range in {
		out[i] = MarketplaceIndexPackage{
			Name:         in[i].Name,
			Version:      in[i].Version,
			PackageURL:   in[i].PackageURL,
			PackageHash:  in[i].PackageHash,
			Dependencies: cloneDependencies(in[i].Dependencies),
		}
	}
	return out
}

func normalizeStringList(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		set[item] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for item := range set {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}
