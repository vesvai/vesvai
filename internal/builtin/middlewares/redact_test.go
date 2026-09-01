package middlewares

import (
	"strings"
	"testing"

	agentmw "github.com/vesvai/vesvai/internal/agent/middleware"
	"github.com/vesvai/vesvai/internal/llm"
)

func TestRedactionBeforeLLMRedactsAllMessages(t *testing.T) {
	r := NewRedaction()
	req := llm.NewRequest("m", []llm.Message{
		llm.SystemMessage(`password = "sys-secret-1234567890"`),
		llm.UserMessage(`my key is sk-abcdefghijklmnopqrstuvwxyz1234`),
		llm.ToolMessage(`read result: {"token": "file-secret-1234567890"}`, "call-1"),
		{
			Role:    llm.RoleAssistant,
			Content: "ok",
			ToolCalls: []llm.ToolCall{
				{Function: llm.Function{Arguments: `{"secret":"sk-abcdefghijklmnopqrstuvwxyz1234"}`}},
			},
		},
	})
	if err := r.BeforeLLM(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	for i, m := range req.Messages {
		s, _ := m.Content.(string)
		if strings.Contains(s, "sk-abcdefghijklmnopqrstuvwxyz1234") {
			t.Errorf("message %d content not redacted: %q", i, s)
		}
		if strings.Contains(s, "file-secret-1234567890") {
			t.Errorf("message %d content not redacted: %q", i, s)
		}
		if strings.Contains(s, "sys-secret-1234567890") {
			t.Errorf("message %d content not redacted: %q", i, s)
		}
	}
	args := req.Messages[3].ToolCalls[0].Function.Arguments
	if strings.Contains(args, "sk-abcdefghijklmnopqrstuvwxyz1234") {
		t.Error("tool-call args not redacted at request boundary")
	}
}

func TestRedactionAfterLLMPreservesToolCalls(t *testing.T) {
	r := NewRedaction()
	msg := &llm.Message{
		Content: "echo sk-abcdefghijklmnopqrstuvwxyz1234",
		ToolCalls: []llm.ToolCall{
			{Function: llm.Function{Arguments: `{"content":"sk-abcdefghijklmnopqrstuvwxyz1234"}`}},
		},
	}
	resp := &llm.Response{Choices: []llm.Choice{{Message: msg}}}
	if err := r.AfterLLM(t.Context(), nil, resp); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(msg.Content.(string), "sk-abcdefghijklmnopqrstuvwxyz1234") {
		t.Error("content not redacted in output")
	}
	if !strings.Contains(msg.ToolCalls[0].Function.Arguments, "sk-abcdefghijklmnopqrstuvwxyz1234") {
		t.Error("tool-call args must survive AfterLLM for execution")
	}
}

func TestRedactionAfterRun(t *testing.T) {
	r := NewRedaction()
	res := &agentmw.Result{
		Output: "token sk-abcdefghijklmnopqrstuvwxyz1234",
		History: []llm.Message{
			llm.ToolMessage(`{"api_key":"sk-abcdefghijklmnopqrstuvwxyz1234"}`, "c1"),
		},
	}
	if err := r.AfterRun(t.Context(), "a", res, nil); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.Output, "sk-abcdefghijklmnopqrstuvwxyz1234") {
		t.Error("result output not redacted")
	}
	h := res.History[0].Content.(string)
	if strings.Contains(h, "sk-abcdefghijklmnopqrstuvwxyz1234") {
		t.Error("history not redacted")
	}
}

func TestRedactionRedactString(t *testing.T) {
	r := NewRedaction()
	if got := r.RedactString("AKIAIOSFODNN7EXAMPLE"); !strings.Contains(got, "[**REDACTED**]") {
		t.Errorf("RedactString = %q", got)
	}
}

func TestRedactionNilSafe(t *testing.T) {
	r := NewRedaction()
	if err := r.BeforeLLM(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if err := r.AfterLLM(t.Context(), nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := r.AfterRun(t.Context(), "a", nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestRedactionBeforeLLMSkipsMediaContentParts(t *testing.T) {
	r := NewRedaction()
	img := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="
	req := llm.NewRequest("m", []llm.Message{
		{Content: []any{
			map[string]any{"type": "text", "text": `api_key="sk-1234567890abcdefghijklmno"`},
			map[string]any{"type": "image_url", "image_url": map[string]any{"url": "data:image/png;base64," + img}},
		}},
	})
	if err := r.BeforeLLM(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	parts := req.Messages[0].Content.([]any)
	text := parts[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "[**REDACTED**]") {
		t.Errorf("text part not redacted: %q", text)
	}
	url := parts[1].(map[string]any)["image_url"].(map[string]any)["url"].(string)
	if strings.Contains(url, "[**REDACTED**]") {
		t.Errorf("image url payload must not be redacted: %q", url)
	}
}

func TestRedactionAfterRunNilResult(t *testing.T) {
	r := NewRedaction()
	if err := r.AfterRun(t.Context(), "a", nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestRedactionBeforeLLMEmptyMessages(t *testing.T) {
	r := NewRedaction()
	req := llm.NewRequest("m", nil)
	if err := r.BeforeLLM(t.Context(), req); err != nil {
		t.Fatal(err)
	}
}
