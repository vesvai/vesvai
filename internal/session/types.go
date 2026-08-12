package session

import (
	"time"

	"github.com/vesvai/vesvai/internal/llm"
)

type Session struct {
	ID        string          `json:"id"`
	Title     string          `json:"title,omitempty"`
	Messages  []llm.Message   `json:"messages"`
	Metadata  SessionMetadata `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`

	// Parts preserves the display order of each assistant message (interleaved
	// thinking/tools/subagents) plus full subagent transcripts, which the
	// protocol-correct Messages list cannot represent. Absent for sessions
	// saved before this field existed.
	Parts []MessageParts `json:"parts,omitempty"`
}

// PartKind identifies a display segment of an assistant message.
type PartKind string

const (
	PartThinking PartKind = "thinking"
	PartContent  PartKind = "content"
	PartTool     PartKind = "tool"
	PartSubagent PartKind = "subagent"
)

// MessageParts is the ordered display metadata of one assistant message.
type MessageParts struct {
	Parts []PartRecord `json:"parts,omitempty"`
}

// PartRecord describes one ordered segment. Thinking/content parts record the
// length (in runes) of their text within the message's accumulated
// Thinking/Content; tool parts index the message's ToolCalls.
type PartRecord struct {
	Kind     PartKind `json:"kind"`
	TextLen  int      `json:"text_len,omitempty"`
	ToolIdx  int      `json:"tool_idx,omitempty"`
	Subagent *SubagentRecord `json:"subagent,omitempty"`
}

// SubagentRecord is a full sub-agent run (transcript, tools, outcome).
type SubagentRecord struct {
	Name       string             `json:"name"`
	Prompt     string             `json:"prompt,omitempty"`
	Thinking   string             `json:"thinking,omitempty"`
	Content    string             `json:"content,omitempty"`
	Result     string             `json:"result,omitempty"`
	Error      string             `json:"error,omitempty"`
	DurationMs int64              `json:"duration_ms,omitempty"`
	Tools      []SubagentToolRecord `json:"tools,omitempty"`
	Parts      []PartRecord       `json:"parts,omitempty"`
}

// SubagentToolRecord is one tool invocation of a sub-agent.
type SubagentToolRecord struct {
	Name       string         `json:"name"`
	Args       map[string]any `json:"args,omitempty"`
	Result     string         `json:"result,omitempty"`
	Error      string         `json:"error,omitempty"`
	DurationMs int64          `json:"duration_ms,omitempty"`
}

type SessionMetadata struct {
	Model         string  `json:"model,omitempty"`
	Provider      string  `json:"provider,omitempty"`
	Temperature   float64 `json:"temperature,omitempty"`
	MaxSteps      int     `json:"max_steps,omitempty"`
	Agent         string  `json:"agent,omitempty"`
	MessageCount  int     `json:"message_count"`
	TotalTokens   int     `json:"total_tokens,omitempty"`
	WorkspacePath string  `json:"workspace_path,omitempty"`
}

type SessionMetadataIndex struct {
	ID           string    `json:"id"`
	Title        string    `json:"title,omitempty"`
	MessageCount int       `json:"message_count"`
	Model        string    `json:"model,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ListOptions struct {
	Page     int
	PageSize int
	SortBy   string
	Reverse  bool
}

type ListResult struct {
	Sessions   []SessionMetadataIndex `json:"sessions"`
	Total      int                    `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
	TotalPages int                    `json:"total_pages"`
}

type Store interface {
	Save(session *Session) error
	Load(id string) (*Session, error)
	Delete(id string) error
	List(opts ListOptions) (*ListResult, error)
	Exists(id string) bool
	Close() error
}
