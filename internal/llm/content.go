package llm

type Content struct {
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

func TextContent(text string) Content {
	return Content{Text: text}
}

func ContentWithAttachments(text string, attachments []Attachment) Content {
	return Content{
		Text:        text,
		Attachments: attachments,
	}
}

func (c *Content) HasAttachments() bool {
	return len(c.Attachments) > 0
}

func MessageText(m Message) string {
	switch c := m.Content.(type) {
	case string:
		return c
	case Content:
		return c.Text
	case []any:
		for _, item := range c {
			if mm, ok := item.(map[string]any); ok {
				if t, ok := mm["type"].(string); ok && t == "text" {
					if text, ok := mm["text"].(string); ok {
						return text
					}
				}
			}
		}
	}
	return ""
}

type Message struct {
	Role       Role       `json:"role"`
	Content    any        `json:"content"`
	Reasoning  any        `json:"reasoning,omitempty"`
	Name       string     `json:"name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

func NewMessage(role Role, content any) Message {
	return Message{
		Role:    role,
		Content: content,
	}
}

func UserMessage(content any) Message {
	return NewMessage(RoleUser, content)
}

func SystemMessage(content any) Message {
	return NewMessage(RoleSystem, content)
}

func AssistantMessage(content any) Message {
	return NewMessage(RoleAssistant, content)
}

func ToolMessage(content string, toolCallID string) Message {
	return Message{
		Role:       RoleTool,
		Content:    content,
		ToolCallID: toolCallID,
	}
}
