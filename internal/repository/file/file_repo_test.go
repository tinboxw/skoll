package file

import (
	"testing"

	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestListFilterUsesDomainTypes(t *testing.T) {
	filter := ListFilter{
		OwnerType:      "user",
		OwnerID:        shared.ID("u-1"),
		Visibility:     domainfile.VisibilityPrivate,
		StorageDriver:  "local",
		Status:         domainfile.StatusAvailable,
		SourceModule:   "system",
		SourcePluginID: "demo",
	}
	if filter.OwnerID.String() != "u-1" || filter.Status != domainfile.StatusAvailable {
		t.Fatalf("filter = %#v", filter)
	}
}
