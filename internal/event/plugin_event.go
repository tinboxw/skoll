package event

import "github.com/tinboxw/skoll/pkg/pluginsdk"

type PluginEvent struct {
	Envelope pluginsdk.EventEnvelope `json:"envelope"`
}

func (e PluginEvent) Name() string {
	return e.Envelope.Name
}

func (e PluginEvent) Validate() error {
	return e.Envelope.Payload.Validate()
}

var _ Event = PluginEvent{}
