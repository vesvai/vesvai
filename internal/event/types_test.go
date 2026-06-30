package event

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEventType_String(t *testing.T) {
	tests := []struct {
		name     string
		eventTyp EventType
		want     string
	}{
		{"simple type", EventType("user.created"), "user.created"},
		{"empty type", EventType(""), ""},
		{"special chars", EventType("order/paid@v2"), "order/paid@v2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.eventTyp.String(); got != tt.want {
				t.Errorf("EventType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewBaseEvent(t *testing.T) {
	before := time.Now()
	e := NewBaseEvent("test.event")
	after := time.Now()

	if e.Type() != "test.event" {
		t.Errorf("Type() = %q, want %q", e.Type(), "test.event")
	}

	ts := e.Timestamp()
	if ts.Before(before) || ts.After(after) {
		t.Errorf("Timestamp() = %v, not between %v and %v", ts, before, after)
	}
}

func TestBaseEvent_Type(t *testing.T) {
	e := BaseEvent{eventType: "my.type"}
	if got := e.Type(); got != "my.type" {
		t.Errorf("Type() = %q, want %q", got, "my.type")
	}
}

func TestBaseEvent_Timestamp(t *testing.T) {
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	e := BaseEvent{timestamp: ts}
	if got := e.Timestamp(); got != ts {
		t.Errorf("Timestamp() = %v, want %v", got, ts)
	}
}

func TestEventHandlerFunc_Handle(t *testing.T) {
	var called bool
	var receivedEvent Event
	var receivedCtx context.Context

	f := EventHandlerFunc(func(ctx context.Context, event Event) error {
		called = true
		receivedCtx = ctx
		receivedEvent = event
		return nil
	})

	ctx := context.Background()
	e := newTestEvent("test", "data")

	err := f.Handle(ctx, e)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if !called {
		t.Error("handler function was not called")
	}
	if receivedCtx != ctx {
		t.Error("context was not passed correctly")
	}
	if receivedEvent != e {
		t.Error("event was not passed correctly")
	}
}

func TestEventHandlerFunc_Handle_ReturnsError(t *testing.T) {
	wantErr := errors.New("handler error")
	f := EventHandlerFunc(func(ctx context.Context, event Event) error {
		return wantErr
	})

	err := f.Handle(context.Background(), newTestEvent("test", ""))
	if !errors.Is(err, wantErr) {
		t.Errorf("Handle() error = %v, want %v", err, wantErr)
	}
}

func TestSubscription_Active(t *testing.T) {
	sub := &Subscription{ID: uuid.New().String(), active: true}
	if !sub.Active() {
		t.Error("Active() = false, want true")
	}

	sub.Deactive()
	if sub.Active() {
		t.Error("Active() = true after Deactive(), want false")
	}
}

func TestSubscription_Deactive_Concurrent(t *testing.T) {
	sub := &Subscription{ID: uuid.New().String(), active: true}
	var wg sync.WaitGroup

	// Concurrent reads while deactivating
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = sub.Active()
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		sub.Deactive()
	}()

	wg.Wait()
	if sub.Active() {
		t.Error("Active() = true after concurrent Deactive()")
	}
}

func TestWithPriority(t *testing.T) {
	opts := &SubscriptionOptions{}
	WithPriority(PriorityHigh)(opts)
	if opts.Priority != PriorityHigh {
		t.Errorf("Priority = %v, want %v", opts.Priority, PriorityHigh)
	}
}

func TestWithFilter(t *testing.T) {
	opts := &SubscriptionOptions{}
	filter := func(e Event) bool { return true }
	WithFilter(filter)(opts)
	if opts.Filter == nil {
		t.Error("Filter was not set")
	}
}

func TestWithOnce(t *testing.T) {
	opts := &SubscriptionOptions{}
	WithOnce(true)(opts)
	if !opts.Once {
		t.Error("Once = false, want true")
	}
}
