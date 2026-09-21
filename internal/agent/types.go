package agent

import (
	"github.com/vesvai/vesvai/internal/agent/middleware"
	"github.com/vesvai/vesvai/internal/llm"
)

type RunResult = middleware.Result

type StreamEventType string

const (
	StreamToken      StreamEventType = "token"
	StreamReasoning  StreamEventType = "reasoning"
	StreamToolCall   StreamEventType = "tool_call"
	StreamToolResult StreamEventType = "tool_result"
	StreamCompaction StreamEventType = "compaction"
	StreamDone       StreamEventType = "done"
)

type StreamEvent struct {
	Type       StreamEventType
	AgentID    string
	AgentName  string
	Model      llm.Model
	Content    string
	Reasoning  string
	ToolCall   *llm.ToolCall
	ToolOutput string
	ToolErr    error
	Usage      *llm.Usage
	Strategy   string
	Messages   int
	Tokens     int
}

type StreamHandler func(StreamEvent) error
