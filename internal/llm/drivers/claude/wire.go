package claude

import json "github.com/goccy/go-json"

type claudeMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type claudeTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema"`
}

type claudeRequest struct {
	Model       string          `json:"model"`
	Messages    []claudeMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	System      string          `json:"system,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Tools       []claudeTool    `json:"tools,omitempty"`
	ToolChoice  any             `json:"tool_choice,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Metadata    *claudeMetadata `json:"metadata,omitempty"`
}

type claudeMetadata struct {
	UserID string `json:"user_id,omitempty"`
}

type claudeResponse struct {
	ID           string          `json:"id"`
	Type         string          `json:"type"`
	Role         string          `json:"role"`
	Content      []claudeContent `json:"content"`
	Model        string          `json:"model"`
	StopReason   *string         `json:"stop_reason"`
	StopSequence *string         `json:"stop_sequence"`
	Usage        claudeUsage     `json:"usage"`
}

type claudeContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`

	ID         string `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	Input      any    `json:"input,omitempty"`
	ToolCallID string `json:"tool_use_id,omitempty"`
	Content    string `json:"content,omitempty"`
}

type claudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type claudeModelsResponse struct {
	Data []claudeModel `json:"data"`
}

type claudeModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created_at,omitempty"`
}

type claudeStreamEvent struct {
	Type         string          `json:"type"`
	Index        int             `json:"index,omitempty"`
	Delta        json.RawMessage `json:"delta,omitempty"`
	Message      json.RawMessage `json:"message,omitempty"`
	ContentBlock json.RawMessage `json:"content_block,omitempty"`
	Usage        *claudeUsage    `json:"usage,omitempty"`
}

type claudeStreamMessage struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Role  string `json:"role"`
	Model string `json:"model"`
}

type claudeStreamContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type claudeStreamDelta struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`

	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Input any    `json:"input,omitempty"`

	PartialJSON string `json:"partial_json,omitempty"`

	StopReason string `json:"stop_reason,omitempty"`
}

type claudeStreamUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
