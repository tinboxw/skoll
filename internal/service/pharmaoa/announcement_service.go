package pharmaoa

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
)

type AnnouncementService interface {
	Create(ctx context.Context, in AnnouncementCreateInput) (*domainpharma.Announcement, error)
	Publish(ctx context.Context, id string, actorID string) (*domainpharma.Announcement, error)
	List(ctx context.Context, in AnnouncementListInput) ([]*domainpharma.Announcement, error)
	Get(ctx context.Context, id string, in AnnouncementListInput) (*domainpharma.Announcement, error)
	ConfirmRead(ctx context.Context, id, userID string, organizationIDs, roleIDs []string) (*domainpharma.AnnouncementReadConfirmation, error)
	ListReadConfirmations(ctx context.Context, id string) ([]domainpharma.AnnouncementReadConfirmation, error)
}

type AnnouncementCreateInput struct {
	Kind      domainpharma.AnnouncementKind
	Title     string
	Content   string
	Audience  domainpharma.AnnouncementAudience
	Documents []domainpharma.AnnouncementDocument
	ActorID   string
}

type AnnouncementListInput struct {
	ActorID         string
	OrganizationIDs []string
	RoleIDs         []string
	IncludeDraft    bool
}

type announcementService struct {
	mu       sync.RWMutex
	items    map[string]*domainpharma.Announcement
	receipts map[string]map[string]domainpharma.AnnouncementReadConfirmation
	audit    auditsvc.Service
	nowFn    func() time.Time
	nextID   int64
}

func NewAnnouncementService(audit auditsvc.Service) AnnouncementService {
	return &announcementService{items: map[string]*domainpharma.Announcement{}, receipts: map[string]map[string]domainpharma.AnnouncementReadConfirmation{}, audit: audit, nowFn: func() time.Time { return time.Now().UTC() }}
}

func (s *announcementService) Create(ctx context.Context, in AnnouncementCreateInput) (*domainpharma.Announcement, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.nextID++
	item, err := domainpharma.NewAnnouncement(shared.ID("pharma-announcement-"+strconv.FormatInt(s.nextID, 10)), in.Kind, in.Title, in.Content, in.Audience, in.Documents, in.ActorID, s.nowFn())
	if err == nil {
		s.items[item.ID.String()] = cloneAnnouncement(item)
	}
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	s.appendAudit(ctx, in.ActorID, "pharma_oa.announcement.create", item.ID.String(), map[string]any{"kind": item.Kind, "organizations": item.Audience.OrganizationIDs, "roles": item.Audience.RoleIDs})
	return cloneAnnouncement(item), nil
}

func (s *announcementService) Publish(ctx context.Context, id string, actorID string) (*domainpharma.Announcement, error) {
	id = strings.TrimSpace(id)
	s.mu.Lock()
	item := cloneAnnouncement(s.items[id])
	if item == nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("announcement not found")
	}
	if err := item.Publish(actorID, s.nowFn()); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.items[id] = cloneAnnouncement(item)
	s.mu.Unlock()
	s.appendAudit(ctx, actorID, "pharma_oa.announcement.publish", id, map[string]any{"kind": item.Kind})
	return cloneAnnouncement(item), nil
}

func (s *announcementService) List(ctx context.Context, in AnnouncementListInput) ([]*domainpharma.Announcement, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	out := make([]*domainpharma.Announcement, 0, len(s.items))
	for _, item := range s.items {
		if announcementVisible(item, in) {
			out = append(out, cloneAnnouncement(item))
		}
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *announcementService) Get(ctx context.Context, id string, in AnnouncementListInput) (*domainpharma.Announcement, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	item := cloneAnnouncement(s.items[strings.TrimSpace(id)])
	s.mu.RUnlock()
	if item == nil || !announcementVisible(item, in) {
		return nil, fmt.Errorf("announcement not found")
	}
	return item, nil
}

func (s *announcementService) ConfirmRead(ctx context.Context, id, userID string, organizationIDs, roleIDs []string) (*domainpharma.AnnouncementReadConfirmation, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("user id is required")
	}
	item, err := s.Get(ctx, id, AnnouncementListInput{ActorID: userID, OrganizationIDs: organizationIDs, RoleIDs: roleIDs})
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if s.receipts[item.ID.String()] == nil {
		s.receipts[item.ID.String()] = map[string]domainpharma.AnnouncementReadConfirmation{}
	}
	if current, exists := s.receipts[item.ID.String()][userID]; exists {
		s.mu.Unlock()
		return &current, nil
	}
	receipt := domainpharma.AnnouncementReadConfirmation{AnnouncementID: item.ID.String(), UserID: userID, ReadAt: s.nowFn().UTC()}
	s.receipts[item.ID.String()][userID] = receipt
	s.mu.Unlock()
	s.appendAudit(ctx, userID, "pharma_oa.announcement.read", item.ID.String(), nil)
	return &receipt, nil
}

func (s *announcementService) ListReadConfirmations(ctx context.Context, id string) ([]domainpharma.AnnouncementReadConfirmation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	if s.items[strings.TrimSpace(id)] == nil {
		s.mu.RUnlock()
		return nil, fmt.Errorf("announcement not found")
	}
	out := make([]domainpharma.AnnouncementReadConfirmation, 0, len(s.receipts[id]))
	for _, receipt := range s.receipts[id] {
		out = append(out, receipt)
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ReadAt.Before(out[j].ReadAt) })
	return out, nil
}

func announcementVisible(item *domainpharma.Announcement, in AnnouncementListInput) bool {
	if item == nil {
		return false
	}
	if item.Status == domainpharma.AnnouncementDraft {
		return in.IncludeDraft && strings.TrimSpace(in.ActorID) != "" && item.CreatedBy == strings.TrimSpace(in.ActorID)
	}
	if item.Status != domainpharma.AnnouncementPublished {
		return false
	}
	return intersectsAnnouncementAudience(item.Audience.OrganizationIDs, in.OrganizationIDs) || intersectsAnnouncementAudience(item.Audience.RoleIDs, in.RoleIDs)
}

func intersectsAnnouncementAudience(expected, actual []string) bool {
	values := map[string]struct{}{}
	for _, value := range actual {
		values[strings.TrimSpace(value)] = struct{}{}
	}
	for _, value := range expected {
		if _, ok := values[value]; ok {
			return true
		}
	}
	return false
}

func cloneAnnouncement(item *domainpharma.Announcement) *domainpharma.Announcement {
	if item == nil {
		return nil
	}
	out := *item
	out.Audience.OrganizationIDs = append([]string(nil), item.Audience.OrganizationIDs...)
	out.Audience.RoleIDs = append([]string(nil), item.Audience.RoleIDs...)
	out.Documents = append([]domainpharma.AnnouncementDocument(nil), item.Documents...)
	if item.PublishedAt != nil {
		publishedAt := *item.PublishedAt
		out.PublishedAt = &publishedAt
	}
	return &out
}

func (s *announcementService) appendAudit(ctx context.Context, actorID, action, resourceID string, detail map[string]any) {
	if s.audit == nil {
		return
	}
	_, _ = s.audit.Append(ctx, strings.TrimSpace(actorID), action, "pharma_oa_announcement", resourceID, detail)
}
