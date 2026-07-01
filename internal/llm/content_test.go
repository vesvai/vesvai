package llm

import "testing"

func TestTextContent(t *testing.T) {
	c := TextContent("hello world")
	if c.Text != "hello world" {
		t.Errorf("Text = %q, want %q", c.Text, "hello world")
	}
	if c.Attachments != nil {
		t.Error("Attachments should be nil for TextContent")
	}
}

func TestTextContent_Empty(t *testing.T) {
	c := TextContent("")
	if c.Text != "" {
		t.Errorf("Text = %q, want empty", c.Text)
	}
}

func TestContentWithAttachments(t *testing.T) {
	atts := []Attachment{
		NewImageAttachmentFromURL("https://example.com/img.png"),
		NewAudioAttachmentFromURL("https://example.com/audio.mp3"),
	}
	c := ContentWithAttachments("describe this", atts)
	if c.Text != "describe this" {
		t.Errorf("Text = %q, want %q", c.Text, "describe this")
	}
	if len(c.Attachments) != 2 {
		t.Errorf("Attachments len = %d, want 2", len(c.Attachments))
	}
}

func TestContent_HasAttachments(t *testing.T) {
	tests := []struct {
		name     string
		content  Content
		expected bool
	}{
		{
			name:     "no attachments",
			content:  TextContent("hello"),
			expected: false,
		},
		{
			name:     "nil attachments",
			content:  Content{Text: "hello"},
			expected: false,
		},
		{
			name:     "has attachments",
			content:  ContentWithAttachments("hello", []Attachment{NewImageAttachmentFromURL("https://x.com/a.png")}),
			expected: true,
		},
		{
			name:     "empty text with attachments",
			content:  Content{Attachments: []Attachment{NewImageAttachmentFromURL("https://x.com/a.png")}},
			expected: true,
		},
		{
			name:     "empty slice",
			content:  Content{Text: "hello", Attachments: []Attachment{}},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.content.HasAttachments(); got != tt.expected {
				t.Errorf("HasAttachments() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewMessage(t *testing.T) {
	msg := NewMessage(RoleUser, "what is 2+2?")
	if msg.Role != RoleUser {
		t.Errorf("Role = %q, want %q", msg.Role, RoleUser)
	}
	if msg.Content != "what is 2+2?" {
		t.Errorf("Content = %v, want %q", msg.Content, "what is 2+2?")
	}
	if msg.Name != "" {
		t.Errorf("Name = %q, want empty", msg.Name)
	}
	if msg.ToolCallID != "" {
		t.Errorf("ToolCallID = %q, want empty", msg.ToolCallID)
	}
}

func TestUserMessage(t *testing.T) {
	msg := UserMessage("hello")
	if msg.Role != RoleUser {
		t.Errorf("Role = %q, want %q", msg.Role, RoleUser)
	}
	if msg.Content != "hello" {
		t.Errorf("Content = %v, want %q", msg.Content, "hello")
	}
}

func TestUserMessage_StructContent(t *testing.T) {
	type multimodal struct {
		Type    string   `json:"type"`
		Text    string   `json:"text"`
		Images  []string `json:"images"`
	}
	content := multimodal{
		Type:   "multimodal",
		Text:   "look at this",
		Images: []string{"https://example.com/1.png"},
	}
	msg := UserMessage(content)
	if msg.Role != RoleUser {
		t.Errorf("Role = %q, want %q", msg.Role, RoleUser)
	}
	mc, ok := msg.Content.(multimodal)
	if !ok {
		t.Fatal("Content should be multimodal type")
	}
	if mc.Text != "look at this" {
		t.Errorf("Text = %q, want %q", mc.Text, "look at this")
	}
	if len(mc.Images) != 1 {
		t.Errorf("Images len = %d, want 1", len(mc.Images))
	}
}

func TestSystemMessage(t *testing.T) {
	msg := SystemMessage("you are a helpful assistant")
	if msg.Role != RoleSystem {
		t.Errorf("Role = %q, want %q", msg.Role, RoleSystem)
	}
	if msg.Content != "you are a helpful assistant" {
		t.Errorf("Content = %v, want %q", msg.Content, "you are a helpful assistant")
	}
}

func TestAssistantMessage(t *testing.T) {
	msg := AssistantMessage("the answer is 42")
	if msg.Role != RoleAssistant {
		t.Errorf("Role = %q, want %q", msg.Role, RoleAssistant)
	}
	if msg.Content != "the answer is 42" {
		t.Errorf("Content = %v, want %q", msg.Content, "the answer is 42")
	}
}

func TestToolMessage(t *testing.T) {
	msg := ToolMessage(`{"result": 42}`, "call_123")
	if msg.Role != RoleTool {
		t.Errorf("Role = %q, want %q", msg.Role, RoleTool)
	}
	if msg.Content != `{"result": 42}` {
		t.Errorf("Content = %v, want %q", msg.Content, `{"result": 42}`)
	}
	if msg.ToolCallID != "call_123" {
		t.Errorf("ToolCallID = %q, want %q", msg.ToolCallID, "call_123")
	}
}

func TestMessage_ContentIsAny(t *testing.T) {
	// Content is `any` - verify different types work
	t.Run("string", func(t *testing.T) {
		msg := NewMessage(RoleUser, "hello")
		if s, ok := msg.Content.(string); !ok || s != "hello" {
			t.Errorf("Content = %v, want %q", msg.Content, "hello")
		}
	})
	t.Run("int", func(t *testing.T) {
		msg := NewMessage(RoleUser, 42)
		if n, ok := msg.Content.(int); !ok || n != 42 {
			t.Errorf("Content = %v, want 42", msg.Content)
		}
	})
	t.Run("nil", func(t *testing.T) {
		msg := NewMessage(RoleUser, nil)
		if msg.Content != nil {
			t.Errorf("Content = %v, want nil", msg.Content)
		}
	})
	t.Run("slice", func(t *testing.T) {
		msg := NewMessage(RoleUser, []string{"a", "b"})
		s, ok := msg.Content.([]string)
		if !ok || len(s) != 2 || s[0] != "a" || s[1] != "b" {
			t.Errorf("Content = %v, want [a b]", msg.Content)
		}
	})
	t.Run("map", func(t *testing.T) {
		msg := NewMessage(RoleUser, map[string]int{"x": 1})
		m, ok := msg.Content.(map[string]int)
		if !ok || m["x"] != 1 {
			t.Errorf("Content = %v, want {x:1}", msg.Content)
		}
	})
}
