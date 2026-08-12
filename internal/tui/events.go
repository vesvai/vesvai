package tui

import "time"

type StreamEventKind int

const (
	EventStart StreamEventKind = iota
	EventContent
	EventReasoning
	EventToolCall
	EventToolResult
	EventDone
	EventError

	EventSubagentStart
	EventSubagentChunk
	EventSubagentDone
)

type ToolCallInfo struct {
	ToolName   string
	Args       map[string]any
	SubagentID string
}

type ToolResultInfo struct {
	ToolName   string
	Result     string
	Error      error
	Duration   time.Duration
	SubagentID string
}

type SubagentInfo struct {
	Name   string
	Prompt string
}

type SubagentResult struct {
	Name     string
	Result   string
	Error    error
	Duration time.Duration
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type StreamEvent struct {
	Kind      StreamEventKind
	Content   string
	Reasoning string

	SubagentReasoning bool

	ToolCall   *ToolCallInfo
	ToolResult *ToolResultInfo

	Subagent       *SubagentInfo
	SubagentID     string
	SubagentResult *SubagentResult

	Usage *Usage
	Error error
}
