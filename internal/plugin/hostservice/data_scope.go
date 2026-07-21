package hostservice

import (
	"context"
	"fmt"
	"sort"
	"strings"

	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	organizationrepo "github.com/tinboxw/skoll/internal/repository/organization"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"github.com/tinboxw/skoll/pkg/security"
)

type trustedScopeResolver interface {
	ResolveDataScope(ctx context.Context, in rbacsvc.ResolveDataScopeInput) (rbacsvc.DataScopeDecision, error)
}

type dataScopeService struct {
	resolver      trustedScopeResolver
	organizations organizationrepo.OrganizationRepository
}

func NewDataScopeService(resolver trustedScopeResolver, organizations organizationrepo.OrganizationRepository) (pluginsdk.DataScopeService, error) {
	if resolver == nil {
		return nil, fmt.Errorf("plugin host data-scope resolver is required")
	}
	if organizations == nil {
		return nil, fmt.Errorf("plugin host organization repository is required")
	}
	return &dataScopeService{resolver: resolver, organizations: organizations}, nil
}

func (s *dataScopeService) Resolve(ctx context.Context, permission pluginsdk.Permission) (pluginsdk.ScopePredicate, error) {
	claims, ok := security.JWTClaimsFromContext(ctx)
	if !ok || strings.TrimSpace(claims.Subject) == "" {
		return pluginsdk.ScopePredicate{}, fmt.Errorf("trusted plugin identity is required")
	}
	resource := strings.TrimSpace(permission.Resource)
	action := strings.TrimSpace(permission.Action)
	if resource == "" || action == "" {
		return pluginsdk.ScopePredicate{}, fmt.Errorf("plugin scope resource and action are required")
	}
	decision, err := s.resolver.ResolveDataScope(ctx, rbacsvc.ResolveDataScopeInput{Resource: resource, Action: action})
	if err != nil {
		return pluginsdk.ScopePredicate{}, err
	}
	trusted, err := s.trustedScope(ctx, claims, decision)
	if err != nil {
		return pluginsdk.ScopePredicate{}, err
	}
	return pluginsdk.NewScopePredicate(trusted)
}

func (s *dataScopeService) trustedScope(ctx context.Context, claims security.JWTClaims, decision rbacsvc.DataScopeDecision) (pluginsdk.TrustedScope, error) {
	subjectID := strings.TrimSpace(claims.Subject)
	if claims.HasRole("super_admin") {
		return pluginsdk.TrustedScope{SubjectID: subjectID, AllTenants: true, AllOwners: true, AllOrganizations: true}, nil
	}

	tenantID := trustedTenantID(claims)
	if tenantID == "" {
		return pluginsdk.TrustedScope{}, fmt.Errorf("trusted tenant is required for plugin data scope")
	}
	trusted := pluginsdk.TrustedScope{SubjectID: subjectID, TenantIDs: []string{tenantID}}
	switch domainrbac.NormalizeDataScope(decision.Scope) {
	case domainrbac.DataScopeSelf:
		trusted.OwnerIDs = []string{subjectID}
		trusted.AllOrganizations = true
	case domainrbac.DataScopeDepartment, domainrbac.DataScopeDepartmentTree:
		trusted.AllOwners = true
		trusted.OrganizationIDs = compactIDs(decision.DepartmentIDs)
	case domainrbac.DataScopeAll:
		organizationIDs, err := s.organizationTreeIDs(ctx, tenantID)
		if err != nil {
			return pluginsdk.TrustedScope{}, err
		}
		trusted.AllOwners = true
		trusted.OrganizationIDs = organizationIDs
	default:
		return pluginsdk.TrustedScope{}, fmt.Errorf("unsupported trusted plugin data scope %q", decision.Scope)
	}
	return trusted, nil
}

func (s *dataScopeService) organizationTreeIDs(ctx context.Context, rootID string) ([]string, error) {
	items, err := s.organizations.ListDepartments(ctx, 0, 0)
	if err != nil {
		return nil, err
	}
	included := map[string]struct{}{rootID: {}}
	rootFound := false
	for _, item := range items {
		if item.ID.String() == rootID {
			rootFound = true
			break
		}
	}
	if !rootFound {
		return nil, fmt.Errorf("trusted tenant organization no longer exists")
	}
	for changed := true; changed; {
		changed = false
		for _, item := range items {
			id := strings.TrimSpace(item.ID.String())
			parentID := strings.TrimSpace(item.ParentID.String())
			if id == "" {
				continue
			}
			if _, ok := included[parentID]; !ok {
				continue
			}
			if _, exists := included[id]; exists {
				continue
			}
			included[id] = struct{}{}
			changed = true
		}
	}
	out := make([]string, 0, len(included))
	for id := range included {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

func trustedTenantID(claims security.JWTClaims) string {
	for _, id := range claims.OrganizationPath {
		if id = strings.TrimSpace(id); id != "" {
			return id
		}
	}
	return strings.TrimSpace(claims.OrganizationID)
}

func compactIDs(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
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
	sort.Strings(out)
	return out
}

var _ pluginsdk.DataScopeService = (*dataScopeService)(nil)
