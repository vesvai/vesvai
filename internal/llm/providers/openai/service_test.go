package openai

import (
	"encoding/json"
	"testing"

	"github.com/vesvai/vesvai/internal/llm"
)

func TestParseContent_JSONString(t *testing.T) {
	got := parseContent(json.RawMessage(`"hello"`))
	if s, ok := got.(string); !ok || s != "hello" {
		t.Errorf("parseContent = %v (%T), want \"hello\"", got, got)
	}
}

func TestParseContent_Array(t *testing.T) {
	raw := json.RawMessage(`[{"type":"text","text":"hi"}]`)
	got := parseContent(raw)
	arr, ok := got.([]any)
	if !ok || len(arr) != 1 {
		t.Fatalf("parseContent = %v (%T), want 1-element array", got, got)
	}
}

func TestParseContent_Empty(t *testing.T) {
	if got := parseContent(json.RawMessage(``)); got != "" {
		t.Errorf("parseContent(empty) = %v, want \"\"", got)
	}
}

func TestParseRaw_Nil(t *testing.T) {
	if got := parseRaw(json.RawMessage(``)); got != nil {
		t.Errorf("parseRaw(empty) = %v, want nil", got)
	}
}

func TestParseRaw_String(t *testing.T) {
	got := parseRaw(json.RawMessage(`"thinking…"`))
	if s, ok := got.(string); !ok || s != "thinking…" {
		t.Errorf("parseRaw = %v (%T), want \"thinking…\"", got, got)
	}
}

func TestMustJSONString(t *testing.T) {
	if got := mustJSONString("a\"b"); got != `"a\"b"` {
		t.Errorf("mustJSONString = %q, want %q", got, `"a\"b"`)
	}
}

func TestContentToWire_TextOnly(t *testing.T) {
	raw := contentToWire(llm.TextContent("hello"))
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		t.Fatalf("text-only content should be an array, got string %q", s)
	}
	var blocks []map[string]any
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(blocks) != 1 || blocks[0]["type"] != "text" || blocks[0]["text"] != "hello" {
		t.Errorf("blocks = %v, want single text block", blocks)
	}
}

func TestContentToWire_Image(t *testing.T) {
	raw := contentToWire(llm.Content{
		Text:        "look",
		Attachments: []llm.Attachment{llm.NewImageAttachmentFromBase64("image/png", "AAA=")},
	})
	var blocks []map[string]any
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("blocks len = %d, want 2", len(blocks))
	}
	if blocks[1]["type"] != "image_url" {
		t.Errorf("second block type = %v, want image_url", blocks[1]["type"])
	}
}

func TestContentToWire_Empty(t *testing.T) {
	raw := contentToWire(llm.Content{})
	var s string
	if err := json.Unmarshal(raw, &s); err != nil || s != "" {
		t.Errorf("empty content should be \"\", got %s (err=%v)", raw, err)
	}
}

func TestToChatMessage_StringContent(t *testing.T) {
	cm := toChatMessage(llm.UserMessage("hi"))
	if cm.Role != "user" {
		t.Errorf("role = %q, want user", cm.Role)
	}
	var s string
	if err := json.Unmarshal(cm.Content, &s); err != nil || s != "hi" {
		t.Errorf("content = %s, want \"hi\" (err=%v)", cm.Content, err)
	}
}

func TestToLLMResponse_ToolCall(t *testing.T) {
	fr := "tool_calls"
	resp := chatResponse{
		ID: "id1",
		Choices: []chatChoice{{
			Index: 0,
			Message: &chatMessage{
				Role:    "assistant",
				Content: json.RawMessage(`"ok"`),
				ToolCalls: []chatToolCall{{
					ID: "call_1", Type: "function",
					Function: chatFunction{Name: "get_weather", Arguments: `{"loc":"sf"}`},
				}},
			},
			FinishReason: &fr,
		}},
	}
	s := &Service{}
	out := s.toLLMResponse(resp)
	if out.GetContent() != "ok" {
		t.Errorf("GetContent = %q, want ok", out.GetContent())
	}
	tcs := out.GetToolCalls()
	if len(tcs) != 1 || tcs[0].ID != "call_1" || tcs[0].Function.Name != "get_weather" {
		t.Errorf("tool calls = %+v, want call_1/get_weather", tcs)
	}
	if out.GetFinishReason() != llm.FinishReasonToolCalls {
		t.Errorf("finish = %q, want tool_calls", out.GetFinishReason())
	}
}

func TestToStreamChunk_ContentAndFinish(t *testing.T) {
	content := "hi"
	fr := "stop"
	resp := chatResponse{
		Choices: []chatChoice{{
			Index:        0,
			Delta:        &chatDelta{Content: &content},
			FinishReason: &fr,
		}},
	}
	s := &Service{}
	chunk := s.toStreamChunk(resp)
	if chunk.Content != "hi" {
		t.Errorf("content = %q, want hi", chunk.Content)
	}
	if chunk.FinishReason != llm.FinishReasonStop || !chunk.IsDone {
		t.Errorf("finish = %q done=%v, want stop/true", chunk.FinishReason, chunk.IsDone)
	}
}

func TestToStreamChunk_NullFinishNotDone(t *testing.T) {
	fr := "null"
	resp := chatResponse{
		Choices: []chatChoice{{
			Index:        0,
			FinishReason: &fr,
		}},
	}
	s := &Service{}
	chunk := s.toStreamChunk(resp)
	if chunk.IsDone {
		t.Error("IsDone should be false for finish_reason=null")
	}
}

func TestToStreamChunk_ToolCallDelta(t *testing.T) {
	resp := chatResponse{
		Choices: []chatChoice{{
			Index: 0,
			Delta: &chatDelta{ToolCalls: []chatDeltaTC{{
				Index: 0, ID: "call_1", Type: "function",
				Function: chatFunc{Name: "search"},
			}}},
		}},
	}
	s := &Service{}
	chunk := s.toStreamChunk(resp)
	if len(chunk.ToolCalls) != 1 || chunk.ToolCalls[0].Function.Name != "search" || chunk.ToolCalls[0].Index != 0 {
		t.Errorf("tool calls = %+v, want search@0", chunk.ToolCalls)
	}
}

func TestToStreamChunk_Usage(t *testing.T) {
	resp := chatResponse{
		Usage: chatUsage{
			PromptTokens: 5, CompletionTokens: 3, TotalTokens: 8,
			CompletionTokensDetails: &chatCompletionDetails{ReasoningTokens: 2},
		},
	}
	s := &Service{}
	chunk := s.toStreamChunk(resp)
	if chunk.Usage == nil || chunk.Usage.TotalTokens != 8 || chunk.Usage.CompletionTokensDetails.ReasoningTokens != 2 {
		t.Errorf("usage = %+v, want totals with reasoning=2", chunk.Usage)
	}
}

func TestToLLMUsage_ReasoningTokens(t *testing.T) {
	u := toLLMUsage(chatUsage{
		PromptTokens:            1,
		CompletionTokens:        2,
		TotalTokens:             3,
		CompletionTokensDetails: &chatCompletionDetails{ReasoningTokens: 9},
	})
	if u.CompletionTokensDetails == nil || u.CompletionTokensDetails.ReasoningTokens != 9 {
		t.Errorf("reasoning tokens = %v, want 9", u.CompletionTokensDetails)
	}
}

func TestSummariseErrorBody_ExtractsMessage(t *testing.T) {
	body := `{"error":{"message":"quota exceeded","type":"insufficient_quota"}}`
	got := summariseErrorBody(body, 429)
	want := "API error (HTTP 429): quota exceeded"
	if got != want {
		t.Errorf("summariseErrorBody = %q, want %q", got, want)
	}
	if got := summariseErrorBody("", 500); got != "API error (HTTP 500)" {
		t.Errorf("empty body = %q", got)
	}
}

func TestMapError_NonHTTP(t *testing.T) {
	e := errSimple{"x"}
	if got := mapError(e); got != error(e) {
		t.Errorf("non-http error should pass through, got %v", got)
	}
}

type errSimple struct{ s string }

func (errSimple) Error() string { return "x" }

func TestName(t *testing.T) {
	s := NewService("openai", ServiceConfig{BaseURL: "https://api.openai.com/v1"})
	if s.Name() != "openai" {
		t.Errorf("Name = %q, want openai", s.Name())
	}
}

