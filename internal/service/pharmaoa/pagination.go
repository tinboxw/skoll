package pharmaoa

type ListPage[T any] struct {
	Items []T
	Total int64
}
