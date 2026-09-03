package llm

import (
	"errors"
	"fmt"
	json "github.com/goccy/go-json"
	"strings"
)

type PromptTokensDetails struct {
	CachedTokens     int `json:"cached_tokens,omitempty"`
	CacheWriteTokens int `json:"cache_write_tokens,omitempty"`
	AudioTokens      int `json:"audio_tokens,omitempty"`
	VideoTokens      int `json:"video_tokens,omitempty"`
}

type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
	ImageTokens     int `json:"image_tokens,omitempty"`
	AudioTokens     int `json:"audio_tokens,omitempty"`
}

type CostDetails struct {
	UpstreamInferenceCost            float64 `json:"upstream_inference_cost,omitempty"`
	UpstreamInferencePromptCost      float64 `json:"upstream_inference_prompt_cost,omitempty"`
	UpstreamInferenceCompletionsCost float64 `json:"upstream_inference_completions_cost,omitempty"`
}

type Usage struct {
	PromptTokens            int                      `json:"prompt_tokens"`
	CompletionTokens        int                      `json:"completion_tokens"`
	TotalTokens             int                      `json:"total_tokens"`
	Cost                    float64                  `json:"cost,omitempty"`
	IsByok                  bool                     `json:"is_byok,omitempty"`
	PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
	CostDetails             *CostDetails             `json:"cost_details,omitempty"`
}

type ToolCall struct {
	Index    int      `json:"index,omitempty"`
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type MessageDelta struct {
	Role      *Role      `json:"role,omitempty"`
	Content   *string    `json:"content,omitempty"`
	Reasoning *string    `json:"reasoning,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Refusal   *string    `json:"refusal,omitempty"`
}

type Choice struct {
	Index        int           `json:"index"`
	Message      *Message      `json:"message,omitempty"`
	Delta        *MessageDelta `json:"delta,omitempty"`
	FinishReason *FinishReason `json:"finish_reason,omitempty"`
}

type Response struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
}

func (r *Response) GetContent() string {
	if len(r.Choices) == 0 {
		return ""
	}
	msg := r.Choices[0].Message
	if msg == nil {
		return ""
	}
	switch c := msg.Content.(type) {
	case string:
		return c
	case []any:
		for _, item := range c {
			if m, ok := item.(map[string]any); ok {
				if t, ok := m["type"].(string); ok && t == "text" {
					if text, ok := m["text"].(string); ok {
						return text
					}
				}
			}
		}
	}
	return ""
}

func (r *Response) GetToolCalls() []ToolCall {
	if len(r.Choices) == 0 {
		return nil
	}
	msg := r.Choices[0].Message
	if msg == nil {
		return nil
	}

	if len(msg.ToolCalls) > 0 {
		return msg.ToolCalls
	}

	switch c := msg.Content.(type) {
	case []any:
		for _, item := range c {
			if m, ok := item.(map[string]any); ok {
				if t, ok := m["type"].(string); ok && t == "tool_calls" {
					if tc, ok := m["tool_calls"].([]any); ok {
						var calls []ToolCall
						for _, tcItem := range tc {
							if tcMap, ok := tcItem.(map[string]any); ok {
								call := ToolCall{
									Type: "function",
								}
								if t, ok := tcMap["type"].(string); ok {
									call.Type = t
								}
								if id, ok := tcMap["id"].(string); ok {
									call.ID = id
								}
								if fn, ok := tcMap["function"].(map[string]any); ok {
									name, _ := fn["name"].(string)
									args, _ := fn["arguments"].(string)
									call.Function = Function{
										Name:      name,
										Arguments: args,
									}
								}
								calls = append(calls, call)
							}
						}
						return calls
					}
				}
			}
		}
	}
	return nil
}

func (r *Response) GetFinishReason() FinishReason {
	if len(r.Choices) == 0 {
		return ""
	}
	if r.Choices[0].FinishReason != nil {
		return *r.Choices[0].FinishReason
	}
	return ""
}

func (r *Response) GetReasoning() string {
	if len(r.Choices) == 0 {
		return ""
	}
	msg := r.Choices[0].Message
	if msg == nil {
		return ""
	}
	switch c := msg.Reasoning.(type) {
	case string:
		return c
	}
	return ""
}

func (r *Response) Structured(out any) error {
	if r == nil {
		return errors.New("llm: nil response")
	}
	return ParseStructured(r.GetContent(), out)
}

func ParseStructured(content string, out any) error {
	if out == nil {
		return errors.New("llm: nil output")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return errors.New("llm: empty structured output")
	}
	if strings.HasPrefix(content, "```") {
		content = stripCodeFence(content)
	}
	if err := json.Unmarshal([]byte(content), out); err != nil {
		return fmt.Errorf("llm: parse structured output: %w", err)
	}
	return nil
}

func stripCodeFence(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) > 0 {
		lines = lines[1:]
	}
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
			lines = lines[:i]
			break
		}
	}
	return strings.Join(lines, "\n")
}
