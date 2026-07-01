package llm

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

type FinishReason string

const (
	FinishReasonStop          FinishReason = "stop"
	FinishReasonLength        FinishReason = "length"
	FinishReasonContentFilter FinishReason = "content_filter"
	FinishReasonToolCalls     FinishReason = "tool_calls"
	FinishReasonNull          FinishReason = "null"
)

type ToolChoice struct {
	Type     string         `json:"type,omitempty"`
	Function ToolChoiceFunc `json:"function,omitempty"`
}

type ToolChoiceFunc struct {
	Name string `json:"name"`
}
