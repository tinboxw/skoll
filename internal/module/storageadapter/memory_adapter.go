package storageadapter

import "github.com/tinboxw/skoll/internal/module/storageadapter/memory"

func NewInMemoryAdapter() Adapter {
	adapter, err := memory.NewAdapter()
	if err != nil {
		panic(err)
	}
	return adapter
}
