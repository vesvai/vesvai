package llm

import (
	"encoding/json"
	"testing"
)

func TestNewRequest(t *testing.T) {
	msgs := []Message{UserMessage("hi")}
	r := NewRequest("gpt-4", msgs)
	if r.Model != "gpt-4" {
		t.Errorf("Model = %q, want %q", r.Model, "gpt-4")
	}
	if len(r.Messages) != 1 {
		t.Errorf("Messages len = %d, want 1", len(r.Messages))
	}
	// Zero values for optional fields
	if r.Temperature != 0 {
		t.Errorf("Temperature = %v, want 0", r.Temperature)
	}
	if r.MaxTokens != 0 {
		t.Errorf("MaxTokens = %v, want 0", r.MaxTokens)
	}
	if r.Stream {
		t.Error("Stream should be false")
	}
}

func TestRequest_BuilderChaining(t *testing.T) {
	r := NewRequest("gpt-4", []Message{UserMessage("hi")})
	fr := FinishReasonStop
	tc := &ToolChoice{Type: "function", Function: ToolChoiceFunc{Name: "get_weather"}}
	rf := &ResponseFormat{Type: "json_object"}

	result := r.
		WithTemperature(0.7).
		WithMaxTokens(1024).
		WithTopP(0.9).
		WithTools([]Tool{
			{Type: "function", Function: ToolFunction{Name: "get_weather", Description: "Get weather"}},
		}).
		WithToolChoice(tc).
		WithResponseFormat(rf).
		WithStream(true).
		WithN(2).
		WithPresencePenalty(0.5).
		WithFrequencyPenalty(0.3).
		WithUser("user-123")

	if result != r {
		t.Error("Builder methods should return the same pointer")
	}
	if r.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want 0.7", r.Temperature)
	}
	if r.MaxTokens != 1024 {
		t.Errorf("MaxTokens = %d, want 1024", r.MaxTokens)
	}
	if r.TopP != 0.9 {
		t.Errorf("TopP = %v, want 0.9", r.TopP)
	}
	if len(r.Tools) != 1 {
		t.Fatalf("Tools len = %d, want 1", len(r.Tools))
	}
	if r.Tools[0].Function.Name != "get_weather" {
		t.Errorf("Tool Name = %q, want %q", r.Tools[0].Function.Name, "get_weather")
	}
	if r.ToolChoice != tc {
		t.Error("ToolChoice pointer mismatch")
	}
	if r.ResponseFormat != rf {
		t.Error("ResponseFormat pointer mismatch")
	}
	if !r.Stream {
		t.Error("Stream should be true")
	}
	if r.N != 2 {
		t.Errorf("N = %d, want 2", r.N)
	}
	if r.PresencePenalty != 0.5 {
		t.Errorf("PresencePenalty = %v, want 0.5", r.PresencePenalty)
	}
	if r.FrequencyPenalty != 0.3 {
		t.Errorf("FrequencyPenalty = %v, want 0.3", r.FrequencyPenalty)
	}
	if r.User != "user-123" {
		t.Errorf("User = %q, want %q", r.User, "user-123")
	}
	_ = fr // used above
}

func TestRequest_JSONSerialization(t *testing.T) {
	r := NewRequest("gpt-4", []Message{
		{Role: RoleUser, Content: "hello"},
	})
	r.Temperature = 0.5
	r.MaxTokens = 100

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if decoded["model"] != "gpt-4" {
		t.Errorf("model = %v, want %q", decoded["model"], "gpt-4")
	}
	if decoded["temperature"] != 0.5 {
		t.Errorf("temperature = %v, want 0.5", decoded["temperature"])
	}
	if decoded["max_tokens"] != float64(100) {
		t.Errorf("max_tokens = %v, want 100", decoded["max_tokens"])
	}
	// stream=false should be omitted with omitempty
	if _, exists := decoded["stream"]; exists {
		t.Error("stream should be omitted when false")
	}
}

func TestRequest_JSONOmitEmpty(t *testing.T) {
	r := NewRequest("gpt-4", []Message{UserMessage("hi")})

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	// These zero-value fields should be absent
	absent := []string{"temperature", "top_p", "max_tokens", "stream", "tools", "tool_choice", "response_format", "n", "presence_penalty", "frequency_penalty", "user"}
	for _, key := range absent {
		if _, exists := decoded[key]; exists {
			t.Errorf("field %q should be omitted when zero", key)
		}
	}
}

func TestRequest_IndividualBuilders(t *testing.T) {
	t.Run("WithTemperature", func(t *testing.T) {
		r := NewRequest("m", nil).WithTemperature(1.5)
		if r.Temperature != 1.5 {
			t.Errorf("got %v, want 1.5", r.Temperature)
		}
	})

	t.Run("WithMaxTokens", func(t *testing.T) {
		r := NewRequest("m", nil).WithMaxTokens(4096)
		if r.MaxTokens != 4096 {
			t.Errorf("got %d, want 4096", r.MaxTokens)
		}
	})

	t.Run("WithTopP", func(t *testing.T) {
		r := NewRequest("m", nil).WithTopP(0.95)
		if r.TopP != 0.95 {
			t.Errorf("got %v, want 0.95", r.TopP)
		}
	})

	t.Run("WithStream", func(t *testing.T) {
		r := NewRequest("m", nil).WithStream(true)
		if !r.Stream {
			t.Error("Stream should be true")
		}
	})

	t.Run("WithN", func(t *testing.T) {
		r := NewRequest("m", nil).WithN(3)
		if r.N != 3 {
			t.Errorf("got %d, want 3", r.N)
		}
	})

	t.Run("WithPresencePenalty", func(t *testing.T) {
		r := NewRequest("m", nil).WithPresencePenalty(0.5)
		if r.PresencePenalty != 0.5 {
			t.Errorf("got %v, want 0.5", r.PresencePenalty)
		}
	})

	t.Run("WithFrequencyPenalty", func(t *testing.T) {
		r := NewRequest("m", nil).WithFrequencyPenalty(0.3)
		if r.FrequencyPenalty != 0.3 {
			t.Errorf("got %v, want 0.3", r.FrequencyPenalty)
		}
	})

	t.Run("WithUser", func(t *testing.T) {
		r := NewRequest("m", nil).WithUser("u1")
		if r.User != "u1" {
			t.Errorf("got %q, want %q", r.User, "u1")
		}
	})
}
