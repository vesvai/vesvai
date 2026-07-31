package plugin

import "github.com/vesvai/vesvai/internal/event"

const (
	EventPluginLoaded   event.EventType = "plugin:loaded"
	EventPluginUnloaded event.EventType = "plugin:unloaded"
	EventPluginStarted  event.EventType = "plugin:started"
	EventPluginStopped  event.EventType = "plugin:stopped"
	EventPluginError    event.EventType = "plugin:error"
	EventPluginInit     event.EventType = "plugin:init"
)

type PluginEvent struct {
	event.BaseEvent
	PluginName string `json:"plugin_name"`
	Version    string `json:"version,omitempty"`
	Error      error  `json:"error,omitempty"`
}

func NewPluginEvent(eventType event.EventType, pluginName, version string) *PluginEvent {
	return &PluginEvent{
		BaseEvent:  event.NewBaseEvent(eventType),
		PluginName: pluginName,
		Version:    version,
	}
}

func NewPluginErrorEvent(pluginName string, err error) *PluginEvent {
	return &PluginEvent{
		BaseEvent:  event.NewBaseEvent(EventPluginError),
		PluginName: pluginName,
		Error:      err,
	}
}
