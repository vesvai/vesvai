package event

type DeadEvent struct {
	BaseEvent
	Event       Event
	subscribers []string
}

func NewDeadEvent(originalEvent Event) *DeadEvent {
	return &DeadEvent{
		BaseEvent: NewBaseEvent("dead_event"),
		Event:     originalEvent,
	}
}

func (d *DeadEvent) AddSubscriber(id string) {
	d.subscribers = append(d.subscribers, id)
}

func (d *DeadEvent) Subscribers() []string {
	return d.subscribers
}

type SystemEventType EventType

const (
	EventSystemInit     SystemEventType = "system:init"
	EventSystemReady    SystemEventType = "system:ready"
	EventSystemShutdown SystemEventType = "system:shutdown"
	EventSystemError    SystemEventType = "system:error"
	EventSystemConfig   SystemEventType = "system:config"
)

type SystemEvent struct {
	BaseEvent
	Data interface{}
}

func NewSystemEvent(eventType SystemEventType, data interface{}) *SystemEvent {
	return &SystemEvent{
		BaseEvent: NewBaseEvent(EventType(eventType)),
		Data:      data,
	}
}
