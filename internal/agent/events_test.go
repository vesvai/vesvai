package agent

import (
	"context"
	"testing"

	"github.com/vesvai/vesvai/internal/event"
)

func TestNewAgentEvent(t *testing.T) {
	data := MessageEventData{Role: "user", Content: "hello"}
	evt := NewAgentEvent(EventAgentStart, data)

	if evt.Type() != event.EventType(EventAgentStart) {
		t.Errorf("Type() = %q, want %q", evt.Type(), EventAgentStart)
	}
	if evt.SequenceNum == 0 {
		t.Error("SequenceNum should not be 0")
	}
	if d, ok := evt.Data.(MessageEventData); !ok || d.Role != "user" {
		t.Errorf("Data = %v, want MessageEventData with role=user", evt.Data)
	}
}

func TestNewAgentEvent_SequenceIncremental(t *testing.T) {
	evt1 := NewAgentEvent(EventAgentStart, nil)
	evt2 := NewAgentEvent(EventAgentComplete, nil)

	if evt2.SequenceNum <= evt1.SequenceNum {
		t.Errorf("SequenceNum not incremental: %d <= %d", evt2.SequenceNum, evt1.SequenceNum)
	}
}

func TestMessageEventData_SetMetadata(t *testing.T) {
	d := &MessageEventData{}
	d.SetMetadata("agent-1", "session-1", 5)

	if d.AgentID != "agent-1" {
		t.Errorf("AgentID = %q, want agent-1", d.AgentID)
	}
	if d.SessionID != "session-1" {
		t.Errorf("SessionID = %q, want session-1", d.SessionID)
	}
	if d.Step != 5 {
		t.Errorf("Step = %d, want 5", d.Step)
	}
}

func TestMessageEventData_Getters(t *testing.T) {
	d := &MessageEventData{AgentID: "a1", SessionID: "s1"}
	if d.GetAgentID() != "a1" {
		t.Errorf("GetAgentID() = %q, want a1", d.GetAgentID())
	}
	if d.GetSessionID() != "s1" {
		t.Errorf("GetSessionID() = %q, want s1", d.GetSessionID())
	}
}

func TestToolEventData_SetMetadata(t *testing.T) {
	d := &ToolEventData{}
	d.SetMetadata("agent-1", "session-1", 3)

	if d.AgentID != "agent-1" || d.SessionID != "session-1" || d.Step != 3 {
		t.Errorf("SetMetadata failed: %+v", d)
	}
}

func TestToolEventData_Getters(t *testing.T) {
	d := &ToolEventData{AgentID: "a1", SessionID: "s1"}
	if d.GetAgentID() != "a1" || d.GetSessionID() != "s1" {
		t.Error("Getters failed")
	}
}

func TestTaskEventData_SetMetadata(t *testing.T) {
	d := &TaskEventData{}
	d.SetMetadata("agent-1", "session-1", 0)

	if d.AgentID != "agent-1" || d.SessionID != "session-1" {
		t.Errorf("SetMetadata failed: %+v", d)
	}
}

func TestTaskEventData_Getters(t *testing.T) {
	d := &TaskEventData{AgentID: "a1", SessionID: "s1"}
	if d.GetAgentID() != "a1" || d.GetSessionID() != "s1" {
		t.Error("Getters failed")
	}
}

func TestAgentInitEventData_SetMetadata(t *testing.T) {
	d := &AgentInitEventData{}
	d.SetMetadata("agent-1", "session-1", 0)

	if d.AgentID != "agent-1" || d.SessionID != "session-1" {
		t.Errorf("SetMetadata failed: %+v", d)
	}
}

func TestAgentInitEventData_Getters(t *testing.T) {
	d := &AgentInitEventData{AgentID: "a1", SessionID: "s1"}
	if d.GetAgentID() != "a1" || d.GetSessionID() != "s1" {
		t.Error("Getters failed")
	}
}

func TestAgentErrorEventData_SetMetadata(t *testing.T) {
	d := &AgentErrorEventData{}
	d.SetMetadata("agent-1", "session-1", 0)

	if d.AgentID != "agent-1" || d.SessionID != "session-1" {
		t.Errorf("SetMetadata failed: %+v", d)
	}
}

func TestAgentErrorEventData_Getters(t *testing.T) {
	d := &AgentErrorEventData{AgentID: "a1", SessionID: "s1"}
	if d.GetAgentID() != "a1" || d.GetSessionID() != "s1" {
		t.Error("Getters failed")
	}
}

func TestFilterByAgentID(t *testing.T) {
	filter := FilterByAgentID("target")

	evt := NewAgentEvent(EventAgentStart, &MessageEventData{AgentID: "target"})
	if !filter(evt) {
		t.Error("FilterByAgentID should match target agent")
	}

	evt2 := NewAgentEvent(EventAgentStart, &MessageEventData{AgentID: "other"})
	if filter(evt2) {
		t.Error("FilterByAgentID should not match other agent")
	}
}

func TestFilterByAgentID_NonAgentEvent(t *testing.T) {
	filter := FilterByAgentID("target")

	plain := event.NewBaseEvent("test")
	if filter(&plain) {
		t.Error("FilterByAgentID should reject non-AgentEvent")
	}
}

func TestFilterByAgentID_NoMetadata(t *testing.T) {
	filter := FilterByAgentID("target")

	evt := NewAgentEvent(EventAgentStart, "plain string data")
	if filter(evt) {
		t.Error("FilterByAgentID should reject data without GetAgentID")
	}
}

func TestFilterBySessionID(t *testing.T) {
	filter := FilterBySessionID("sess-1")

	evt := NewAgentEvent(EventAgentStart, &MessageEventData{SessionID: "sess-1"})
	if !filter(evt) {
		t.Error("FilterBySessionID should match")
	}

	evt2 := NewAgentEvent(EventAgentStart, &MessageEventData{SessionID: "other"})
	if filter(evt2) {
		t.Error("FilterBySessionID should not match")
	}
}

func TestFilterBySessionID_NonAgentEvent(t *testing.T) {
	filter := FilterBySessionID("sess-1")

	plain := event.NewBaseEvent("test")
	if filter(&plain) {
		t.Error("FilterBySessionID should reject non-AgentEvent")
	}
}

func TestFilterBySessionID_NoMetadata(t *testing.T) {
	filter := FilterBySessionID("sess-1")

	evt := NewAgentEvent(EventAgentStart, "no metadata")
	if filter(evt) {
		t.Error("FilterBySessionID should reject data without GetSessionID")
	}
}

func TestFilterByAgentAndSession(t *testing.T) {
	filter := FilterByAgentAndSession("agent-1", "sess-1")

	match := NewAgentEvent(EventAgentStart, &MessageEventData{AgentID: "agent-1", SessionID: "sess-1"})
	if !filter(match) {
		t.Error("FilterByAgentAndSession should match")
	}

	noAgent := NewAgentEvent(EventAgentStart, &MessageEventData{AgentID: "other", SessionID: "sess-1"})
	if filter(noAgent) {
		t.Error("FilterByAgentAndSession should reject wrong agent")
	}

	noSession := NewAgentEvent(EventAgentStart, &MessageEventData{AgentID: "agent-1", SessionID: "other"})
	if filter(noSession) {
		t.Error("FilterByAgentAndSession should reject wrong session")
	}
}

func TestFilterByAgentAndSession_NonAgentEvent(t *testing.T) {
	filter := FilterByAgentAndSession("a", "s")
	plain := event.NewBaseEvent("test")
	if filter(&plain) {
		t.Error("should reject non-AgentEvent")
	}
}

func TestFilterByAgentAndSession_NoMetadata(t *testing.T) {
	filter := FilterByAgentAndSession("a", "s")
	evt := NewAgentEvent(EventAgentStart, "raw")
	if filter(evt) {
		t.Error("should reject data without metadata interface")
	}
}

func TestSubscriptionManager_Subscribe(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	mgr := NewSubscriptionManager(bus)

	sub, err := mgr.Subscribe(EventAgentStart, event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
		return nil
	}))

	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	if sub == nil {
		t.Fatal("Subscribe() returned nil subscription")
	}
	if !bus.HasSubscribers(event.EventType(EventAgentStart)) {
		t.Error("HasSubscribers() = false after subscribe")
	}
}

func TestSubscriptionManager_CancelAll(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	mgr := NewSubscriptionManager(bus)

	mgr.Subscribe(EventAgentStart, event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
		return nil
	}))
	mgr.Subscribe(EventAgentComplete, event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
		return nil
	}))

	mgr.CancelAll()

	if bus.HasSubscribers(event.EventType(EventAgentStart)) {
		t.Error("HasSubscribers() = true after CancelAll")
	}
}

func TestSubscriptionManager_Subscribe_Error(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	bus.Close()

	mgr := NewSubscriptionManager(bus)
	_, err := mgr.Subscribe(EventAgentStart, event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
		return nil
	}))

	if err == nil {
		t.Error("Subscribe() should error on closed bus")
	}
}

func TestEventTypes(t *testing.T) {
	types := []AgentEventType{
		EventAgentInit,
		EventAgentReady,
		EventAgentStart,
		EventAgentComplete,
		EventAgentError,
		EventAgentShutdown,
		EventAgentMessageReceived,
		EventAgentMessageSent,
		EventAgentToolCall,
		EventAgentToolResult,
		EventAgentTaskStart,
		EventAgentTaskComplete,
		EventAgentTaskFailed,
	}

	seen := make(map[AgentEventType]bool)
	for _, et := range types {
		if seen[et] {
			t.Errorf("duplicate event type: %s", et)
		}
		seen[et] = true

		if et == "" {
			t.Error("empty event type")
		}
	}
}
