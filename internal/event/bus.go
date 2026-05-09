package event

import "context"

type Event interface {
	Name() string
}

type Handler func(context.Context, Event) error

type Bus interface {
	Publish(ctx context.Context, evt Event) error
	Subscribe(eventName string, handler Handler) (unsubscribe func())
}
