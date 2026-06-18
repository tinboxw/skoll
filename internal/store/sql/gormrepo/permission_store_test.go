package gormrepo

import (
	"testing"

	permissionrepo "github.com/tinboxw/skoll/internal/repository/permission"
)

func TestPermissionStoreImplementsRepository(t *testing.T) {
	var _ permissionrepo.PermissionRepository = (*PermissionStore)(nil)
}
