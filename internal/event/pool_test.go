package event

import (
	"testing"
	"time"
)

func TestGetDeadEvent(t *testing.T) {
	de := GetDeadEvent()
	if de == nil {
		t.Fatal("GetDeadEvent() returned nil")
	}

	if de.Timestamp().IsZero() {
		t.Error("GetDeadEvent() timestamp should not be zero")
	}
}

func TestPutDeadEvent(t *testing.T) {
	de := GetDeadEvent()
	de.Event = newTestEvent("test", "data")
	de.AddSubscriber("sub1")

	PutDeadEvent(de)

	// Get another one - should be recycled
	de2 := GetDeadEvent()
	if len(de2.Subscribers()) != 0 {
		t.Error("recycled DeadEvent should have empty subscribers")
	}
	if de2.Event != nil {
		t.Error("recycled DeadEvent should have nil Event")
	}
}

func TestDeadEvent_Pool_Reuse(t *testing.T) {
	de := GetDeadEvent()
	de.Event = newTestEvent("test", "")
	de.AddSubscriber("s1")
	de.AddSubscriber("s2")

	PutDeadEvent(de)

	de2 := GetDeadEvent()
	if de2 == nil {
		t.Fatal("GetDeadEvent() returned nil after recycle")
	}
	// Should have fresh timestamp
	if de2.Timestamp().Before(time.Now().Add(-time.Second)) {
		t.Error("recycled DeadEvent should have fresh timestamp")
	}
}
