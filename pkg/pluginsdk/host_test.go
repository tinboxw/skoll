package pluginsdk

import (
	"context"
	"testing"
)

type hostTestTransactions struct{}

func (hostTestTransactions) Within(_ context.Context, fn func(Transaction) error) error {
	return fn(hostTestTransaction{})
}

type hostTestTransaction struct{}

func (hostTestTransaction) Context() context.Context { return context.Background() }

type hostTestDataScopes struct{}

func (hostTestDataScopes) Resolve(_ context.Context, _ Permission) (ScopePredicate, error) {
	return NewScopePredicate(TrustedScope{
		SubjectID: "test", AllTenants: true, AllOwners: true, AllOrganizations: true,
	})
}

func TestHostServicesValidateRequiresEveryPort(t *testing.T) {
	valid := HostServices{Transactions: hostTestTransactions{}, DataScopes: hostTestDataScopes{}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid host services rejected: %v", err)
	}
	if err := (HostServices{DataScopes: hostTestDataScopes{}}).Validate(); err == nil {
		t.Fatal("missing transaction service must be rejected")
	}
	if err := (HostServices{Transactions: hostTestTransactions{}}).Validate(); err == nil {
		t.Fatal("missing data-scope service must be rejected")
	}
}
