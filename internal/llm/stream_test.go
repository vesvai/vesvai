package llm

import (
	"encoding/json"
	"testing"
)

func TestStreamResponse_ToChunk_Content(t *testing.T) {
	content := "hello"
	sr := &StreamResponse{
		Choices: []StreamChoice{
			{
				Index: 0,
				Delta: &MessageDelta{
					Content: &content,
				},
			},
		},
	}
	chunk := sr.ToChunk()
	if chunk.Content != "hello" {
		t.Errorf("Content = %q, want %q", chunk.Content, "hello")
	}
	if chunk.IsDone {
		t.Error("IsDone should be false")
	}
}

func TestStreamResponse_ToChunk_ToolCalls(t *testing.T) {
	sr := &StreamResponse{
		Choices: []StreamChoice{
			{
				Index: 0,
				Delta: &MessageDelta{
					ToolCalls: []ToolCall{
						{ID: "call_1", Type: "function", Function: Function{Name: "get_weather"}},
					},
				},
			},
		},
	}
	chunk := sr.ToChunk()
	if len(chunk.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(chunk.ToolCalls))
	}
	if chunk.ToolCalls[0].ID != "call_1" {
		t.Errorf("ToolCall ID = %q, want %q", chunk.ToolCalls[0].ID, "call_1")
	}
	if chunk.IsDone {
		t.Error("IsDone should be false")
	}
}

func TestStreamResponse_ToChunk_FinishReason_Stop(t *testing.T) {
	fr := FinishReasonStop
	empty := ""
	sr := &StreamResponse{
		Choices: []StreamChoice{
			{
				Index:        0,
				Delta:        &MessageDelta{Content: &empty},
				FinishReason: &fr,
			},
		},
	}
	chunk := sr.ToChunk()
	if chunk.FinishReason != FinishReasonStop {
		t.Errorf("FinishReason = %q, want %q", chunk.FinishReason, FinishReasonStop)
	}
	if !chunk.IsDone {
		t.Error("IsDone should be true for stop reason")
	}
}

func TestStreamResponse_ToChunk_FinishReason_Null(t *testing.T) {
	fr := FinishReasonNull
	empty := ""
	sr := &StreamResponse{
		Choices: []StreamChoice{
			{
				Index:        0,
				Delta:        &MessageDelta{Content: &empty},
				FinishReason: &fr,
			},
		},
	}
	chunk := sr.ToChunk()
	if chunk.FinishReason != FinishReasonNull {
		t.Errorf("FinishReason = %q, want %q", chunk.FinishReason, FinishReasonNull)
	}
	if chunk.IsDone {
		t.Error("IsDone should be false for null finish reason")
	}
}

func TestStreamResponse_ToChunk_FinishReason_Length(t *testing.T) {
	fr := FinishReasonLength
	empty := ""
	sr := &StreamResponse{
		Choices: []StreamChoice{
			{
				Index:        0,
				Delta:        &MessageDelta{Content: &empty},
				FinishReason: &fr,
			},
		},
	}
	chunk := sr.ToChunk()
	if chunk.FinishReason != FinishReasonLength {
		t.Errorf("FinishReason = %q, want %q", chunk.FinishReason, FinishReasonLength)
	}
	if !chunk.IsDone {
		t.Error("IsDone should be true for length reason")
	}
}

func TestStreamResponse_ToChunk_EmptyChoices(t *testing.T) {
	sr := &StreamResponse{
		Choices: []StreamChoice{},
	}
	chunk := sr.ToChunk()
	if chunk.Content != "" {
		t.Errorf("Content = %q, want empty", chunk.Content)
	}
	if chunk.IsDone {
		t.Error("IsDone should be false")
	}
}

func TestStreamResponse_ToChunk_NilDelta(t *testing.T) {
	sr := &StreamResponse{
		Choices: []StreamChoice{
			{Index: 0, Delta: nil},
		},
	}
	chunk := sr.ToChunk()
	if chunk.Content != "" {
		t.Errorf("Content = %q, want empty", chunk.Content)
	}
	if chunk.IsDone {
		t.Error("IsDone should be false")
	}
}

func TestStreamResponse_ToChunk_NilContentPointer(t *testing.T) {
	sr := &StreamResponse{
		Choices: []StreamChoice{
			{
				Index: 0,
				Delta: &MessageDelta{
					Content: nil,
				},
			},
		},
	}
	chunk := sr.ToChunk()
	if chunk.Content != "" {
		t.Errorf("Content = %q, want empty", chunk.Content)
	}
}

func TestStreamResponse_ToChunk_ContentToolCallsAndFinish(t *testing.T) {
	content := "partial"
	fr := FinishReasonStop
	sr := &StreamResponse{
		Choices: []StreamChoice{
			{
				Index: 0,
				Delta: &MessageDelta{
					Content: &content,
					ToolCalls: []ToolCall{
						{ID: "call_1", Type: "function", Function: Function{Name: "search"}},
					},
				},
				FinishReason: &fr,
			},
		},
	}
	chunk := sr.ToChunk()
	if chunk.Content != "partial" {
		t.Errorf("Content = %q, want %q", chunk.Content, "partial")
	}
	if len(chunk.ToolCalls) != 1 {
		t.Errorf("ToolCalls len = %d, want 1", len(chunk.ToolCalls))
	}
	if chunk.FinishReason != FinishReasonStop {
		t.Errorf("FinishReason = %q, want %q", chunk.FinishReason, FinishReasonStop)
	}
	if !chunk.IsDone {
		t.Error("IsDone should be true")
	}
}

func TestStreamResponse_JSONDeserialization(t *testing.T) {
	input := `{
		"id": "chatcmpl-123",
		"object": "chat.completion.chunk",
		"created": 1700000000,
		"model": "gpt-4",
		"choices": [{
			"index": 0,
			"delta": {"content": "hello"},
			"finish_reason": null
		}]
	}`

	var sr StreamResponse
	if err := json.Unmarshal([]byte(input), &sr); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if sr.ID != "chatcmpl-123" {
		t.Errorf("ID = %q, want %q", sr.ID, "chatcmpl-123")
	}
	if sr.Model != "gpt-4" {
		t.Errorf("Model = %q, want %q", sr.Model, "gpt-4")
	}
	if len(sr.Choices) != 1 {
		t.Fatalf("Choices len = %d, want 1", len(sr.Choices))
	}
	if sr.Choices[0].Delta == nil {
		t.Fatal("Delta is nil")
	}
	if sr.Choices[0].Delta.Content == nil || *sr.Choices[0].Delta.Content != "hello" {
		t.Errorf("Delta.Content = %v, want %q", sr.Choices[0].Delta.Content, "hello")
	}

	// Convert to chunk
	chunk := sr.ToChunk()
	if chunk.Content != "hello" {
		t.Errorf("ToChunk().Content = %q, want %q", chunk.Content, "hello")
	}
	if chunk.IsDone {
		t.Error("IsDone should be false for null finish_reason")
	}
}

func TestStreamChunk_DefaultValues(t *testing.T) {
	chunk := StreamChunk{}
	if chunk.Content != "" {
		t.Errorf("Content = %q, want empty", chunk.Content)
	}
	if chunk.ToolCalls != nil {
		t.Errorf("ToolCalls = %v, want nil", chunk.ToolCalls)
	}
	if chunk.FinishReason != "" {
		t.Errorf("FinishReason = %q, want empty", chunk.FinishReason)
	}
	if chunk.IsDone {
		t.Error("IsDone should be false")
	}
}

func TestStreamResponse_MultipleChoices(t *testing.T) {
	content1 := "first"
	content2 := "second"
	sr := &StreamResponse{
		Choices: []StreamChoice{
			{Index: 0, Delta: &MessageDelta{Content: &content1}},
			{Index: 1, Delta: &MessageDelta{Content: &content2}},
		},
	}
	// ToChunk only processes first choice
	chunk := sr.ToChunk()
	if chunk.Content != "first" {
		t.Errorf("Content = %q, want %q (should use first choice)", chunk.Content, "first")
	}
}

func TestMessageDelta_Role(t *testing.T) {
	role := RoleAssistant
	md := MessageDelta{
		Role: &role,
	}
	if md.Role == nil || *md.Role != RoleAssistant {
		t.Errorf("Role = %v, want %q", md.Role, RoleAssistant)
	}
}

func TestMessageDelta_Refusal(t *testing.T) {
	refusal := "I cannot help with that"
	md := MessageDelta{
		Refusal: &refusal,
	}
	if md.Refusal == nil || *md.Refusal != "I cannot help with that" {
		t.Errorf("Refusal = %v, want %q", md.Refusal, refusal)
	}
}
