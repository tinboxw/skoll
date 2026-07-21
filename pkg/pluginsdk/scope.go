package pluginsdk

import (
	"fmt"
	"strings"
)

type TrustedScope struct {
	SubjectID        string
	TenantIDs        []string
	OwnerIDs         []string
	OrganizationIDs  []string
	AllTenants       bool
	AllOwners        bool
	AllOrganizations bool
}

type ScopeFilter struct {
	TenantIDs       []string
	OwnerIDs        []string
	OrganizationIDs []string
}

type ScopedRecord struct {
	TenantID       string
	OwnerID        string
	OrganizationID string
}

type ScopePredicate struct {
	subjectID        string
	tenantIDs        []string
	ownerIDs         []string
	organizationIDs  []string
	allTenants       bool
	allOwners        bool
	allOrganizations bool
	denied           bool
}

func NewScopePredicate(scope TrustedScope) (ScopePredicate, error) {
	predicate := ScopePredicate{
		subjectID: strings.TrimSpace(scope.SubjectID), tenantIDs: cleanScopeIDs(scope.TenantIDs),
		ownerIDs: cleanScopeIDs(scope.OwnerIDs), organizationIDs: cleanScopeIDs(scope.OrganizationIDs),
		allTenants: scope.AllTenants, allOwners: scope.AllOwners, allOrganizations: scope.AllOrganizations,
	}
	if predicate.subjectID == "" {
		return ScopePredicate{}, fmt.Errorf("trusted scope subject is required")
	}
	if !predicate.allTenants && len(predicate.tenantIDs) == 0 {
		return ScopePredicate{}, fmt.Errorf("trusted tenant scope is empty")
	}
	if !predicate.allOwners && len(predicate.ownerIDs) == 0 {
		return ScopePredicate{}, fmt.Errorf("trusted owner scope is empty")
	}
	if !predicate.allOrganizations && len(predicate.organizationIDs) == 0 {
		return ScopePredicate{}, fmt.Errorf("trusted organization scope is empty")
	}
	return predicate, nil
}

func (p ScopePredicate) SubjectID() string   { return p.subjectID }
func (p ScopePredicate) TenantIDs() []string { return append([]string(nil), p.tenantIDs...) }
func (p ScopePredicate) OwnerIDs() []string  { return append([]string(nil), p.ownerIDs...) }
func (p ScopePredicate) OrganizationIDs() []string {
	return append([]string(nil), p.organizationIDs...)
}
func (p ScopePredicate) AllTenants() bool       { return p.allTenants }
func (p ScopePredicate) AllOwners() bool        { return p.allOwners }
func (p ScopePredicate) AllOrganizations() bool { return p.allOrganizations }
func (p ScopePredicate) Denied() bool           { return p.denied }

func (p ScopePredicate) Constrain(requested ScopeFilter) ScopePredicate {
	if p.denied {
		return p
	}
	p.tenantIDs, p.allTenants, p.denied = constrainScopeDimension(p.tenantIDs, p.allTenants, requested.TenantIDs)
	if p.denied {
		return p
	}
	p.ownerIDs, p.allOwners, p.denied = constrainScopeDimension(p.ownerIDs, p.allOwners, requested.OwnerIDs)
	if p.denied {
		return p
	}
	p.organizationIDs, p.allOrganizations, p.denied = constrainScopeDimension(p.organizationIDs, p.allOrganizations, requested.OrganizationIDs)
	return p
}

func (p ScopePredicate) Allows(record ScopedRecord) bool {
	if p.denied {
		return false
	}
	return scopeDimensionAllows(p.tenantIDs, p.allTenants, record.TenantID) &&
		scopeDimensionAllows(p.ownerIDs, p.allOwners, record.OwnerID) &&
		scopeDimensionAllows(p.organizationIDs, p.allOrganizations, record.OrganizationID)
}

func constrainScopeDimension(trusted []string, all bool, requested []string) ([]string, bool, bool) {
	requested = cleanScopeIDs(requested)
	if len(requested) == 0 {
		return append([]string(nil), trusted...), all, false
	}
	if all {
		return requested, false, false
	}
	allowed := make(map[string]struct{}, len(trusted))
	for _, id := range trusted {
		allowed[id] = struct{}{}
	}
	intersection := make([]string, 0, len(requested))
	for _, id := range requested {
		if _, ok := allowed[id]; ok {
			intersection = append(intersection, id)
		}
	}
	if len(intersection) == 0 {
		return nil, false, true
	}
	return intersection, false, false
}

func scopeDimensionAllows(trusted []string, all bool, value string) bool {
	if all {
		return true
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, id := range trusted {
		if id == value {
			return true
		}
	}
	return false
}

func cleanScopeIDs(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
