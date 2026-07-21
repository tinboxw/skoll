package pluginsdk

import "testing"

func TestScopePredicateCannotBroadenTrustedDimensions(t *testing.T) {
	predicate, err := NewScopePredicate(TrustedScope{
		SubjectID: "employee-1", TenantIDs: []string{"tenant-a"}, OwnerIDs: []string{"employee-1"},
		OrganizationIDs: []string{"org-a", "org-a-child"},
	})
	if err != nil {
		t.Fatalf("NewScopePredicate error: %v", err)
	}
	forged := predicate.Constrain(ScopeFilter{
		TenantIDs: []string{"tenant-b"}, OwnerIDs: []string{"employee-2"}, OrganizationIDs: []string{"org-b"},
	})
	if !forged.Denied() || forged.Allows(ScopedRecord{TenantID: "tenant-b", OwnerID: "employee-2", OrganizationID: "org-b"}) {
		t.Fatalf("forged scope broadened trusted predicate: %+v", forged)
	}
	narrowed := predicate.Constrain(ScopeFilter{OrganizationIDs: []string{"org-a-child", "org-b"}})
	if narrowed.Denied() || len(narrowed.OrganizationIDs()) != 1 || narrowed.OrganizationIDs()[0] != "org-a-child" {
		t.Fatalf("valid narrowing failed: %+v", narrowed)
	}
	if !narrowed.Allows(ScopedRecord{TenantID: "tenant-a", OwnerID: "employee-1", OrganizationID: "org-a-child"}) {
		t.Fatal("narrowed predicate rejected an authorized record")
	}
}

func TestScopePredicateDefensivelyCopiesIDs(t *testing.T) {
	owners := []string{"employee-1"}
	predicate, err := NewScopePredicate(TrustedScope{SubjectID: "employee-1", AllTenants: true, OwnerIDs: owners, AllOrganizations: true})
	if err != nil {
		t.Fatalf("NewScopePredicate error: %v", err)
	}
	owners[0] = "employee-2"
	copyOfOwners := predicate.OwnerIDs()
	copyOfOwners[0] = "employee-3"
	if predicate.OwnerIDs()[0] != "employee-1" {
		t.Fatal("predicate ownership was mutable from caller data")
	}
}
