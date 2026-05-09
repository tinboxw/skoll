package user

type CreateUserInput struct {
	Username     string
	DisplayName  string
	Email        string
	PasswordHash string
	ActorID      string
}

type UpdateEmailInput struct {
	ID      string
	Email   string
	ActorID string
}

type ListInput struct {
	Offset int
	Limit  int
}
