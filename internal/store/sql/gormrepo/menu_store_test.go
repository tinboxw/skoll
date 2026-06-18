package gormrepo

import (
	"testing"

	menurepo "github.com/tinboxw/skoll/internal/repository/menu"
)

func TestMenuStoreImplementsRepository(t *testing.T) {
	var _ menurepo.MenuRepository = (*MenuStore)(nil)
}
