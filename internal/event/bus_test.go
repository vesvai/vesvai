package event

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewEventBus_DefaultConfig(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	eb, ok := bus.(*eventBus)
	if !ok {
		t.Fatal("NewEventBus() did not return *eventBus")
	}

	if eb.config.AsyncWorkers != 4 {
		t.Errorf("AsyncWorkers = %d, want 4", eb.config.AsyncWorkers)
	}
	if eb.config.AsyncQueueSize != 1024 {
		t.Errorf("AsyncQueueSize = %d, want 1024", eb.config.AsyncQueueSize)
	}
	if eb.config.RequestWorkers != 4 {
		t.Errorf("RequestWorkers = %d, want 4", eb.config.RequestWorkers)
	}
	if !eb.config.EnableDeadEvent {
		t.Error("EnableDeadEvent = false, want true")
	}
}

func TestNewEventBus_CustomConfig(t *testing.T) {
	config := EventBusConfig{
		AsyncWorkers:    8,
		AsyncQueueSize:  512,
		RequestWorkers:  2,
		EnableDeadEvent: false,
	}
	bus := NewEventBus(config)
	defer bus.Close()

	eb := bus.(*eventBus)
	if eb.config.AsyncWorkers != 8 {
		t.Errorf("AsyncWorkers = %d, want 8", eb.config.AsyncWorkers)
	}
	if eb.config.AsyncQueueSize != 512 {
		t.Errorf("AsyncQueueSize = %d, want 512", eb.config.AsyncQueueSize)
	}
	if eb.config.RequestWorkers != 2 {
		t.Errorf("RequestWorkers = %d, want 2", eb.config.RequestWorkers)
	}
	if eb.config.EnableDeadEvent {
		t.Error("EnableDeadEvent = true, want false")
	}
}

func TestDefaultEventBusConfig(t *testing.T) {
	config := DefaultEventBusConfig()
	if config.AsyncWorkers != 4 || config.AsyncQueueSize != 1024 ||
		config.RequestWorkers != 4 || !config.EnableDeadEvent {
		t.Errorf("DefaultEventBusConfig() = %+v", config)
	}
}

func TestEventBus_Publish_NoSubscribers(t *testing.T) {
	bus := NewEventBus(EventBusConfig{EnableDeadEvent: false})
	defer bus.Close()

	err := bus.Publish(context.Background(), newTestEvent("no.subscribers", ""))
	if err != nil {
		t.Errorf("Publish() with no subscribers error = %v", err)
	}
}

func TestEventBus_Publish_ToSubscribers(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	h := newTestHandler()
	_, err := bus.Subscribe("test.event", h)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	e := newTestEvent("test.event", "payload")
	err = bus.Publish(context.Background(), e)
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	if h.CallCount() != 1 {
		t.Errorf("handler called %d times, want 1", h.CallCount())
	}

	events := h.Events()
	if len(events) != 1 || events[0].Type() != "test.event" {
		t.Errorf("handler received wrong event: %v", events)
	}
}

func TestEventBus_Publish_MultipleSubscribers(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	h1 := newTestHandler()
	h2 := newTestHandler()

	bus.Subscribe("multi.test", h1)
	bus.Subscribe("multi.test", h2)

	bus.Publish(context.Background(), newTestEvent("multi.test", ""))

	if h1.CallCount() != 1 {
		t.Errorf("h1 called %d times, want 1", h1.CallCount())
	}
	if h2.CallCount() != 1 {
		t.Errorf("h2 called %d times, want 1", h2.CallCount())
	}
}

func TestEventBus_Publish_FilteredOut(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	h := newTestHandler()
	bus.Subscribe("target.event", h,
		WithFilter(func(e Event) bool {
			return e.(*testEvent).payload == "pass"
		}),
	)

	// Should be filtered out
	bus.Publish(context.Background(), newTestEvent("target.event", "fail"))
	if h.CallCount() != 0 {
		t.Errorf("handler called %d times for filtered event, want 0", h.CallCount())
	}

	// Should pass filter
	bus.Publish(context.Background(), newTestEvent("target.event", "pass"))
	if h.CallCount() != 1 {
		t.Errorf("handler called %d times for passing event, want 1", h.CallCount())
	}
}

func TestEventBus_Publish_OnceSubscription(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	h := newTestHandler()
	bus.Subscribe("once.event", h, WithOnce(true))

	bus.Publish(context.Background(), newTestEvent("once.event", ""))
	if h.CallCount() != 1 {
		t.Errorf("handler called %d times after first publish, want 1", h.CallCount())
	}

	// Second publish - handler should be deactivated
	bus.Publish(context.Background(), newTestEvent("once.event", ""))
	if h.CallCount() != 1 {
		t.Errorf("handler called %d times after second publish, want 1 (once)", h.CallCount())
	}
}

func TestEventBus_Publish_HandlerError(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	wantErr := errors.New("handler failed")
	bus.Subscribe("err.event", &errHandler{err: wantErr})

	err := bus.Publish(context.Background(), newTestEvent("err.event", ""))
	if !errors.Is(err, wantErr) {
		t.Errorf("Publish() error = %v, want %v", err, wantErr)
	}
}

func TestEventBus_Publish_AfterClose(t *testing.T) {
	bus := NewEventBus()
	bus.Close()

	err := bus.Publish(context.Background(), newTestEvent("closed", ""))
	if !errors.Is(err, ErrBusClosed) {
		t.Errorf("Publish() after Close() error = %v, want %v", err, ErrBusClosed)
	}
}

func TestEventBus_PublishAsync(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	h := newTestHandler()
	bus.Subscribe("async.event", h)

	err := bus.PublishAsync(context.Background(), newTestEvent("async.event", "data"))
	if err != nil {
		t.Fatalf("PublishAsync() error = %v", err)
	}

	// Wait for async processing
	select {
	case <-h.calledC:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for async handler")
	}

	if h.CallCount() != 1 {
		t.Errorf("handler called %d times, want 1", h.CallCount())
	}
}

func TestEventBus_PublishAsync_AfterClose(t *testing.T) {
	bus := NewEventBus()
	bus.Close()

	err := bus.PublishAsync(context.Background(), newTestEvent("closed", ""))
	if !errors.Is(err, ErrBusClosed) {
		t.Errorf("PublishAsync() after Close() error = %v, want %v", err, ErrBusClosed)
	}
}

func TestEventBus_Subscribe_NilHandler(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	_, err := bus.Subscribe("test", nil)
	if !errors.Is(err, ErrHandlerNil) {
		t.Errorf("Subscribe(nil) error = %v, want %v", err, ErrHandlerNil)
	}
}

func TestEventBus_Subscribe_AfterClose(t *testing.T) {
	bus := NewEventBus()
	bus.Close()

	_, err := bus.Subscribe("test", newTestHandler())
	if !errors.Is(err, ErrBusClosed) {
		t.Errorf("Subscribe() after Close() error = %v, want %v", err, ErrBusClosed)
	}
}

func TestEventBus_Subscribe_WithPriority(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	eb := bus.(*eventBus)

	sub1, _ := bus.Subscribe("prio.test", newTestHandler(), WithPriority(PriorityLow))
	sub2, _ := bus.Subscribe("prio.test", newTestHandler(), WithPriority(PriorityHigh))

	if sub1.Priority != PriorityLow {
		t.Errorf("sub1.Priority = %v, want %v", sub1.Priority, PriorityLow)
	}
	if sub2.Priority != PriorityHigh {
		t.Errorf("sub2.Priority = %v, want %v", sub2.Priority, PriorityHigh)
	}

	// Wait for sort goroutine
	time.Sleep(50 * time.Millisecond)

	eb.mu.RLock()
	list := eb.subs["prio.test"]
	eb.mu.RUnlock()

	if list == nil {
		t.Fatal("subscription list is nil")
	}

	list.mu.RLock()
	defer list.mu.RUnlock()

	if len(list.subs) >= 2 {
		positions := map[string]int{}
		for i, s := range list.subs {
			positions[s.ID] = i
		}
		if positions[sub2.ID] >= positions[sub1.ID] {
			t.Error("high priority subscription should come before low priority")
		}
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	h := newTestHandler()
	sub, _ := bus.Subscribe("unsub.test", h)

	err := bus.Unsubscribe(sub)
	if err != nil {
		t.Fatalf("Unsubscribe() error = %v", err)
	}

	// Should not receive events after unsubscribe
	bus.Publish(context.Background(), newTestEvent("unsub.test", ""))
	if h.CallCount() != 0 {
		t.Errorf("handler called %d times after unsubscribe, want 0", h.CallCount())
	}
}

func TestEventBus_Unsubscribe_Nil(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	err := bus.Unsubscribe(nil)
	if !errors.Is(err, ErrSubscription) {
		t.Errorf("Unsubscribe(nil) error = %v, want %v", err, ErrSubscription)
	}
}

func TestEventBus_Unsubscribe_NotFound(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	fakeSub := &Subscription{
		ID:        uuid.New().String(),
		EventType: "nonexistent",
	}

	err := bus.Unsubscribe(fakeSub)
	if !errors.Is(err, ErrSubscription) {
		t.Errorf("Unsubscribe() on nonexistent error = %v, want %v", err, ErrSubscription)
	}
}

func TestEventBus_Unsubscribe_WrongEventType(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	sub, _ := bus.Subscribe("real.type", newTestHandler())

	// Try to unsubscribe with wrong event type
	wrongSub := &Subscription{
		ID:        sub.ID,
		EventType: "wrong.type",
	}

	err := bus.Unsubscribe(wrongSub)
	if !errors.Is(err, ErrSubscription) {
		t.Errorf("Unsubscribe() wrong type error = %v, want %v", err, ErrSubscription)
	}
}

func TestEventBus_HasSubscribers(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	if bus.HasSubscribers("exists") {
		t.Error("HasSubscribers() = true, want false before subscribe")
	}

	sub, _ := bus.Subscribe("exists", newTestHandler())
	if !bus.HasSubscribers("exists") {
		t.Error("HasSubscribers() = false, want true after subscribe")
	}

	bus.Unsubscribe(sub)
	if bus.HasSubscribers("exists") {
		t.Error("HasSubscribers() = true, want false after unsubscribe")
	}
}

func TestEventBus_HasSubscribers_AllDeactivated(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	sub, _ := bus.Subscribe("deactivated", newTestHandler())
	sub.Deactive()

	if bus.HasSubscribers("deactivated") {
		t.Error("HasSubscribers() = true, want false when all subscribers deactivated")
	}
}

func TestEventBus_Close(t *testing.T) {
	bus := NewEventBus()

	err := bus.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Second close should return ErrBusClosed
	err = bus.Close()
	if !errors.Is(err, ErrBusClosed) {
		t.Errorf("Close() twice error = %v, want %v", err, ErrBusClosed)
	}
}

func TestEventBus_Close_ClearsSubscriptions(t *testing.T) {
	bus := NewEventBus()
	h := newTestHandler()
	bus.Subscribe("clear.test", h)

	bus.Close()

	eb := bus.(*eventBus)
	eb.mu.RLock()
	subsLen := len(eb.subs)
	reqSubsLen := len(eb.requestSubs)
	eb.mu.RUnlock()

	if subsLen != 0 {
		t.Errorf("subs not cleared, len = %d", subsLen)
	}
	if reqSubsLen != 0 {
		t.Errorf("requestSubs not cleared, len = %d", reqSubsLen)
	}
}

func TestEventBus_Request_Timeout(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	_, err := bus.Request(context.Background(), newTestEvent("timeout.test", ""), 100*time.Millisecond)
	if err == nil {
		t.Error("Request() with no handler should error")
	}
}

func TestEventBus_Request_ContextCancelled(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := bus.Request(ctx, newTestEvent("ctx.cancel", ""), 5*time.Second)
	if err == nil {
		t.Error("Request() with cancelled context should error")
	}
}

func TestEventBus_Request_AfterClose(t *testing.T) {
	bus := NewEventBus()
	bus.Close()

	_, err := bus.Request(context.Background(), newTestEvent("closed", ""), time.Second)
	if !errors.Is(err, ErrBusClosed) {
		t.Errorf("Request() after Close() error = %v, want %v", err, ErrBusClosed)
	}
}

func TestEventBus_PublishRequest(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	req := &Request{
		Event:    newTestEvent("pub.req", ""),
		Response: make(chan Event, 1),
		Error:    make(chan error, 1),
		Timeout:  5 * time.Second,
	}

	err := bus.PublishRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("PublishRequest() error = %v", err)
	}
}

func TestEventBus_PublishRequest_AfterClose(t *testing.T) {
	bus := NewEventBus()
	bus.Close()

	req := &Request{
		Event:    newTestEvent("closed", ""),
		Response: make(chan Event, 1),
		Error:    make(chan error, 1),
		Timeout:  5 * time.Second,
	}

	err := bus.PublishRequest(context.Background(), req)
	if !errors.Is(err, ErrBusClosed) {
		t.Errorf("PublishRequest() after Close() error = %v, want %v", err, ErrBusClosed)
	}
}

func TestEventBus_SubscribeRequest_NilHandler(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	_, err := bus.SubscribeRequest("test", nil)
	if !errors.Is(err, ErrHandlerNil) {
		t.Errorf("SubscribeRequest(nil) error = %v, want %v", err, ErrHandlerNil)
	}
}

func TestEventBus_SubscribeRequest_AfterClose(t *testing.T) {
	bus := NewEventBus()
	bus.Close()

	_, err := bus.SubscribeRequest("test", newTestHandler())
	if !errors.Is(err, ErrBusClosed) {
		t.Errorf("SubscribeRequest() after Close() error = %v, want %v", err, ErrBusClosed)
	}
}

func TestEventBus_Metrics(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	h := newTestHandler()
	bus.Subscribe("metrics.test", h)

	bus.Publish(context.Background(), newTestEvent("metrics.test", ""))
	bus.Publish(context.Background(), newTestEvent("metrics.test", ""))

	eb := bus.(*eventBus)
	publishCount := eb.metrics.publishCount.Load()
	if publishCount < 2 {
		t.Errorf("publishCount = %d, want >= 2", publishCount)
	}

	if eb.metrics.handlerDuration.Load() <= 0 {
		t.Error("handlerDuration should be > 0")
	}
}

func TestEventBus_ConcurrentPublish(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	var count atomic.Int32
	bus.Subscribe("concurrent.test", EventHandlerFunc(func(ctx context.Context, event Event) error {
		count.Add(1)
		return nil
	}))

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(context.Background(), newTestEvent("concurrent.test", ""))
		}()
	}
	wg.Wait()

	if int(count.Load()) != 100 {
		t.Errorf("handler called %d times, want 100", count.Load())
	}
}

func TestEventBus_ConcurrentSubscribe(t *testing.T) {
	bus := NewEventBus()
	defer bus.Close()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Subscribe("concurrent.sub", newTestHandler())
		}()
	}
	wg.Wait()

	if !bus.HasSubscribers("concurrent.sub") {
		t.Error("HasSubscribers() = false after concurrent subscribes")
	}
}
