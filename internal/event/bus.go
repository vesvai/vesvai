package event

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

var (
	ErrHandlerNil   = errors.New("handler cannot be nil")
	ErrSubscription = errors.New("subscription not found")
	ErrBusClosed    = errors.New("event bus is closed")
)

type Request struct {
	Event    Event
	Response chan Event
	Error    chan error
	Timeout  time.Duration
}

type EventBus interface {
	Publish(ctx context.Context, event Event) error
	PublishAsync(ctx context.Context, event Event) error
	Subscribe(eventType EventType, handler Handler, opts ...SubscriptionOption) (*Subscription, error)
	Unsubscribe(sub *Subscription) error
	Request(ctx context.Context, event Event, timeout time.Duration) (Event, error)
	PublishRequest(ctx context.Context, req *Request) error
	SubscribeRequest(eventType EventType, handler Handler, opts ...SubscriptionOption) (*Subscription, error)
	HasSubscribers(eventType EventType) bool
	Close() error
}

type EventBusConfig struct {
	AsyncWorkers    int
	AsyncQueueSize  int
	RequestWorkers  int
	EnableDeadEvent bool
}

func DefaultEventBusConfig() EventBusConfig {
	return EventBusConfig{
		AsyncWorkers:    4,
		AsyncQueueSize:  1024,
		RequestWorkers:  4,
		EnableDeadEvent: true,
	}
}

type eventBus struct {
	mu           sync.RWMutex
	subs         map[EventType]*subscriptionList
	requestSubs  map[EventType]*subscriptionList
	asyncChan    chan asyncEvent
	requestChan  chan *Request
	workerPool   *workerPool
	requestPool  *workerPool
	deadEventBus *eventBus
	closed       atomic.Bool
	config       EventBusConfig
	metrics      *busMetrics
	loopWg       sync.WaitGroup
}

type subscriptionList struct {
	subs []*Subscription
	mu   sync.RWMutex
}

type asyncEvent struct {
	ctx   context.Context
	event Event
}

type busMetrics struct {
	publishCount    atomic.Int64
	subscribeCount  atomic.Int64
	handlerDuration atomic.Int64
	deadEventCount  atomic.Int64
}

type PrePublishHook func(ctx context.Context, event Event) error
type PostPublishHook func(ctx context.Context, event Event, err error)

func NewEventBus(configs ...EventBusConfig) EventBus {
	config := DefaultEventBusConfig()
	if len(configs) > 0 {
		config = configs[0]
	}

	bus := &eventBus{
		subs:        make(map[EventType]*subscriptionList),
		requestChan: make(chan *Request, config.AsyncQueueSize),
		config:      config,
		metrics:     &busMetrics{},
	}

	bus.asyncChan = make(chan asyncEvent, config.AsyncQueueSize)
	bus.workerPool = newWorkerPool("async-worker", config.AsyncWorkers)
	bus.requestPool = newWorkerPool("request-worker", config.RequestWorkers)

	if config.EnableDeadEvent {
		bus.deadEventBus = &eventBus{
			subs:        make(map[EventType]*subscriptionList),
			requestChan: make(chan *Request, config.AsyncQueueSize),
			config:      config,
			metrics:     &busMetrics{},
		}
		bus.deadEventBus.asyncChan = make(chan asyncEvent, config.AsyncQueueSize)
		bus.deadEventBus.workerPool = newWorkerPool("dead-async-worker", config.AsyncWorkers)
		bus.deadEventBus.requestPool = newWorkerPool("dead-request-worker", config.RequestWorkers)
		bus.deadEventBus.workerPool.Start()
		bus.deadEventBus.requestPool.Start()
		bus.deadEventBus.loopWg.Add(2)
		go func() {
			defer bus.deadEventBus.loopWg.Done()
			bus.deadEventBus.processAsyncEvents()
		}()
		go func() {
			defer bus.deadEventBus.loopWg.Done()
			bus.deadEventBus.processRequests()
		}()
	}

	bus.workerPool.Start()
	bus.requestPool.Start()

	bus.loopWg.Add(2)
	go func() {
		defer bus.loopWg.Done()
		bus.processAsyncEvents()
	}()
	go func() {
		defer bus.loopWg.Done()
		bus.processRequests()
	}()

	return bus
}

func (b *eventBus) Publish(ctx context.Context, event Event) error {
	if b.closed.Load() {
		return ErrBusClosed
	}

	start := time.Now()
	err := b.publishToSubscribers(ctx, event)
	elapsed := time.Since(start)

	b.metrics.publishCount.Add(1)
	b.metrics.handlerDuration.Add(elapsed.Nanoseconds())

	return err
}

func (b *eventBus) PublishAsync(ctx context.Context, event Event) error {
	if b.closed.Load() {
		return ErrBusClosed
	}

	select {
	case b.asyncChan <- asyncEvent{ctx: ctx, event: event}:
		b.metrics.publishCount.Add(1)
		return nil
	default:
		return errors.New("async channel full")
	}
}

func (b *eventBus) publishToSubscribers(ctx context.Context, event Event) error {
	eventType := event.Type()

	b.mu.RLock()
	list := b.subs[eventType]
	b.mu.RUnlock()

	if list == nil {
		if b.config.EnableDeadEvent && b.deadEventBus != nil {
			b.handleDeadEvent(ctx, event)
		}
		return nil
	}

	list.mu.RLock()
	subs := make([]*Subscription, len(list.subs))
	copy(subs, list.subs)
	list.mu.RUnlock()

	var wg sync.WaitGroup
	var firstErr error
	var mu sync.Mutex

	for _, sub := range subs {
		if !sub.Active() {
			continue
		}

		opts := b.getSubscriptionOptions(sub)
		if opts != nil && opts.Filter != nil && !opts.Filter(event) {
			continue
		}

		wg.Add(1)
		go func(s *Subscription) {
			defer wg.Done()
			if err := s.Handler.Handle(ctx, event); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}

			if s.Once {
				s.Deactive()
			}
		}(sub)
	}

	wg.Wait()
	return firstErr
}

func (b *eventBus) handleDeadEvent(ctx context.Context, event Event) {
	dead := GetDeadEvent()
	dead.Event = event

	b.mu.RLock()
	list := b.subs[event.Type()]
	if list != nil {
		list.mu.RLock()
		for _, sub := range list.subs {
			dead.AddSubscriber(sub.ID)
		}
		list.mu.RUnlock()
	}
	b.mu.RUnlock()

	b.metrics.deadEventCount.Add(1)
	b.deadEventBus.Publish(ctx, dead)
	PutDeadEvent(dead)
}

func (b *eventBus) Subscribe(eventType EventType, handler Handler, opts ...SubscriptionOption) (*Subscription, error) {
	if handler == nil {
		return nil, ErrHandlerNil
	}

	if b.closed.Load() {
		return nil, ErrBusClosed
	}

	sub := &Subscription{
		ID:        uuid.New().String(),
		EventType: eventType,
		Handler:   handler,
		Priority:  PriorityNormal,
		active:    true,
	}

	options := &SubscriptionOptions{}
	for _, opt := range opts {
		opt(options)
	}

	sub.Priority = options.Priority
	sub.Once = options.Once
	sub.Filter = options.Filter

	b.mu.Lock()
	list := b.subs[eventType]
	if list == nil {
		list = &subscriptionList{}
		b.subs[eventType] = list
	}
	b.mu.Unlock()

	list.mu.Lock()
	list.subs = append(list.subs, sub)
	list.mu.Unlock()

	b.metrics.subscribeCount.Add(1)

	go b.sortSubscriptions(eventType)

	return sub, nil
}

func (b *eventBus) SubscribeRequest(eventType EventType, handler Handler, opts ...SubscriptionOption) (*Subscription, error) {
	if handler == nil {
		return nil, ErrHandlerNil
	}

	if b.closed.Load() {
		return nil, ErrBusClosed
	}

	sub := &Subscription{
		ID:        uuid.New().String(),
		EventType: eventType,
		Handler:   handler,
		Priority:  PriorityNormal,
		active:    true,
	}

	options := &SubscriptionOptions{}
	for _, opt := range opts {
		opt(options)
	}

	sub.Priority = options.Priority

	b.mu.Lock()
	list := b.requestSubs[eventType]
	if list == nil {
		list = &subscriptionList{}
		b.requestSubs[eventType] = list
	}
	b.mu.Unlock()

	list.mu.Lock()
	list.subs = append(list.subs, sub)
	list.mu.Unlock()

	go b.sortRequestSubscriptions(eventType)

	return sub, nil
}

func (b *eventBus) sortSubscriptions(eventType EventType) {
	b.mu.RLock()
	list := b.subs[eventType]
	b.mu.RUnlock()

	if list == nil {
		return
	}

	list.mu.Lock()
	defer list.mu.Unlock()

	for i := 0; i < len(list.subs)-1; i++ {
		for j := i + 1; j < len(list.subs); j++ {
			if list.subs[i].Priority < list.subs[j].Priority {
				list.subs[i], list.subs[j] = list.subs[j], list.subs[i]
			}
		}
	}
}

func (b *eventBus) sortRequestSubscriptions(eventType EventType) {
	b.mu.RLock()
	list := b.requestSubs[eventType]
	b.mu.RUnlock()

	if list == nil {
		return
	}

	list.mu.Lock()
	defer list.mu.Unlock()

	for i := 0; i < len(list.subs)-1; i++ {
		for j := i + 1; j < len(list.subs); j++ {
			if list.subs[i].Priority < list.subs[j].Priority {
				list.subs[i], list.subs[j] = list.subs[j], list.subs[i]
			}
		}
	}
}

func (b *eventBus) Unsubscribe(sub *Subscription) error {
	if sub == nil {
		return ErrSubscription
	}

	b.mu.RLock()
	list := b.subs[sub.EventType]
	b.mu.RUnlock()

	if list == nil {
		return ErrSubscription
	}

	list.mu.Lock()
	defer list.mu.Unlock()

	for i, s := range list.subs {
		if s.ID == sub.ID {
			list.subs = append(list.subs[:i], list.subs[i+1:]...)
			return nil
		}
	}

	return ErrSubscription
}

func (b *eventBus) Request(ctx context.Context, event Event, timeout time.Duration) (Event, error) {
	if b.closed.Load() {
		return nil, ErrBusClosed
	}

	req := &Request{
		Event:    event,
		Response: make(chan Event, 1),
		Error:    make(chan error, 1),
		Timeout:  timeout,
	}

	err := b.PublishRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	select {
	case resp := <-req.Response:
		return resp, nil
	case err := <-req.Error:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, fmt.Errorf("request timeout after %v", timeout)
	}
}

func (b *eventBus) PublishRequest(ctx context.Context, req *Request) error {
	if b.closed.Load() {
		return ErrBusClosed
	}

	select {
	case b.requestChan <- req:
		return nil
	default:
		return errors.New("request channel full")
	}
}

func (b *eventBus) processRequests() {
	for {
		select {
		case req := <-b.requestChan:
			b.handleRequest(req)
		case <-b.workerPool.Done():
			return
		}
	}
}

func (b *eventBus) handleRequest(req *Request) {
	if req == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), req.Timeout)
	defer cancel()

	eventType := req.Event.Type()

	b.mu.RLock()
	list := b.requestSubs[eventType]
	b.mu.RUnlock()

	if list == nil {
		req.Error <- fmt.Errorf("no handler for %s", eventType)
		return
	}

	list.mu.RLock()
	subs := make([]*Subscription, len(list.subs))
	copy(subs, list.subs)
	list.mu.RUnlock()

	for _, sub := range subs {
		if !sub.Active() {
			continue
		}

		err := sub.Handler.Handle(ctx, req.Event)
		if err == nil {
			return
		}
	}

	req.Error <- fmt.Errorf("no handler processed request for %s", eventType)
}

func (b *eventBus) processAsyncEvents() {
	for {
		select {
		case ae := <-b.asyncChan:
			if ae.event != nil {
				b.publishToSubscribers(ae.ctx, ae.event)
			}
		case <-b.workerPool.Done():
			return
		}
	}
}

func (b *eventBus) HasSubscribers(eventType EventType) bool {
	b.mu.RLock()
	list := b.subs[eventType]
	b.mu.RUnlock()

	if list == nil {
		return false
	}

	list.mu.RLock()
	defer list.mu.RUnlock()

	for _, sub := range list.subs {
		if sub.Active() {
			return true
		}
	}

	return false
}

func (b *eventBus) Close() error {
	if b.closed.CompareAndSwap(false, true) {
		close(b.asyncChan)
		close(b.requestChan)

		b.workerPool.Stop()
		b.requestPool.Stop()

		b.loopWg.Wait()

		if b.deadEventBus != nil {
			b.deadEventBus.Close()
		}

		b.mu.Lock()
		b.subs = make(map[EventType]*subscriptionList)
		b.requestSubs = make(map[EventType]*subscriptionList)
		b.mu.Unlock()

		return nil
	}

	return ErrBusClosed
}

func (b *eventBus) getSubscriptionOptions(sub *Subscription) *SubscriptionOptions {
	return &SubscriptionOptions{
		Priority: sub.Priority,
		Once:     sub.Once,
		Filter:   sub.Filter,
	}
}
