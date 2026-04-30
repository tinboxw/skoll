package storageadapter

import (
	"testing"

	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
	"github.com/tinboxw/skoll/internal/module/storageadapter/contracttest"
	"github.com/tinboxw/skoll/internal/module/storageadapter/memory"
)

func TestInMemoryAdapterContract(t *testing.T) {
	contracttest.Run(t, func(t *testing.T) contracts.Adapter {
		t.Helper()
		a, err := memory.NewAdapter()
		if err != nil {
			t.Fatalf("new memory adapter failed: %v", err)
		}
		return a
	})
}
