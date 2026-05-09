package event

type Subscriber interface {
	Subscribe(eventName string, handler Handler) (unsubscribe func())
}

type subscriber struct {
	bus Bus
}

func NewSubscriber(bus Bus) Subscriber {
	return &subscriber{bus: bus}
}

func (s *subscriber) Subscribe(eventName string, handler Handler) (unsubscribe func()) {
	if s == nil || s.bus == nil {
		return func() {}
	}
	return s.bus.Subscribe(eventName, handler)
}
