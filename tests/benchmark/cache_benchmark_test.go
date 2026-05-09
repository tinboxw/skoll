package benchmark

import (
	"strconv"
	"testing"

	"github.com/tinboxw/skoll/internal/cache/local"
)

func BenchmarkLocalCacheSetGet(b *testing.B) {
	cache := local.NewLRUCache(4096)
	payload := []byte("benchmark-payload")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "k-" + strconv.Itoa(i)
		cache.Set(key, payload, 0)
		_, _ = cache.Get(key)
	}
}
