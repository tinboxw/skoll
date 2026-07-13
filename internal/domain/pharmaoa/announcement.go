package pharmaoa

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type AnnouncementKind string
type AnnouncementStatus string

const (
	AnnouncementKindNotice AnnouncementKind = "announcement"
	AnnouncementKindPolicy AnnouncementKind = "policy"

	AnnouncementDraft     AnnouncementStatus = "draft"
	AnnouncementPublished AnnouncementStatus = "published"
)

type AnnouncementAudience struct {
	OrganizationIDs []string `json:"organizationIds"`
	RoleIDs         []string `json:"roleIds"`
}

type AnnouncementDocument struct {
	FileID   string `json:"fileId"`
	FileName string `json:"fileName"`
}

type Announcement struct {
	ID          shared.ID              `json:"id"`
	Kind        AnnouncementKind       `json:"kind"`
	Title       string                 `json:"title"`
	Content     string                 `json:"content"`
	Audience    AnnouncementAudience   `json:"audience"`
	Documents   []AnnouncementDocument `json:"documents"`
	Status      AnnouncementStatus     `json:"status"`
	CreatedBy   string                 `json:"createdBy"`
	CreatedAt   time.Time              `json:"createdAt"`
	PublishedBy string                 `json:"publishedBy,omitempty"`
	PublishedAt *time.Time             `json:"publishedAt,omitempty"`
}

type AnnouncementReadConfirmation struct {
	AnnouncementID string    `json:"announcementId"`
	UserID         string    `json:"userId"`
	ReadAt         time.Time `json:"readAt"`
}

func NewAnnouncement(id shared.ID, kind AnnouncementKind, title, content string, audience AnnouncementAudience, documents []AnnouncementDocument, actorID string, now time.Time) (*Announcement, error) {
	if id.IsZero() || strings.TrimSpace(title) == "" || strings.TrimSpace(content) == "" || strings.TrimSpace(actorID) == "" {
		return nil, fmt.Errorf("announcement input is incomplete")
	}
	if kind != AnnouncementKindNotice && kind != AnnouncementKindPolicy {
		return nil, fmt.Errorf("announcement kind is invalid")
	}
	audience.OrganizationIDs = normalizeAnnouncementIDs(audience.OrganizationIDs)
	audience.RoleIDs = normalizeAnnouncementIDs(audience.RoleIDs)
	if len(audience.OrganizationIDs) == 0 && len(audience.RoleIDs) == 0 {
		return nil, fmt.Errorf("announcement audience is required")
	}
	documents = normalizeAnnouncementDocuments(documents)
	if kind == AnnouncementKindPolicy && len(documents) == 0 {
		return nil, fmt.Errorf("policy document attachment is required")
	}
	return &Announcement{ID: id, Kind: kind, Title: strings.TrimSpace(title), Content: strings.TrimSpace(content), Audience: audience, Documents: documents, Status: AnnouncementDraft, CreatedBy: strings.TrimSpace(actorID), CreatedAt: now.UTC()}, nil
}

func (a *Announcement) Publish(actorID string, now time.Time) error {
	if a == nil || strings.TrimSpace(actorID) == "" {
		return fmt.Errorf("publisher is required")
	}
	if a.Status == AnnouncementPublished {
		return nil
	}
	if a.Status != AnnouncementDraft {
		return fmt.Errorf("announcement cannot be published")
	}
	publishedAt := now.UTC()
	a.Status = AnnouncementPublished
	a.PublishedBy = strings.TrimSpace(actorID)
	a.PublishedAt = &publishedAt
	return nil
}

func normalizeAnnouncementIDs(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func normalizeAnnouncementDocuments(values []AnnouncementDocument) []AnnouncementDocument {
	out := make([]AnnouncementDocument, 0, len(values))
	for _, value := range values {
		value.FileID = strings.TrimSpace(value.FileID)
		value.FileName = strings.TrimSpace(value.FileName)
		if value.FileID != "" && value.FileName != "" {
			out = append(out, value)
		}
	}
	return out
}
