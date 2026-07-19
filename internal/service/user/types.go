package user

type CreateUserInput struct {
	Account      string `json:"account"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	PasswordHash string `json:"passwordHash"`
	DepartmentID string `json:"departmentId"`
	PositionID   string `json:"positionId"`
	ActorID      string `json:"actorId"`
}

type BatchCreateInput struct {
	Items  []CreateUserInput `json:"items"`
	Atomic bool              `json:"atomic"`
}

type BatchCreateResult struct {
	Index   int    `json:"index"`
	Account string `json:"account"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	ID      string `json:"id"`
}

type UpdateEmailInput struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	ActorID string `json:"actorId"`
}

type UpdateUserInput struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Status       string `json:"status"`
	DepartmentID string `json:"departmentId"`
	PositionID   string `json:"positionId"`
}

type ListInput struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}
