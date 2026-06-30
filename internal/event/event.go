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
