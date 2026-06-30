package event

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewScopedEventBus(t *testing.T) {
	parent := NewEventBus()
	defer parent.Close()

	scoped := NewScopedEventBus(parent, "scope-1")
	if scoped == nil {
		t.Fatal("NewScopedEventBus() returned nil")
	}
	if scoped.prefix != "scope-1" {
		t.Errorf("prefix = %q, want %q", scoped.prefix, "scope-1")
	}
}

func TestNewScopedEventBus_NonEventBus(t *testing.T) {
	// Passing non-*eventBus should create a fallback
	scoped := NewScopedEventBus(nil, "fallback")
	if scoped.parent == nil {
		t.Error("parent should not be nil even with nil input")
	}
}

func TestScopedEventBus_Publish(t *testing.T) {
	parent := NewEventBus()
	defer parent.Close()

	scoped := NewScopedEventBus(parent, "s1")

	h := newTestHandler()
	scoped.Subscribe("scoped.event", h)

	scoped.Publish(context.Background(), newTestEvent("scoped.event", "data"))

	if h.CallCount() != 1 {
		t.Errorf("handler called %d times, want 1", h.CallCount())
	}
}

func TestScopedEventBus_Publish_Filtered(t *testing.T) {
	parent := NewEventBus()
	defer parent.Close()

	scoped := NewScopedEventBus(parent, "filtered",
		EventType("allowed.event"),
	)

	h := newTestHandler()
	parent.Subscribe("allowed.event", h)
	parent.Subscribe("denied.event", h)

	// Should pass filter
	scoped.Publish(context.Background(), newTestEvent("allowed.event", ""))
	// Should be filtered out
	scoped.Publish(context.Background(), newTestEvent("denied.event", ""))

	if h.CallCount() != 1 {
		t.Errorf("handler called %d times, want 1 (only allowed event)", h.CallCount())
	}
}

func TestScopedEventBus_Publish_Intercepted(t *testing.T) {
	parent := NewEventBus()
	defer parent.Close()

	scoped := NewScopedEventBus(parent, "intercepted")

	intercepted := make(chan Event, 1)
	scoped.Intercept("intercepted.event", EventHandlerFunc(func(ctx context.Context, event Event) error {
		intercepted <- event
		return nil
	}))

	// Parent should NOT receive this event
	parentH := newTestHandler()
	parent.Subscribe("intercepted.event", parentH)

	scoped.Publish(context.Background(), newTestEvent("intercepted.event", ""))

	select {
	case e := <-intercepted:
		if e.Type() != "intercepted.event" {
			t.Errorf("intercepted event type = %q", e.Type())
		}
	case <-time.After(time.Second):
		t.Error("intercepted handler not called")
	}

	if parentH.CallCount() != 0 {
		t.Error("parent handler should not be called when intercepted")
	}
}

func TestScopedEventBus_PublishAsync(t *testing.T) {
	parent := NewEventBus()
	defer parent.Close()

	scoped := NewScopedEventBus(parent, "async")

	h := newTestHandler()
	scoped.Subscribe("async.scoped", h)

	scoped.PublishAsync(context.Background(), newTestEvent("async.scoped", ""))

	select {
	case <-h.calledC:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for async handler")
	}
}

func TestScopedEventBus_PublishAsync_Filtered(t *testing.T) {
	parent := NewEventBus()
	defer parent.Close()

	scoped := NewScopedEventBus(parent, "async.filtered",
		EventType("allowed"),
	)

	h := newTestHandler()
	parent.Subscribe("denied", h)

	scoped.PublishAsync(context.Background(), newTestEvent("denied", ""))

	time.Sleep(100 * time.Millisecond)
	if h.CallCount() != 0 {
		t.Error("filtered async event should not reach parent")
	}
}

func TestScopedEventBus_Subscribe(t *testing.T) {
	parent := NewEventBus()
	defer parent.Close()

	scoped := NewScopedEventBus(parent, "sub")

	h := newTestHandler()
	sub, err := scoped.Subscribe("scoped.sub", h)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	if sub == nil {
		t.Fatal("Subscribe() returned nil subscription")
	}

	scoped.Publish(context.Background(), newTestEvent("scoped.sub", ""))
	if h.CallCount() != 1 {
		t.Errorf("handler called %d times, want 1", h.CallCount())
	}
}

func TestScopedEventBus_Unsubscribe(t *testing.T) {
	parent := NewEventBus()
	defer parent.Close()

	scoped := NewScopedEventBus(parent, "unsub")

	h := newTestHandler()
	sub, _ := scoped.Subscribe("scoped.unsub", h)

	err := scoped.Unsubscribe(sub)
	if err != nil {
		t.Fatalf("Unsubscribe() error = %v", err)
	}

	scoped.Publish(context.Background(), newTestEvent("scoped.unsub", ""))
	if h.CallCount() != 0 {
		t.Error("handler called after unsubscribe")
	}
}

func TestScopedEventBus_Unsubscribe_ParentNotFound(t *testing.T) {
	parent := NewEventBus()
	defer parent.Close()

	scoped := NewScopedEventBus(parent, "unsub.parent")

	fakeSub := &Subscription{
		ID:        uuid.New().String(),
		EventType: "nonexistent",
	}

	err := scoped.Unsubscribe(fakeSub)
	if !errors.Is(err, ErrSubscription) {
		t.Errorf("Unsubscribe() error = %v, want %v", err, ErrSubscription)
	}
}

func TestScopedEventBus_Request_Filtered(t *testing.T) {
	parent := NewEventBus()
	defer parent.Close()

	scoped := NewScopedEventBus(parent, "req.filtered",
		EventType("allowed"),
	)

	_, err := scoped.Request(context.Background(), newTestEvent("denied", ""), time.Second)
	if err == nil {
		t.Error("Request() for filtered event should return error")
	}
}

func TestScopedEventBus_PublishRequest(t *testing.T) {
	parent := NewEventBus()
	defer parent.Close()

	scoped := NewScopedEventBus(parent, "pub.req")

	req := &Request{
		Event:    newTestEvent("scoped.pubreq", ""),
		Response: make(chan Event, 1),
		Error:    make(chan error, 1),
		Timeout:  5 * time.Second,
	}

	err := scoped.PublishRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("PublishRequest() error = %v", err)
	}
}

func TestScopedEventBus_HasSubscribers(t *testing.T) {
	parent := NewEventBus()
	defer parent.Close()

	scoped := NewScopedEventBus(parent, "has")

	if scoped.HasSubscribers("exists") {
		t.Error("HasSubscribers() = true before subscribe")
	}

	scoped.Subscribe("exists", newTestHandler())
	if !scoped.HasSubscribers("exists") {
		t.Error("HasSubscribers() = false after subscribe")
	}
}

func TestScopedEventBus_Close(t *testing.T) {
	parent := NewEventBus()
	defer parent.Close()

	scoped := NewScopedEventBus(parent, "close")
	h := newTestHandler()
	scoped.Subscribe("close.event", h)

	err := scoped.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Parent should not receive events for scoped subscriptions
	scoped.Publish(context.Background(), newTestEvent("close.event", ""))
	if h.CallCount() != 0 {
		t.Error("handler still receives events after scoped Close()")
	}
}

func TestScopedEventBus_RemoveIntercept(t *testing.T) {
	parent := NewEventBus()
	defer parent.Close()

	scoped := NewScopedEventBus(parent, "no.intercept")

	intercepted := make(chan Event, 1)
	scoped.Intercept("intercept.event", EventHandlerFunc(func(ctx context.Context, event Event) error {
		intercepted <- event
		return nil
	}))

	scoped.RemoveIntercept("intercept.event")

	// Now parent should receive it
	parentH := newTestHandler()
	parent.Subscribe("intercept.event", parentH)

	scoped.Publish(context.Background(), newTestEvent("intercept.event", ""))

	select {
	case <-intercepted:
		t.Error("intercepted handler should not be called after RemoveIntercept")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}

	if parentH.CallCount() != 1 {
		t.Error("parent handler should be called after RemoveIntercept")
	}
}

func TestScopedEventBus_AddRemoveFilter(t *testing.T) {
	parent := NewEventBus()
	defer parent.Close()

	scoped := NewScopedEventBus(parent, "filters")

	h := newTestHandler()
	parent.Subscribe("a", h)
	parent.Subscribe("b", h)

	scoped.Publish(context.Background(), newTestEvent("a", ""))
	scoped.Publish(context.Background(), newTestEvent("b", ""))
	if h.CallCount() != 2 {
		t.Errorf("handler called %d times without filters, want 2", h.CallCount())
	}

	scoped.AddFilter("a")
	scoped.Publish(context.Background(), newTestEvent("a", ""))
	scoped.Publish(context.Background(), newTestEvent("b", ""))
	if h.CallCount() != 3 {
		t.Errorf("handler called %d times after AddFilter, want 3 (only 'a' allowed)", h.CallCount())
	}

	scoped.RemoveFilter("a")
	scoped.Publish(context.Background(), newTestEvent("a", ""))
	scoped.Publish(context.Background(), newTestEvent("b", ""))
	if h.CallCount() != 5 {
		t.Errorf("handler called %d times after RemoveFilter, want 5", h.CallCount())
	}
}

func TestScopedEventBus_RemoveFilter_NotFound(t *testing.T) {
	parent := NewEventBus()
	defer parent.Close()

	scoped := NewScopedEventBus(parent, "rm.nf")
	// Should not panic
	scoped.RemoveFilter("nonexistent")
}

func TestScopedEventBus_ClearFilters(t *testing.T) {
	parent := NewEventBus()
	defer parent.Close()

	scoped := NewScopedEventBus(parent, "clear",
		EventType("a"),
		EventType("b"),
	)

	h := newTestHandler()
	parent.Subscribe("a", h)
	parent.Subscribe("b", h)

	scoped.Publish(context.Background(), newTestEvent("a", ""))
	scoped.Publish(context.Background(), newTestEvent("b", ""))
	if h.CallCount() != 2 {
		t.Errorf("handler called %d times with filters, want 2", h.CallCount())
	}

	scoped.Publish(context.Background(), newTestEvent("c", ""))
	if h.CallCount() != 2 {
		t.Errorf("handler called %d times for unfiltered event, want 2", h.CallCount())
	}

	scoped.ClearFilters()
	scoped.Publish(context.Background(), newTestEvent("a", ""))
	scoped.Publish(context.Background(), newTestEvent("b", ""))
	if h.CallCount() != 4 {
		t.Errorf("handler called %d times after ClearFilters, want 4", h.CallCount())
	}
}
