package agent

import (
	"sync"
	"sync/atomic"

	"github.com/vesvai/vesvai/internal/event"
)

var globalSequence atomic.Int64

type AgentEventType event.EventType

const (
	EventAgentInit     AgentEventType = "agent:init"
	EventAgentReady    AgentEventType = "agent:ready"
	EventAgentStart    AgentEventType = "agent:start"
	EventAgentComplete AgentEventType = "agent:complete"
	EventAgentError    AgentEventType = "agent:error"
	EventAgentShutdown AgentEventType = "agent:shutdown"

	EventAgentMessageReceived AgentEventType = "agent:message:received"
	EventAgentMessageSent     AgentEventType = "agent:message:sent"

	EventAgentToolCall   AgentEventType = "agent:tool:call"
	EventAgentToolResult AgentEventType = "agent:tool:result"

	EventAgentTaskStart    AgentEventType = "agent:task:start"
	EventAgentTaskComplete AgentEventType = "agent:task:complete"
	EventAgentTaskFailed   AgentEventType = "agent:task:failed"
)

type AgentEvent struct {
	event.BaseEvent
	SequenceNum int64
	Data        interface{}
}

func NewAgentEvent(eventType AgentEventType, data interface{}) *AgentEvent {
	return &AgentEvent{
		BaseEvent:   event.NewBaseEvent(event.EventType(eventType)),
		SequenceNum: globalSequence.Add(1),
		Data:        data,
	}
}

type MessageEventData struct {
	Role      string
	Content   string
	AgentID   string
	SessionID string
	MessageID string
	Step      int
}

func (d *MessageEventData) SetMetadata(agentID, sessionID string, step int) {
	d.AgentID = agentID
	d.SessionID = sessionID
	d.Step = step
}

func (d *MessageEventData) GetAgentID() string   { return d.AgentID }
func (d *MessageEventData) GetSessionID() string { return d.SessionID }

type ToolEventData struct {
	ToolName  string
	AgentID   string
	SessionID string
	Args      interface{}
	Result    interface{}
	Error     error
	Duration  int64
	Step      int
}

func (d *ToolEventData) SetMetadata(agentID, sessionID string, step int) {
	d.AgentID = agentID
	d.SessionID = sessionID
	d.Step = step
}

func (d *ToolEventData) GetAgentID() string   { return d.AgentID }
func (d *ToolEventData) GetSessionID() string { return d.SessionID }

type TaskEventData struct {
	TaskID    string
	AgentID   string
	SessionID string
	Status    string
	Error     error
	Result    interface{}
}

func (d *TaskEventData) SetMetadata(agentID, sessionID string, step int) {
	d.AgentID = agentID
	d.SessionID = sessionID
}

func (d *TaskEventData) GetAgentID() string   { return d.AgentID }
func (d *TaskEventData) GetSessionID() string { return d.SessionID }

type AgentInitEventData struct {
	AgentID   string
	AgentType string
	SessionID string
	Config    interface{}
}

func (d *AgentInitEventData) SetMetadata(agentID, sessionID string, step int) {
	d.AgentID = agentID
	d.SessionID = sessionID
}

func (d *AgentInitEventData) GetAgentID() string   { return d.AgentID }
func (d *AgentInitEventData) GetSessionID() string { return d.SessionID }

type AgentErrorEventData struct {
	AgentID   string
	SessionID string
	Error     error
	Context   interface{}
}

func (d *AgentErrorEventData) SetMetadata(agentID, sessionID string, step int) {
	d.AgentID = agentID
	d.SessionID = sessionID
}

func (d *AgentErrorEventData) GetAgentID() string   { return d.AgentID }
func (d *AgentErrorEventData) GetSessionID() string { return d.SessionID }

func FilterByAgentID(agentID string) event.FilterFunc {
	return func(e event.Event) bool {
		ae, ok := e.(*AgentEvent)
		if !ok {
			return false
		}
		if m, ok := ae.Data.(interface{ GetAgentID() string }); ok {
			return m.GetAgentID() == agentID
		}
		return false
	}
}

func FilterBySessionID(sessionID string) event.FilterFunc {
	return func(e event.Event) bool {
		ae, ok := e.(*AgentEvent)
		if !ok {
			return false
		}
		if m, ok := ae.Data.(interface{ GetSessionID() string }); ok {
			return m.GetSessionID() == sessionID
		}
		return false
	}
}

func FilterByAgentAndSession(agentID, sessionID string) event.FilterFunc {
	return func(e event.Event) bool {
		ae, ok := e.(*AgentEvent)
		if !ok {
			return false
		}
		if m, ok := ae.Data.(interface {
			GetAgentID() string
			GetSessionID() string
		}); ok {
			return m.GetAgentID() == agentID && m.GetSessionID() == sessionID
		}
		return false
	}
}

type SubscriptionManager struct {
	bus  event.EventBus
	subs []*event.Subscription
	mu   sync.Mutex
}

func NewSubscriptionManager(bus event.EventBus) *SubscriptionManager {
	return &SubscriptionManager{bus: bus}
}

func (m *SubscriptionManager) Subscribe(eventType AgentEventType, handler event.Handler, opts ...event.SubscriptionOption) (*event.Subscription, error) {
	sub, err := m.bus.Subscribe(event.EventType(eventType), handler, opts...)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.subs = append(m.subs, sub)
	m.mu.Unlock()
	return sub, nil
}

func (m *SubscriptionManager) CancelAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, sub := range m.subs {
		m.bus.Unsubscribe(sub)
	}
	m.subs = m.subs[:0]
}
