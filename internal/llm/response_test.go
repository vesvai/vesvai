package llm

import (
	"encoding/json"
	"testing"
)

func strPtr(s string) *string    { return &s }
func frPtr(r FinishReason) *FinishReason { return &r }

func TestResponse_GetContent_StringContent(t *testing.T) {
	r := &Response{
		Choices: []Choice{
			{
				Message: &Message{
					Role:    RoleAssistant,
					Content: "hello world",
				},
			},
		},
	}
	if got := r.GetContent(); got != "hello world" {
		t.Errorf("GetContent() = %q, want %q", got, "hello world")
	}
}

func TestResponse_GetContent_ArrayContent_TextItem(t *testing.T) {
	content := []any{
		map[string]any{
			"type": "text",
			"text": "the answer is 42",
		},
	}
	r := &Response{
		Choices: []Choice{
			{Message: &Message{Role: RoleAssistant, Content: content}},
		},
	}
	if got := r.GetContent(); got != "the answer is 42" {
		t.Errorf("GetContent() = %q, want %q", got, "the answer is 42")
	}
}

func TestResponse_GetContent_ArrayContent_NoTextItem(t *testing.T) {
	content := []any{
		map[string]any{
			"type": "image_url",
			"url":  "https://example.com/img.png",
		},
	}
	r := &Response{
		Choices: []Choice{
			{Message: &Message{Role: RoleAssistant, Content: content}},
		},
	}
	if got := r.GetContent(); got != "" {
		t.Errorf("GetContent() = %q, want empty", got)
	}
}

func TestResponse_GetContent_EmptyChoices(t *testing.T) {
	r := &Response{Choices: []Choice{}}
	if got := r.GetContent(); got != "" {
		t.Errorf("GetContent() = %q, want empty", got)
	}
}

func TestResponse_GetContent_NilMessage(t *testing.T) {
	r := &Response{
		Choices: []Choice{{Index: 0, Message: nil}},
	}
	if got := r.GetContent(); got != "" {
		t.Errorf("GetContent() = %q, want empty", got)
	}
}

func TestResponse_GetContent_NilContent(t *testing.T) {
	r := &Response{
		Choices: []Choice{
			{Message: &Message{Role: RoleAssistant, Content: nil}},
		},
	}
	if got := r.GetContent(); got != "" {
		t.Errorf("GetContent() = %q, want empty", got)
	}
}

func TestResponse_GetToolCalls_WithToolCalls(t *testing.T) {
	content := []any{
		map[string]any{
			"type": "tool_calls",
			"tool_calls": []any{
				map[string]any{
					"id":   "call_abc",
					"type": "function",
					"function": map[string]any{
						"name":      "get_weather",
						"arguments": `{"city":"NYC"}`,
					},
				},
				map[string]any{
					"id":   "call_def",
					"type": "function",
					"function": map[string]any{
						"name":      "get_time",
						"arguments": `{"tz":"EST"}`,
					},
				},
			},
		},
	}
	r := &Response{
		Choices: []Choice{
			{Message: &Message{Role: RoleAssistant, Content: content}},
		},
	}

	calls := r.GetToolCalls()
	if len(calls) != 2 {
		t.Fatalf("GetToolCalls() len = %d, want 2", len(calls))
	}

	if calls[0].ID != "call_abc" {
		t.Errorf("call[0].ID = %q, want %q", calls[0].ID, "call_abc")
	}
	if calls[0].Type != "function" {
		t.Errorf("call[0].Type = %q, want %q", calls[0].Type, "function")
	}
	if calls[0].Function.Name != "get_weather" {
		t.Errorf("call[0].Function.Name = %q, want %q", calls[0].Function.Name, "get_weather")
	}
	if calls[0].Function.Arguments != `{"city":"NYC"}` {
		t.Errorf("call[0].Function.Arguments = %q, want %q", calls[0].Function.Arguments, `{"city":"NYC"}`)
	}

	if calls[1].ID != "call_def" {
		t.Errorf("call[1].ID = %q, want %q", calls[1].ID, "call_def")
	}
	if calls[1].Function.Name != "get_time" {
		t.Errorf("call[1].Function.Name = %q, want %q", calls[1].Function.Name, "get_time")
	}
}

func TestResponse_GetToolCalls_NoToolCalls(t *testing.T) {
	r := &Response{
		Choices: []Choice{
			{Message: &Message{Role: RoleAssistant, Content: "just text"}},
		},
	}
	if calls := r.GetToolCalls(); calls != nil {
		t.Errorf("GetToolCalls() = %v, want nil", calls)
	}
}

func TestResponse_GetToolCalls_EmptyChoices(t *testing.T) {
	r := &Response{Choices: []Choice{}}
	if calls := r.GetToolCalls(); calls != nil {
		t.Errorf("GetToolCalls() = %v, want nil", calls)
	}
}

func TestResponse_GetToolCalls_NilMessage(t *testing.T) {
	r := &Response{
		Choices: []Choice{{Message: nil}},
	}
	if calls := r.GetToolCalls(); calls != nil {
		t.Errorf("GetToolCalls() = %v, want nil", calls)
	}
}

func TestResponse_GetFinishReason_WithReason(t *testing.T) {
	fr := FinishReasonStop
	r := &Response{
		Choices: []Choice{
			{FinishReason: &fr},
		},
	}
	if got := r.GetFinishReason(); got != FinishReasonStop {
		t.Errorf("GetFinishReason() = %q, want %q", got, FinishReasonStop)
	}
}

func TestResponse_GetFinishReason_ToolCalls(t *testing.T) {
	fr := FinishReasonToolCalls
	r := &Response{
		Choices: []Choice{
			{FinishReason: &fr},
		},
	}
	if got := r.GetFinishReason(); got != FinishReasonToolCalls {
		t.Errorf("GetFinishReason() = %q, want %q", got, FinishReasonToolCalls)
	}
}

func TestResponse_GetFinishReason_NilFinishReason(t *testing.T) {
	r := &Response{
		Choices: []Choice{
			{FinishReason: nil},
		},
	}
	if got := r.GetFinishReason(); got != "" {
		t.Errorf("GetFinishReason() = %q, want empty", got)
	}
}

func TestResponse_GetFinishReason_EmptyChoices(t *testing.T) {
	r := &Response{Choices: []Choice{}}
	if got := r.GetFinishReason(); got != "" {
		t.Errorf("GetFinishReason() = %q, want empty", got)
	}
}

func TestResponse_JSONRoundTrip(t *testing.T) {
	fr := FinishReasonStop
	original := &Response{
		ID:      "chatcmpl-123",
		Object:  "chat.completion",
		Created: 1700000000,
		Model:   "gpt-4",
		Choices: []Choice{
			{
				Index: 0,
				Message: &Message{
					Role:    RoleAssistant,
					Content: "the answer is 42",
				},
				FinishReason: &fr,
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if decoded.ID != "chatcmpl-123" {
		t.Errorf("ID = %q, want %q", decoded.ID, "chatcmpl-123")
	}
	if decoded.Model != "gpt-4" {
		t.Errorf("Model = %q, want %q", decoded.Model, "gpt-4")
	}
	if decoded.Usage.TotalTokens != 15 {
		t.Errorf("Usage.TotalTokens = %d, want 15", decoded.Usage.TotalTokens)
	}
	if len(decoded.Choices) != 1 {
		t.Fatalf("Choices len = %d, want 1", len(decoded.Choices))
	}
	if decoded.Choices[0].Message == nil {
		t.Fatal("Message is nil")
	}
	if decoded.Choices[0].Message.Content != "the answer is 42" {
		t.Errorf("Content = %q, want %q", decoded.Choices[0].Message.Content, "the answer is 42")
	}
	if decoded.Choices[0].FinishReason == nil || *decoded.Choices[0].FinishReason != FinishReasonStop {
		t.Errorf("FinishReason = %v, want %q", decoded.Choices[0].FinishReason, FinishReasonStop)
	}
}

func TestResponse_GetContent_ComplexArray(t *testing.T) {
	// Test with multiple items, only first text item returned
	content := []any{
		map[string]any{
			"type": "image_url",
			"url":  "https://example.com/img.png",
		},
		map[string]any{
			"type": "text",
			"text": "first text block",
		},
		map[string]any{
			"type": "text",
			"text": "second text block",
		},
	}
	r := &Response{
		Choices: []Choice{
			{Message: &Message{Role: RoleAssistant, Content: content}},
		},
	}
	if got := r.GetContent(); got != "first text block" {
		t.Errorf("GetContent() = %q, want %q", got, "first text block")
	}
}

func TestResponse_GetContent_MalformedArray(t *testing.T) {
	// Array with non-map items should not panic
	content := []any{"not a map", 42, nil}
	r := &Response{
		Choices: []Choice{
			{Message: &Message{Role: RoleAssistant, Content: content}},
		},
	}
	if got := r.GetContent(); got != "" {
		t.Errorf("GetContent() = %q, want empty", got)
	}
}

func TestUsage_JSON(t *testing.T) {
	u := Usage{
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
	}
	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	var decoded Usage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}
	if decoded.PromptTokens != 10 || decoded.CompletionTokens != 20 || decoded.TotalTokens != 30 {
		t.Errorf("decoded Usage = %+v", decoded)
	}
}

func TestToolCall_JSON(t *testing.T) {
	tc := ToolCall{
		ID:   "call_123",
		Type: "function",
		Function: Function{
			Name:      "get_weather",
			Arguments: `{"city":"NYC"}`,
		},
	}
	data, err := json.Marshal(tc)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	var decoded ToolCall
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}
	if decoded.ID != "call_123" {
		t.Errorf("ID = %q, want %q", decoded.ID, "call_123")
	}
	if decoded.Function.Name != "get_weather" {
		t.Errorf("Name = %q, want %q", decoded.Function.Name, "get_weather")
	}
}
