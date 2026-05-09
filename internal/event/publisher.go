package event

import "context"

type Publisher interface {
	Publish(ctx context.Context, evt Event) error
}

type publisher struct {
	bus Bus
}

func NewPublisher(bus Bus) Publisher {
	return &publisher{bus: bus}
}

func (p *publisher) Publish(ctx context.Context, evt Event) error {
	if p == nil || p.bus == nil {
		return nil
	}
	return p.bus.Publish(ctx, evt)
}
