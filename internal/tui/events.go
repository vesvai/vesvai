package tui

import (
	"context"
	"fmt"
	"sync"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/event"
)

type TUIEventType string

const (
	TUIEventThinking TUIEventType = "tui:thinking"
	TUIEventContent  TUIEventType = "tui:content"
	TUIEventToolCall TUIEventType = "tui:tool:call"
	TUIEventToolDone TUIEventType = "tui:tool:done"
	TUIEventDone     TUIEventType = "tui:done"
	TUIEventError    TUIEventType = "tui:error"
)

type TUIEvent interface {
	Type() TUIEventType
}

type TUIThinkingEvent struct {
	Content string
}

func (e TUIThinkingEvent) Type() TUIEventType { return TUIEventThinking }

type TUIContentEvent struct {
	Content string
}

func (e TUIContentEvent) Type() TUIEventType { return TUIEventContent }

type TUIToolCallEvent struct {
	ToolName string
	Args     map[string]any
}

func (e TUIToolCallEvent) Type() TUIEventType { return TUIEventToolCall }

type TUIToolDoneEvent struct {
	ToolName string
	Result   string
	Error    error
	Duration int64
}

func (e TUIToolDoneEvent) Type() TUIEventType { return TUIEventToolDone }

type TUIDoneEvent struct {
	Content string
}

func (e TUIDoneEvent) Type() TUIEventType { return TUIEventDone }

type TUIErrorEvent struct {
	Error error
}

func (e TUIErrorEvent) Type() TUIEventType { return TUIEventError }

type TUIEventHandler func(event TUIEvent)

type TUIEventAdapter struct {
	bus     event.EventBus
	handler TUIEventHandler
	mu      sync.Mutex
	subs    []*event.Subscription
}

func NewTUIEventAdapter(bus event.EventBus) *TUIEventAdapter {
	return &TUIEventAdapter{
		bus: bus,
	}
}

func (a *TUIEventAdapter) SetHandler(handler TUIEventHandler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.handler = handler
}

func (a *TUIEventAdapter) Start() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	toolCallSub, err := a.bus.Subscribe(event.EventType(agent.EventAgentToolCall), event.EventHandlerFunc(a.handleToolCall))
	if err != nil {
		return err
	}
	a.subs = append(a.subs, toolCallSub)

	toolResultSub, err := a.bus.Subscribe(event.EventType(agent.EventAgentToolResult), event.EventHandlerFunc(a.handleToolResult))
	if err != nil {
		return err
	}
	a.subs = append(a.subs, toolResultSub)

	completeSub, err := a.bus.Subscribe(event.EventType(agent.EventAgentComplete), event.EventHandlerFunc(a.handleComplete))
	if err != nil {
		return err
	}
	a.subs = append(a.subs, completeSub)

	return nil
}

func (a *TUIEventAdapter) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, sub := range a.subs {
		a.bus.Unsubscribe(sub)
	}
	a.subs = nil
}

func (a *TUIEventAdapter) handleToolCall(ctx context.Context, e event.Event) error {
	ae, ok := e.(*agent.AgentEvent)
	if !ok {
		return nil
	}
	data, ok := ae.Data.(*agent.ToolEventData)
	if !ok {
		return nil
	}
	args, _ := data.Args.(map[string]any)
	a.emit(TUIToolCallEvent{
		ToolName: data.ToolName,
		Args:     args,
	})
	return nil
}

func (a *TUIEventAdapter) handleToolResult(ctx context.Context, e event.Event) error {
	ae, ok := e.(*agent.AgentEvent)
	if !ok {
		return nil
	}
	data, ok := ae.Data.(*agent.ToolEventData)
	if !ok {
		return nil
	}
	a.emit(TUIToolDoneEvent{
		ToolName: data.ToolName,
		Result:   formatResult(data.Result),
		Error:    data.Error,
		Duration: data.Duration,
	})
	return nil
}

func (a *TUIEventAdapter) handleComplete(ctx context.Context, e event.Event) error {
	a.emit(TUIDoneEvent{})
	return nil
}

func (a *TUIEventAdapter) emit(event TUIEvent) {
	a.mu.Lock()
	handler := a.handler
	a.mu.Unlock()
	if handler != nil {
		handler(event)
	}
}

func formatResult(result interface{}) string {
	if result == nil {
		return ""
	}
	s := fmt.Sprintf("%v", result)
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}
