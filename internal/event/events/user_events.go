package events

import "time"

const (
	UserCreatedEventName  = "user.created"
	UserDisabledEventName = "user.disabled"
)

type UserCreated struct {
	ID         string
	Username   string
	ActorID    string
	OccurredAt time.Time
}

func (e UserCreated) Name() string { return UserCreatedEventName }

type UserDisabled struct {
	ID         string
	ActorID    string
	OccurredAt time.Time
}

func (e UserDisabled) Name() string { return UserDisabledEventName }
