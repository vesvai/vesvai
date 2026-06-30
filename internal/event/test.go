package event

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type testEvent struct {
	BaseEvent
	payload string
}

func newTestEvent(eventType EventType, payload string) *testEvent {
	return &testEvent{
		BaseEvent: NewBaseEvent(eventType),
		payload:   payload,
	}
}

type testHandler struct {
	mu      sync.Mutex
	calls   []Event
	err     error
	delay   time.Duration
	called  int32
	calledC chan struct{}
}

func newTestHandler() *testHandler {
	return &testHandler{
		calls:   make([]Event, 0),
		calledC: make(chan struct{}, 1),
	}
}

func (h *testHandler) Handle(ctx context.Context, event Event) error {
	atomic.AddInt32(&h.called, 1)
	h.mu.Lock()
	h.calls = append(h.calls, event)
	h.mu.Unlock()

	select {
	case h.calledC <- struct{}{}:
	default:
	}

	if h.delay > 0 {
		time.Sleep(h.delay)
	}
	return h.err
}

func (h *testHandler) CallCount() int {
	return int(atomic.LoadInt32(&h.called))
}

func (h *testHandler) Events() []Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]Event, len(h.calls))
	copy(out, h.calls)
	return out
}

type errHandler struct {
	err error
}

func (h *errHandler) Handle(ctx context.Context, event Event) error {
	return h.err
}

func TestEventBus_DeadEvent(t *testing.T) {
	bus := NewEventBus(EventBusConfig{EnableDeadEvent: true})
	defer bus.Close()

	eb := bus.(*eventBus)

	deadHandler := newTestHandler()
	eb.deadEventBus.Subscribe("dead_event", deadHandler)

	bus.Publish(context.Background(), newTestEvent("no.subscribers", "data"))

	select {
	case <-deadHandler.calledC:
	case <-time.After(time.Second):
		t.Fatal("dead event handler not called")
	}

	events := deadHandler.Events()
	if len(events) != 1 {
		t.Fatalf("dead handler received %d events, want 1", len(events))
	}

	dead, ok := events[0].(*DeadEvent)
	if !ok {
		t.Fatal("dead event is not *DeadEvent")
	}

	if dead.Event.Type() != "no.subscribers" {
		t.Errorf("dead event original type = %q, want %q", dead.Event.Type(), "no.subscribers")
	}
}

func TestEventBus_DeadEvent_Disabled(t *testing.T) {
	bus := NewEventBus(EventBusConfig{EnableDeadEvent: false})
	defer bus.Close()

	h := newTestHandler()
	bus.Subscribe("dead_event", h)

	bus.Publish(context.Background(), newTestEvent("no.subs", ""))

	select {
	case <-h.calledC:
		t.Error("dead event handler should not be called when dead events disabled")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}

func TestEventBus_PublishAsync_ChannelFull(t *testing.T) {
	config := EventBusConfig{
		AsyncWorkers:   1,
		AsyncQueueSize: 2,
	}
	bus := NewEventBus(config)
	defer bus.Close()

	bus.Subscribe("fill.event", newTestHandler())

	for i := 0; i < 100; i++ {
		err := bus.PublishAsync(context.Background(), newTestEvent("fill.event", ""))
		if err != nil {
			return
		}
	}

	t.Error("PublishAsync() should eventually error when channel full")
}

func TestEventBus_PublishRequest_ChannelFull(t *testing.T) {
	config := EventBusConfig{
		RequestWorkers: 1,
		AsyncQueueSize: 1,
	}
	bus := NewEventBus(config)
	defer bus.Close()

	req1 := &Request{
		Event:    newTestEvent("full.req", ""),
		Response: make(chan Event, 1),
		Error:    make(chan error, 1),
		Timeout:  5 * time.Second,
	}

	bus.PublishRequest(context.Background(), req1)

	req2 := &Request{
		Event:    newTestEvent("full.req", ""),
		Response: make(chan Event, 1),
		Error:    make(chan error, 1),
		Timeout:  5 * time.Second,
	}

	err := bus.PublishRequest(context.Background(), req2)
	if err == nil {
		t.Error("PublishRequest() should error when channel full")
	}
}

func TestEventBus_Subscribe_MultipleEventTypes(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	h1 := newTestHandler()
	h2 := newTestHandler()

	bus.Subscribe("type.a", h1)
	bus.Subscribe("type.b", h2)

	bus.Publish(context.Background(), newTestEvent("type.a", ""))
	bus.Publish(context.Background(), newTestEvent("type.b", ""))

	if h1.CallCount() != 1 || h2.CallCount() != 1 {
		t.Errorf("h1=%d, h2=%d, want both 1", h1.CallCount(), h2.CallCount())
	}
}

func TestEventBus_Publish_SameEventTypeMultipleSubscribers(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	var count atomic.Int32
	for i := 0; i < 10; i++ {
		bus.Subscribe("same.type", EventHandlerFunc(func(ctx context.Context, event Event) error {
			count.Add(1)
			return nil
		}))
	}

	bus.Publish(context.Background(), newTestEvent("same.type", ""))

	if count.Load() != 10 {
		t.Errorf("count = %d, want 10", count.Load())
	}
}

func TestEventBus_Unsubscribe_DoesNotAffectOtherTypes(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	h1 := newTestHandler()
	h2 := newTestHandler()

	sub1, _ := bus.Subscribe("type.a", h1)
	bus.Subscribe("type.b", h2)

	bus.Unsubscribe(sub1)

	bus.Publish(context.Background(), newTestEvent("type.a", ""))
	bus.Publish(context.Background(), newTestEvent("type.b", ""))

	if h1.CallCount() != 0 {
		t.Error("h1 should not be called after unsubscribe")
	}
	if h2.CallCount() != 1 {
		t.Error("h2 should still be called")
	}
}

func TestScopedEventBus_ConcurrentPublish(t *testing.T) {
	parent := NewEventBus()
	defer parent.Close()

	scoped := NewScopedEventBus(parent, "concurrent")

	var count atomic.Int32
	scoped.Subscribe("conc.event", EventHandlerFunc(func(ctx context.Context, event Event) error {
		count.Add(1)
		return nil
	}))

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			scoped.Publish(context.Background(), newTestEvent("conc.event", ""))
		}()
	}
	wg.Wait()

	if count.Load() != 50 {
		t.Errorf("count = %d, want 50", count.Load())
	}
}

func TestScopedEventBus_ConcurrentSubscribe(t *testing.T) {
	parent := NewEventBus()
	defer parent.Close()

	scoped := NewScopedEventBus(parent, "concurrent.sub")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			scoped.Subscribe("conc.sub", newTestHandler())
		}()
	}
	wg.Wait()

	if !scoped.HasSubscribers("conc.sub") {
		t.Error("HasSubscribers() = false after concurrent subscribes")
	}
}

func TestEventBus_Publish_PanicInHandler(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	var handlerCalled atomic.Int32
	bus.Subscribe("panic.event", EventHandlerFunc(func(ctx context.Context, event Event) error {
		handlerCalled.Add(1)
		panic("handler panic")
	}))

	bus.Publish(context.Background(), newTestEvent("panic.event", ""))

	if handlerCalled.Load() != 1 {
		t.Errorf("handler called %d times, want 1", handlerCalled.Load())
	}
}

func TestEventBus_GetSubscriptionOptions(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	eb := bus.(*eventBus)

	sub := &Subscription{
		ID:       "test-id",
		Priority: PriorityHigh,
		Once:     true,
	}

	opts := eb.getSubscriptionOptions(sub)
	if opts.Priority != PriorityHigh {
		t.Errorf("Priority = %v, want %v", opts.Priority, PriorityHigh)
	}
	if !opts.Once {
		t.Error("Once = false, want true")
	}
}

func TestEventBus_SortSubscriptions(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	eb := bus.(*eventBus)

	sub1, _ := bus.Subscribe("sort.test", newTestHandler(), WithPriority(PriorityLow))
	sub2, _ := bus.Subscribe("sort.test", newTestHandler(), WithPriority(PriorityHigh))
	sub3, _ := bus.Subscribe("sort.test", newTestHandler(), WithPriority(PriorityNormal))

	// Wait for sort goroutine
	time.Sleep(50 * time.Millisecond)

	eb.mu.RLock()
	list := eb.subs["sort.test"]
	eb.mu.RUnlock()

	if list == nil {
		t.Fatal("subscription list is nil")
	}

	list.mu.RLock()
	defer list.mu.RUnlock()

	// Find positions
	positions := map[string]int{}
	for i, s := range list.subs {
		positions[s.ID] = i
	}

	if positions[sub2.ID] >= positions[sub3.ID] || positions[sub3.ID] >= positions[sub1.ID] {
		t.Error("subscriptions not sorted by priority (high before normal before low)")
	}
}

func TestEventBus_SortRequestSubscriptions(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	eb := bus.(*eventBus)

	sub1, _ := bus.SubscribeRequest("sort.req", newTestHandler(), WithPriority(PriorityLow))
	sub2, _ := bus.SubscribeRequest("sort.req", newTestHandler(), WithPriority(PriorityHigh))

	time.Sleep(50 * time.Millisecond)

	eb.mu.RLock()
	list := eb.requestSubs["sort.req"]
	eb.mu.RUnlock()

	if list == nil {
		t.Fatal("request subscription list is nil")
	}

	list.mu.RLock()
	defer list.mu.RUnlock()

	positions := map[string]int{}
	for i, s := range list.subs {
		positions[s.ID] = i
	}

	if positions[sub2.ID] >= positions[sub1.ID] {
		t.Error("request subscriptions not sorted by priority")
	}
}

func TestEventBus_SortSubscriptions_NilList(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	eb := bus.(*eventBus)

	// Should not panic
	eb.sortSubscriptions("nonexistent")
	eb.sortRequestSubscriptions("nonexistent")
}

func BenchmarkEventBus_Publish(b *testing.B) {
	bus := NewEventBus()
	defer bus.Close()

	bus.Subscribe("bench.event", newTestHandler())
	ctx := context.Background()
	e := newTestEvent("bench.event", "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(ctx, e)
	}
}

func BenchmarkEventBus_PublishAsync(b *testing.B) {
	bus := NewEventBus()
	defer bus.Close()

	bus.Subscribe("bench.async", newTestHandler())
	ctx := context.Background()
	e := newTestEvent("bench.async", "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.PublishAsync(ctx, e)
	}
}

func BenchmarkEventBus_Publish_MultipleSubscribers(b *testing.B) {
	bus := NewEventBus()
	defer bus.Close()

	for i := 0; i < 10; i++ {
		bus.Subscribe("bench.multi", newTestHandler())
	}

	ctx := context.Background()
	e := newTestEvent("bench.multi", "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(ctx, e)
	}
}
