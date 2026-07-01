package event

import (
	"reflect"
	"testing"
)

func TestDeadEvent(t *testing.T) {
	original := newTestEvent("original.type", "data")
	de := NewDeadEvent(original)

	if de.Type() != "dead_event" {
		t.Errorf("Type() = %q, want %q", de.Type(), "dead_event")
	}

	if de.Event != original {
		t.Error("Event field not set correctly")
	}

	if len(de.Subscribers()) != 0 {
		t.Errorf("Subscribers() len = %d, want 0", len(de.Subscribers()))
	}
}

func TestDeadEvent_AddSubscriber(t *testing.T) {
	de := NewDeadEvent(newTestEvent("test", ""))
	de.AddSubscriber("sub1")
	de.AddSubscriber("sub2")

	subs := de.Subscribers()
	if len(subs) != 2 {
		t.Fatalf("Subscribers() len = %d, want 2", len(subs))
	}
	if subs[0] != "sub1" || subs[1] != "sub2" {
		t.Errorf("Subscribers() = %v, want [sub1 sub2]", subs)
	}
}

func TestDeadEvent_Subscribers(t *testing.T) {
	de := NewDeadEvent(newTestEvent("test", ""))
	de.AddSubscriber("sub1")

	subs := de.Subscribers()
	if len(subs) != 1 {
		t.Errorf("Subscribers() len = %d, want 1", len(subs))
	}
	if subs[0] != "sub1" {
		t.Errorf("Subscribers()[0] = %q, want %q", subs[0], "sub1")
	}
}

func TestDeadEvent_Timestamp(t *testing.T) {
	de := NewDeadEvent(newTestEvent("test", ""))
	if de.Timestamp().IsZero() {
		t.Error("Timestamp() should not be zero")
	}
}

func TestNewSystemEvent(t *testing.T) {
	data := map[string]string{"key": "value"}
	se := NewSystemEvent(EventSystemInit, data)

	if string(se.Type()) != "system:init" {
		t.Errorf("Type() = %q, want %q", se.Type(), "system:init")
	}

	if !reflect.DeepEqual(se.Data, data) {
		t.Error("Data field not set correctly")
	}
}

func TestSystemEvent_Types(t *testing.T) {
	tests := []struct {
		name     string
		eventType SystemEventType
		wantType string
	}{
		{"init", EventSystemInit, "system:init"},
		{"ready", EventSystemReady, "system:ready"},
		{"shutdown", EventSystemShutdown, "system:shutdown"},
		{"error", EventSystemError, "system:error"},
		{"config", EventSystemConfig, "system:config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := NewSystemEvent(tt.eventType, nil)
			if string(se.Type()) != tt.wantType {
				t.Errorf("Type() = %q, want %q", se.Type(), tt.wantType)
			}
		})
	}
}

func TestSystemEvent_NilData(t *testing.T) {
	se := NewSystemEvent(EventSystemReady, nil)
	if se.Data != nil {
		t.Errorf("Data = %v, want nil", se.Data)
	}
}

func TestSystemEvent_Timestamp(t *testing.T) {
	se := NewSystemEvent(EventSystemInit, nil)
	if se.Timestamp().IsZero() {
		t.Error("Timestamp() should not be zero")
	}
}
