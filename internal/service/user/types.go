package user

type CreateUserInput struct {
	Account      string
	Name         string
	Email        string
	PasswordHash string
	ActorID      string
}

type BatchCreateInput struct {
	Items  []CreateUserInput
	Atomic bool
}

type BatchCreateResult struct {
	Index   int
	Account string
	Success bool
	Message string
	ID      string
}

type UpdateEmailInput struct {
	ID      string
	Email   string
	ActorID string
}

type UpdateUserInput struct {
	ID     string
	Name   string
	Email  string
	Status string
}

type ListInput struct {
	Offset int
	Limit  int
}
