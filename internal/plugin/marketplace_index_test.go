package plugin

import (
	"errors"
	"testing"
)

func TestValidateMarketplaceIndexValid(t *testing.T) {
	index := validMarketplaceIndex()
	if err := ValidateMarketplaceIndex(index); err != nil {
		t.Fatalf("expected valid marketplace index, got %v", err)
	}
}

func TestValidateMarketplaceIndexInvalid(t *testing.T) {
	tests := []struct {
		name string
		edit func(*MarketplaceIndex)
	}{
		{
			name: "missing schema version",
			edit: func(index *MarketplaceIndex) {
				index.SchemaVersion = ""
			},
		},
		{
			name: "invalid plugin id",
			edit: func(index *MarketplaceIndex) {
				index.Plugins[0].ID = "Demo Plugin"
			},
		},
		{
			name: "missing signature",
			edit: func(index *MarketplaceIndex) {
				index.Plugins[0].Signature.Required = false
			},
		},
		{
			name: "invalid source digest",
			edit: func(index *MarketplaceIndex) {
				index.Plugins[0].Source.Digest = "sha256:not-a-digest"
			},
		},
		{
			name: "blank changelog item",
			edit: func(index *MarketplaceIndex) {
				index.Plugins[0].Changelog.Items = []string{""}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index := validMarketplaceIndex()
			tt.edit(&index)
			if err := ValidateMarketplaceIndex(index); !errors.Is(err, ErrMarketplaceIndexInvalid) {
				t.Fatalf("expected ErrMarketplaceIndexInvalid, got %v", err)
			}
		})
	}
}

func TestValidateMarketplaceIndexDuplicateRelease(t *testing.T) {
	index := validMarketplaceIndex()
	index.Plugins = append(index.Plugins, index.Plugins[0])

	if err := ValidateMarketplaceIndex(index); !errors.Is(err, ErrMarketplaceDuplicateRelease) {
		t.Fatalf("expected ErrMarketplaceDuplicateRelease, got %v", err)
	}
}

func TestValidateMarketplaceIndexVersionConflict(t *testing.T) {
	index := validMarketplaceIndex()
	conflict := index.Plugins[0]
	conflict.Version = "v0.2.0"
	index.Plugins = append(index.Plugins, conflict)

	if err := ValidateMarketplaceIndex(index); !errors.Is(err, ErrMarketplaceVersionConflict) {
		t.Fatalf("expected ErrMarketplaceVersionConflict, got %v", err)
	}
}

func TestValidateMarketplaceIndexAllowsMultipleVersions(t *testing.T) {
	index := validMarketplaceIndex()
	next := index.Plugins[0]
	next.Version = "0.3.0"
	next.Source.URL = "https://marketplace.example.com/plugins/demo-0.3.0.zip"
	index.Plugins = append(index.Plugins, next)

	if err := ValidateMarketplaceIndex(index); err != nil {
		t.Fatalf("expected multiple releases for same plugin id, got %v", err)
	}
}

func TestValidateMarketplaceIndexAllowsDistinctPrereleases(t *testing.T) {
	index := validMarketplaceIndex()
	index.Plugins[0].Version = "0.2.0-alpha.1"
	next := index.Plugins[0]
	next.Version = "0.2.0-beta.1"
	next.Source.URL = "https://marketplace.example.com/plugins/demo-0.2.0-beta.1.zip"
	index.Plugins = append(index.Plugins, next)

	if err := ValidateMarketplaceIndex(index); err != nil {
		t.Fatalf("expected distinct prereleases to remain independently installable, got %v", err)
	}
}

func validMarketplaceIndex() MarketplaceIndex {
	return MarketplaceIndex{
		SchemaVersion: "v1",
		GeneratedAt:   "2026-06-29T00:00:00Z",
		Publisher: &MarketplacePublisher{
			ID:       "skoll",
			Name:     "Skoll",
			Homepage: "https://skoll.example.com",
		},
		Plugins: []MarketplacePluginRelease{
			{
				ID:      "demo",
				Name:    "Demo Separated Plugin",
				Version: "0.2.0",
				Manifest: &MarketplaceManifest{
					Path:   "plugin.yaml",
					Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
				Risk: MarketplaceRiskReport{
					Level:   "medium",
					Summary: "Declares plugin menu and route permissions.",
					Permissions: MarketplaceRiskSection{
						Level: "medium",
						Items: []string{"menu.read", "route.read"},
					},
					Migrations: MarketplaceRiskSection{Level: "low"},
					Network:    MarketplaceRiskSection{Level: "low"},
					Assets: MarketplaceRiskSection{
						Level: "low",
						Items: []string{"frontend dist assets"},
					},
				},
				Signature: MarketplaceSignature{
					Required:    true,
					Algorithm:   "RSA-SHA256",
					PublicKeyID: "skoll-marketplace-2026",
					SignedAt:    "2026-06-29T00:00:00Z",
					Digest:      "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
					Value:       "base64-signature",
				},
				Source: MarketplaceSource{
					Type:      "zip",
					URL:       "https://marketplace.example.com/plugins/demo-0.2.0.zip",
					Digest:    "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
					SizeBytes: 4096,
					Homepage:  "https://marketplace.example.com/plugins/demo",
				},
				Changelog: MarketplaceChangelog{
					Summary: "Adds separated frontend and backend demo features.",
					Items:   []string{"Adds plugin menu entry", "Adds demo route"},
				},
			},
		},
	}
}
