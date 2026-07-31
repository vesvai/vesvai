package mcp

import (
	"sync/atomic"

	"github.com/vesvai/vesvai/internal/event"
)

var globalSequence atomic.Int64

type MCPEventType event.EventType

const (
	EventMCPConnect    MCPEventType = "mcp:connect"
	EventMCPDisconnect MCPEventType = "mcp:disconnect"
	EventMCPError      MCPEventType = "mcp:error"

	EventMCPToolsList  MCPEventType = "mcp:tools:list"
	EventMCPToolCall   MCPEventType = "mcp:tool:call"
	EventMCPToolResult MCPEventType = "mcp:tool:result"

	EventMCPResourcesList MCPEventType = "mcp:resources:list"
	EventMCPResourceRead  MCPEventType = "mcp:resource:read"

	EventMCPPromptsList MCPEventType = "mcp:prompts:list"
	EventMCPPromptGet   MCPEventType = "mcp:prompt:get"
)

type MCPEvent struct {
	event.BaseEvent
	SequenceNum int64
	Data        any
}

func NewMCPEvent(eventType MCPEventType, data any) *MCPEvent {
	return &MCPEvent{
		BaseEvent:   event.NewBaseEvent(event.EventType(eventType)),
		SequenceNum: globalSequence.Add(1),
		Data:        data,
	}
}

type ConnectEventData struct {
	ServerName    string
	ServerVersion string
	Instructions  string
}

func (d ConnectEventData) GetServerName() string { return d.ServerName }

type DisconnectEventData struct {
	ServerName string
}

func (d DisconnectEventData) GetServerName() string { return d.ServerName }

type ErrorEventData struct {
	ServerName string
	Method     string
	Error      error
}

func (d ErrorEventData) GetServerName() string { return d.ServerName }

type ToolsListEventData struct {
	ServerName string
	ToolCount  int
}

func (d ToolsListEventData) GetServerName() string { return d.ServerName }

type ToolCallEventData struct {
	ServerName string
	ToolName   string
	Arguments  map[string]any
}

func (d ToolCallEventData) GetServerName() string { return d.ServerName }

type ToolResultEventData struct {
	ServerName string
	ToolName   string
	IsError    bool
}

func (d ToolResultEventData) GetServerName() string { return d.ServerName }

type ResourcesListEventData struct {
	ServerName    string
	ResourceCount int
}

func (d ResourcesListEventData) GetServerName() string { return d.ServerName }

type ResourceReadEventData struct {
	ServerName string
	URI        string
}

func (d ResourceReadEventData) GetServerName() string { return d.ServerName }

type PromptsListEventData struct {
	ServerName  string
	PromptCount int
}

func (d PromptsListEventData) GetServerName() string { return d.ServerName }

type PromptGetEventData struct {
	ServerName string
	PromptName string
}

func (d PromptGetEventData) GetServerName() string { return d.ServerName }

func FilterByServerName(serverName string) event.FilterFunc {
	return func(e event.Event) bool {
		mcpEvent, ok := e.(*MCPEvent)
		if !ok {
			return false
		}
		if data, ok := mcpEvent.Data.(interface{ GetServerName() string }); ok {
			return data.GetServerName() == serverName
		}
		return false
	}
}
