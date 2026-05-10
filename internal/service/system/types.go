package system

type UpsertInput struct {
	Key       string
	Value     string
	Encrypted bool
}

type ListInput struct {
	Offset int
	Limit  int
}
