package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/utils/http"
)

const (
	BaseURL = "https://openrouter.ai/api/v1"
)

type Config struct {
	APIKey string
}

type Client struct {
	httpClient *http.Client
}

func NewClient(config Config) *Client {
	return &Client{
		httpClient: http.NewClient(
			BaseURL,
			http.WithAPIKey(config.APIKey),
			http.WithHeader("HTTP-Referer", "https://github.com/vesvai/vesv"),
			http.WithHeader("X-Title", "Vesva"),
		),
	}
}

func (c *Client) Name() string {
	return "openrouter"
}

type OpenRouterMessage struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	Reasoning  json.RawMessage `json:"reasoning,omitempty"`
	Name       string          `json:"name,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	ToolCalls  []OpenRouterToolCall `json:"tool_calls,omitempty"`
}

type OpenRouterToolCall struct {
	ID       string              `json:"id"`
	Type     string              `json:"type"`
	Function OpenRouterFunction  `json:"function"`
}

type OpenRouterFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type OpenRouterRequest struct {
	Model             string              `json:"model"`
	Messages          []OpenRouterMessage `json:"messages"`
	Temperature       float64             `json:"temperature,omitempty"`
	TopP              float64             `json:"top_p,omitempty"`
	MaxTokens         int                 `json:"max_tokens,omitempty"`
	Stream            bool                `json:"stream,omitempty"`
	Tools             []llm.Tool          `json:"tools,omitempty"`
	ToolChoice        any                 `json:"tool_choice,omitempty"`
	ResponseFormat    *llm.ResponseFormat `json:"response_format,omitempty"`
	ParallelToolCalls bool                `json:"parallel_tool_calls,omitempty"`
	N                 int                 `json:"n,omitempty"`
	PresencePenalty   float64             `json:"presence_penalty,omitempty"`
	FrequencyPenalty  float64             `json:"frequency_penalty,omitempty"`
	User              string              `json:"user,omitempty"`
	Provider          *struct {
		AllowFallbacks bool `json:"allow_fallbacks,omitempty"`
	} `json:"provider,omitempty"`
}

type OpenRouterResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []OpenRouterChoice `json:"choices"`
	Usage   struct {
		PromptTokens        int     `json:"prompt_tokens"`
		CompletionTokens    int     `json:"completion_tokens"`
		TotalTokens         int     `json:"total_tokens"`
		Cost                float64 `json:"cost,omitempty"`
		IsByok              bool    `json:"is_byok,omitempty"`
		PromptTokensDetails *struct {
			CachedTokens     int `json:"cached_tokens,omitempty"`
			CacheWriteTokens int `json:"cache_write_tokens,omitempty"`
			AudioTokens      int `json:"audio_tokens,omitempty"`
			VideoTokens      int `json:"video_tokens,omitempty"`
		} `json:"prompt_tokens_details,omitempty"`
		CompletionTokensDetails *struct {
			ReasoningTokens int `json:"reasoning_tokens,omitempty"`
			ImageTokens     int `json:"image_tokens,omitempty"`
			AudioTokens     int `json:"audio_tokens,omitempty"`
		} `json:"completion_tokens_details,omitempty"`
		CostDetails *struct {
			UpstreamInferenceCost            float64 `json:"upstream_inference_cost,omitempty"`
			UpstreamInferencePromptCost      float64 `json:"upstream_inference_prompt_cost,omitempty"`
			UpstreamInferenceCompletionsCost float64 `json:"upstream_inference_completions_cost,omitempty"`
		} `json:"cost_details,omitempty"`
	} `json:"usage"`
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
}

type OpenRouterChoice struct {
	Index        int                `json:"index"`
	Message      *OpenRouterMessage `json:"message,omitempty"`
	Delta        *OpenRouterDelta   `json:"delta,omitempty"`
	FinishReason *string            `json:"finish_reason,omitempty"`
}

type OpenRouterDelta struct {
	Role      *string `json:"role,omitempty"`
	Content   *string `json:"content,omitempty"`
	Reasoning *string `json:"reasoning,omitempty"`
	ToolCalls []any   `json:"tool_calls,omitempty"`
	Refusal   *string `json:"refusal,omitempty"`
}

func (c *Client) Chat(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	openRouterReq := c.toOpenRouterRequest(req)

	var openRouterResp OpenRouterResponse
	err := c.httpClient.Do(ctx, "POST", "/chat/completions", openRouterReq, &openRouterResp)
	if err != nil {
		return nil, c.mapError(err)
	}

	return c.toLLMResponse(openRouterResp), nil
}

func (c *Client) ChatStream(ctx context.Context, req *llm.Request, handler llm.StreamHandler) error {
	openRouterReq := c.toOpenRouterRequest(req)
	openRouterReq.Stream = true

	err := c.httpClient.DoStream(ctx, "/chat/completions", openRouterReq, func(line []byte) error {
		event, data := http.ParseSSEvent(line)
		if event == "done" {
			return nil
		}
		if event != "data" || len(data) == 0 {
			return nil
		}

		var streamResp OpenRouterResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			return fmt.Errorf("failed to unmarshal stream response: %w", err)
		}

		chunk := c.toStreamChunk(streamResp)
		return handler(chunk)
	})

	if err != nil {
		return c.mapError(err)
	}

	return nil
}

func (c *Client) toOpenRouterRequest(req *llm.Request) *OpenRouterRequest {
	openRouterReq := &OpenRouterRequest{
		Model:             req.Model,
		Temperature:       req.Temperature,
		TopP:              req.TopP,
		MaxTokens:         req.MaxTokens,
		Stream:            req.Stream,
		ToolChoice:        req.ToolChoice,
		ParallelToolCalls: req.ParallelToolCalls,
		N:                 req.N,
		PresencePenalty:   req.PresencePenalty,
		FrequencyPenalty:  req.FrequencyPenalty,
		User:              req.User,
		ResponseFormat:    req.ResponseFormat,
		Provider: &struct {
			AllowFallbacks bool `json:"allow_fallbacks,omitempty"`
		}{AllowFallbacks: true},
	}

	for _, msg := range req.Messages {
		openRouterMsg := OpenRouterMessage{
			Role:       string(msg.Role),
			Name:       msg.Name,
			ToolCallID: msg.ToolCallID,
		}

		switch content := msg.Content.(type) {
		case string:
			openRouterMsg.Content = json.RawMessage(escapeJSONString(content))
		case llm.Content:
			openRouterMsg.Content = c.contentToOpenRouter(content)
		case map[string]any:
			contentJSON, _ := json.Marshal(content)
			openRouterMsg.Content = contentJSON
		}

		if len(msg.ToolCalls) > 0 {
			openRouterMsg.ToolCalls = make([]OpenRouterToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				openRouterMsg.ToolCalls[j] = OpenRouterToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: OpenRouterFunction{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		openRouterReq.Messages = append(openRouterReq.Messages, openRouterMsg)
	}

	openRouterReq.Tools = req.Tools

	return openRouterReq
}

func (c *Client) contentToOpenRouter(content llm.Content) json.RawMessage {
	var blocks []map[string]any

	if content.Text != "" {
		blocks = append(blocks, map[string]any{
			"type": "text",
			"text": content.Text,
		})
	}

	for _, att := range content.Attachments {
		if att.Type == llm.AttachmentTypeImage {
			var url string
			if att.URL != "" {
				url = att.URL
			} else if att.Data != "" {
				url = "data:" + att.MediaType + ";base64," + att.Data
			}
			if url != "" {
				blocks = append(blocks, map[string]any{
					"type": "image_url",
					"image_url": map[string]any{
						"url": url,
					},
				})
			}
		}
	}

	if len(blocks) == 0 {
		return json.RawMessage(`""`)
	}

	result, _ := json.Marshal(blocks)
	return result
}

func (c *Client) toLLMResponse(resp OpenRouterResponse) *llm.Response {
	choices := make([]llm.Choice, len(resp.Choices))
	for i, choice := range resp.Choices {
		var msg *llm.Message
		if choice.Message != nil {
			msg = &llm.Message{
				Role:      llm.Role(choice.Message.Role),
				Content:   string(choice.Message.Content),
				Reasoning: string(choice.Message.Reasoning),
				Name:      choice.Message.Name,
			}

			if len(choice.Message.ToolCalls) > 0 {
				msg.ToolCalls = make([]llm.ToolCall, len(choice.Message.ToolCalls))
				for j, tc := range choice.Message.ToolCalls {
					msg.ToolCalls[j] = llm.ToolCall{
						ID:   tc.ID,
						Type: tc.Type,
						Function: llm.Function{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
				}
			}
		}

		var finishReason llm.FinishReason
		if choice.FinishReason != nil {
			finishReason = llm.FinishReason(*choice.FinishReason)
		}

		choices[i] = llm.Choice{
			Index:        choice.Index,
			Message:      msg,
			FinishReason: &finishReason,
		}
	}

	return &llm.Response{
		ID:      resp.ID,
		Object:  resp.Object,
		Created: resp.Created,
		Model:   resp.Model,
		Choices: choices,
		Usage: llm.Usage{
			PromptTokens:            resp.Usage.PromptTokens,
			CompletionTokens:        resp.Usage.CompletionTokens,
			TotalTokens:             resp.Usage.TotalTokens,
			Cost:                    resp.Usage.Cost,
			IsByok:                  resp.Usage.IsByok,
			PromptTokensDetails:     mapPromptTokensDetails(resp.Usage.PromptTokensDetails),
			CompletionTokensDetails: mapCompletionTokensDetails(resp.Usage.CompletionTokensDetails),
			CostDetails:             mapCostDetails(resp.Usage.CostDetails),
		},
		SystemFingerprint: resp.SystemFingerprint,
	}
}

func (c *Client) toStreamChunk(resp OpenRouterResponse) llm.StreamChunk {
	chunk := llm.StreamChunk{IsDone: false}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]

		if choice.Delta != nil {
			if choice.Delta.Content != nil {
				chunk.Content = *choice.Delta.Content
			}
			if choice.Delta.Reasoning != nil {
				chunk.Reasoning = *choice.Delta.Reasoning
			}
			if len(choice.Delta.ToolCalls) > 0 {
				for _, tc := range choice.Delta.ToolCalls {
					if tcMap, ok := tc.(map[string]any); ok {
						call := llm.ToolCall{
							Type: "function",
						}
						if idx, ok := tcMap["index"].(float64); ok {
							call.Index = int(idx)
						}
						if id, ok := tcMap["id"].(string); ok {
							call.ID = id
						}
						if fn, ok := tcMap["function"].(map[string]any); ok {
							if name, ok := fn["name"].(string); ok {
								call.Function.Name = name
							}
							if args, ok := fn["arguments"].(string); ok {
								call.Function.Arguments = args
							}
						}
						chunk.ToolCalls = append(chunk.ToolCalls, call)
					}
				}
			}
		}

		if choice.FinishReason != nil {
			chunk.FinishReason = llm.FinishReason(*choice.FinishReason)
			if *choice.FinishReason != "null" {
				chunk.IsDone = true
			}
		}
	}

	if resp.Usage.TotalTokens > 0 {
		chunk.Usage = &llm.Usage{
			PromptTokens:            resp.Usage.PromptTokens,
			CompletionTokens:        resp.Usage.CompletionTokens,
			TotalTokens:             resp.Usage.TotalTokens,
			Cost:                    resp.Usage.Cost,
			IsByok:                  resp.Usage.IsByok,
			PromptTokensDetails:     mapPromptTokensDetails(resp.Usage.PromptTokensDetails),
			CompletionTokensDetails: mapCompletionTokensDetails(resp.Usage.CompletionTokensDetails),
			CostDetails:             mapCostDetails(resp.Usage.CostDetails),
		}
	}

	return chunk
}

func (c *Client) mapError(err error) error {
	if httpErr, ok := err.(*http.HTTPError); ok {
		return &llm.ProviderError{
			StatusCode: httpErr.StatusCode,
			Message:    httpErr.Body,
			Body:       httpErr.Body,
		}
	}
	return err
}

func escapeJSONString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func mapPromptTokensDetails(details *struct {
	CachedTokens     int `json:"cached_tokens,omitempty"`
	CacheWriteTokens int `json:"cache_write_tokens,omitempty"`
	AudioTokens      int `json:"audio_tokens,omitempty"`
	VideoTokens      int `json:"video_tokens,omitempty"`
}) *llm.PromptTokensDetails {
	if details == nil {
		return nil
	}
	return &llm.PromptTokensDetails{
		CachedTokens:     details.CachedTokens,
		CacheWriteTokens: details.CacheWriteTokens,
		AudioTokens:      details.AudioTokens,
		VideoTokens:      details.VideoTokens,
	}
}

func mapCompletionTokensDetails(details *struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
	ImageTokens     int `json:"image_tokens,omitempty"`
	AudioTokens     int `json:"audio_tokens,omitempty"`
}) *llm.CompletionTokensDetails {
	if details == nil {
		return nil
	}
	return &llm.CompletionTokensDetails{
		ReasoningTokens: details.ReasoningTokens,
		ImageTokens:     details.ImageTokens,
		AudioTokens:     details.AudioTokens,
	}
}

func mapCostDetails(details *struct {
	UpstreamInferenceCost            float64 `json:"upstream_inference_cost,omitempty"`
	UpstreamInferencePromptCost      float64 `json:"upstream_inference_prompt_cost,omitempty"`
	UpstreamInferenceCompletionsCost float64 `json:"upstream_inference_completions_cost,omitempty"`
}) *llm.CostDetails {
	if details == nil {
		return nil
	}
	return &llm.CostDetails{
		UpstreamInferenceCost:            details.UpstreamInferenceCost,
		UpstreamInferencePromptCost:      details.UpstreamInferencePromptCost,
		UpstreamInferenceCompletionsCost: details.UpstreamInferenceCompletionsCost,
	}
}

func GetAPIKeyFromEnv() string {
	return os.Getenv("OPENROUTER_API_KEY")
}
