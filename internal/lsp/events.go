package lsp

import (
	"sync/atomic"

	"github.com/vesvai/vesvai/internal/event"
)

var globalSequence atomic.Int64

type LSPEventType event.EventType

const (
	EventLSPServerStart LSPEventType = "lsp:server:start"
	EventLSPServerStop  LSPEventType = "lsp:server:stop"
	EventLSPServerError LSPEventType = "lsp:server:error"
	EventLSPDiagPublish LSPEventType = "lsp:diagnostics:publish"
)

type LSPEvent struct {
	event.BaseEvent
	SequenceNum int64
	Data        any
}

func NewLSPEvent(eventType LSPEventType, data any) *LSPEvent {
	return &LSPEvent{
		BaseEvent:   event.NewBaseEvent(event.EventType(eventType)),
		SequenceNum: globalSequence.Add(1),
		Data:        data,
	}
}

type ServerStartEventData struct {
	ServerName string
	ServerInfo ServerInfo
	Languages  []string
}

type ServerStopEventData struct {
	ServerName string
}

type ServerErrorEventData struct {
	ServerName string
	Error      error
}

type DiagnosticsEventData struct {
	ServerName  string
	URI         string
	Diagnostics []Diagnostic
}
