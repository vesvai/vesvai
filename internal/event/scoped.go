package event

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ScopedEventBus struct {
	parent     *eventBus
	prefix     string
	mu         sync.RWMutex
	filters    []EventType
	intercept  map[EventType]Handler
	scopedSubs map[EventType][]*Subscription
}

func NewScopedEventBus(parent EventBus, prefix string, filters ...EventType) *ScopedEventBus {
	parentBus, ok := parent.(*eventBus)
	if !ok {
		parentBus = &eventBus{
			subs:        make(map[EventType]*subscriptionList),
			requestChan: make(chan *Request, 1024),
			config:      DefaultEventBusConfig(),
			metrics:     &busMetrics{},
		}
	}

	return &ScopedEventBus{
		parent:     parentBus,
		prefix:     prefix,
		filters:    filters,
		intercept:  make(map[EventType]Handler),
		scopedSubs: make(map[EventType][]*Subscription),
	}
}

func (s *ScopedEventBus) Publish(ctx context.Context, event Event) error {
	if len(s.filters) > 0 {
		allowed := false
		for _, f := range s.filters {
			if event.Type() == f {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil
		}
	}

	if handler, ok := s.intercept[event.Type()]; ok {
		return handler.Handle(ctx, event)
	}

	return s.parent.Publish(ctx, event)
}

func (s *ScopedEventBus) PublishAsync(ctx context.Context, event Event) error {
	if len(s.filters) > 0 {
		allowed := false
		for _, f := range s.filters {
			if event.Type() == f {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil
		}
	}

	return s.parent.PublishAsync(ctx, event)
}

func (s *ScopedEventBus) Subscribe(eventType EventType, handler Handler, opts ...SubscriptionOption) (*Subscription, error) {
	sub, err := s.parent.Subscribe(eventType, handler, opts...)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.scopedSubs[eventType] = append(s.scopedSubs[eventType], sub)
	s.mu.Unlock()

	return sub, nil
}

func (s *ScopedEventBus) Unsubscribe(subscription *Subscription) error {
	err := s.parent.Unsubscribe(subscription)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for etype, subs := range s.scopedSubs {
		for i, sub := range subs {
			if sub.ID == subscription.ID {
				s.scopedSubs[etype] = append(subs[:i], subs[i+1:]...)
				return nil
			}
		}
	}

	return nil
}

func (s *ScopedEventBus) Request(ctx context.Context, event Event, timeout time.Duration) (Event, error) {
	if len(s.filters) > 0 {
		allowed := false
		for _, f := range s.filters {
			if event.Type() == f {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("event type %s not allowed in scope", event.Type())
		}
	}

	return s.parent.Request(ctx, event, timeout)
}

func (s *ScopedEventBus) PublishRequest(ctx context.Context, req *Request) error {
	return s.parent.PublishRequest(ctx, req)
}

func (s *ScopedEventBus) SubscribeRequest(eventType EventType, handler Handler, opts ...SubscriptionOption) (*Subscription, error) {
	return s.parent.SubscribeRequest(eventType, handler, opts...)
}

func (s *ScopedEventBus) HasSubscribers(eventType EventType) bool {
	return s.parent.HasSubscribers(eventType)
}

func (s *ScopedEventBus) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for etype, subs := range s.scopedSubs {
		for _, sub := range subs {
			s.parent.Unsubscribe(sub)
		}
		delete(s.scopedSubs, etype)
	}

	s.intercept = make(map[EventType]Handler)
	return nil
}

func (s *ScopedEventBus) Intercept(eventType EventType, handler Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.intercept[eventType] = handler
}

func (s *ScopedEventBus) RemoveIntercept(eventType EventType) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.intercept, eventType)
}

func (s *ScopedEventBus) AddFilter(eventType EventType) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.filters = append(s.filters, eventType)
}

func (s *ScopedEventBus) RemoveFilter(eventType EventType) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, f := range s.filters {
		if f == eventType {
			s.filters = append(s.filters[:i], s.filters[i+1:]...)
			return
		}
	}
}

func (s *ScopedEventBus) ClearFilters() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.filters = s.filters[:0]
}
