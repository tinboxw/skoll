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

type DictionaryTypeInput struct {
	ID          string
	Code        string
	Name        string
	Description string
	Status      string
	Sort        int
	Builtin     bool
}

type DictionaryTypeListInput struct {
	Offset int
	Limit  int
}

type DictionaryItemInput struct {
	ID       string
	TypeCode string
	Label    string
	Value    string
	Status   string
	Sort     int
	Builtin  bool
}

type DictionaryItemListInput struct {
	TypeCode string
	Offset   int
	Limit    int
}
