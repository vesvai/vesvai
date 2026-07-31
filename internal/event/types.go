package event

import (
	"context"
	"sync"
	"time"
)

type EventType string

func (t EventType) String() string {
	return string(t)
}

type Event interface {
	Type() EventType
	Timestamp() time.Time
}

type BaseEvent struct {
	eventType EventType
	timestamp time.Time
}

func (e *BaseEvent) Type() EventType {
	return e.eventType
}

func (e *BaseEvent) Timestamp() time.Time {
	return e.timestamp
}

func NewBaseEvent(eventType EventType) BaseEvent {
	return BaseEvent{
		eventType: eventType,
		timestamp: time.Now(),
	}
}

type EventHandler func(ctx context.Context, event Event) error

type EventHandlerFunc func(ctx context.Context, event Event) error

func (f EventHandlerFunc) Handle(ctx context.Context, event Event) error {
	return f(ctx, event)
}

type Handler interface {
	Handle(ctx context.Context, event Event) error
}

type Priority int

const (
	PriorityLow    Priority = 0
	PriorityNormal Priority = 50
	PriorityHigh   Priority = 100
)

type Subscription struct {
	ID        string
	EventType EventType
	Handler   Handler
	Priority  Priority
	Once      bool
	Filter    FilterFunc
	mu        sync.RWMutex
	active    bool
}

func (s *Subscription) Active() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

func (s *Subscription) Deactive() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active = false
}

type FilterFunc func(event Event) bool

type SubscriptionOptions struct {
	Priority Priority
	Filter   FilterFunc
	Once     bool
}

type SubscriptionOption func(*SubscriptionOptions)

func WithPriority(priority Priority) SubscriptionOption {
	return func(opts *SubscriptionOptions) {
		opts.Priority = priority
	}
}
func WithFilter(filter FilterFunc) SubscriptionOption {
	return func(opts *SubscriptionOptions) {
		opts.Filter = filter
	}
}

func WithOnce(once bool) SubscriptionOption {
	return func(opts *SubscriptionOptions) {
		opts.Once = once
	}
}
