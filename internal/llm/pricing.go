package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const LitellmPricesURL = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"

const PricesCacheKey = "litellm_prices"

var ErrModelConfigNotFound = errors.New("llm: model config not found")

type ModelConfig struct {
	MaxTokens        int      `json:"max_tokens,omitempty"`
	MaxInputTokens   int      `json:"max_input_tokens,omitempty"`
	MaxOutputTokens  int      `json:"max_output_tokens,omitempty"`
	LitellmProvider  string   `json:"litellm_provider,omitempty"`
	Mode             string   `json:"mode,omitempty"`
	DeprecationDate  string   `json:"deprecation_date,omitempty"`
	SupportedRegions []string `json:"supported_regions,omitempty"`

	InputCostPerToken           float64 `json:"input_cost_per_token,omitempty"`
	OutputCostPerToken          float64 `json:"output_cost_per_token,omitempty"`
	InputCostPerAudioToken      float64 `json:"input_cost_per_audio_token,omitempty"`
	OutputCostPerReasoningToken float64 `json:"output_cost_per_reasoning_token,omitempty"`
	VectorStoreCostPerGBPerDay  float64 `json:"vector_store_cost_per_gb_per_day,omitempty"`
	SearchContextCostPerQuery   any     `json:"search_context_cost_per_query,omitempty"`

	SupportsAudioInput              bool `json:"supports_audio_input,omitempty"`
	SupportsAudioOutput             bool `json:"supports_audio_output,omitempty"`
	SupportsFunctionCalling         bool `json:"supports_function_calling,omitempty"`
	SupportsParallelFunctionCalling bool `json:"supports_parallel_function_calling,omitempty"`
	SupportsPromptCaching           bool `json:"supports_prompt_caching,omitempty"`
	SupportsReasoning               bool `json:"supports_reasoning,omitempty"`
	SupportsResponseSchema          bool `json:"supports_response_schema,omitempty"`
	SupportsSystemMessages          bool `json:"supports_system_messages,omitempty"`
	SupportsVision                  bool `json:"supports_vision,omitempty"`
	SupportsWebSearch               bool `json:"supports_web_search,omitempty"`

	ReasoningEfforts []string `json:"reasoning_efforts,omitempty"`
}

func (c *ModelConfig) UnmarshalJSON(data []byte) error {
	type plain ModelConfig
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		normalized, nerr := coerceIntFloats(data)
		if nerr != nil {
			return err
		}
		if err := json.Unmarshal(normalized, &p); err != nil {
			return err
		}
	}
	*c = ModelConfig(p)
	return nil
}

func coerceIntFloats(data []byte) ([]byte, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	for _, key := range []string{"max_tokens", "max_input_tokens", "max_output_tokens"} {
		raw, ok := m[key]
		if !ok {
			continue
		}
		var f float64
		if err := json.Unmarshal(raw, &f); err != nil {
			continue
		}
		if f == math.Trunc(f) {
			m[key] = []byte(strconv.FormatInt(int64(f), 10))
		}
	}
	return json.Marshal(m)
}

var ReasoningEffortVariants = map[string][]string{
	"o1":           {"low", "medium", "high"},
	"o1-mini":      {"low", "medium", "high"},
	"o1-preview":   {"low", "medium", "high"},
	"o1-pro":       {"low", "medium", "high"},
	"o3":           {"low", "medium", "high"},
	"o3-mini":      {"low", "medium", "high"},
	"o3-pro":       {"low", "medium", "high"},
	"o4-mini":      {"low", "medium", "high"},
	"o4-mini-high": {"low", "medium", "high"},

	"gpt-5":              {"minimal", "low", "medium", "high"},
	"gpt-5-mini":         {"minimal", "low", "medium", "high"},
	"gpt-5-nano":         {"minimal", "low", "medium", "high"},
	"gpt-5-codex":        {"low", "medium", "high"},
	"gpt-5-pro":          {"high"},
	"gpt-5.1":            {"none", "low", "medium", "high"},
	"gpt-5.1-codex":      {"low", "medium", "high"},
	"gpt-5.1-codex-mini": {"low", "medium", "high"},
	"gpt-5.1-codex-max":  {"low", "medium", "high", "xhigh"},
	"gpt-5.2":            {"none", "low", "medium", "high", "xhigh"},
	"gpt-5.2-codex":      {"low", "medium", "high", "xhigh"},
	"gpt-5.2-pro":        {"medium", "high", "xhigh"},
	"gpt-5.4":            {"none", "low", "medium", "high", "xhigh"},
	"gpt-5.5":            {"none", "low", "medium", "high", "xhigh"},
	"gpt-5.5-pro":        {"medium", "high", "xhigh"},
	"gpt-5.6":            {"none", "low", "medium", "high", "xhigh", "max"},

	"claude-3-5-sonnet": {"low", "medium", "high"},
	"claude-3-7-sonnet": {"low", "medium", "high"},
	"claude-sonnet-4":   {"low", "medium", "high"},
	"claude-opus-4":     {"low", "medium", "high"},
	"claude-haiku-4":    {"low", "medium", "high"},
	"claude-sonnet-4-5": {"low", "medium", "high"},
	"claude-opus-4-5":   {"low", "medium", "high"},
	"claude-haiku-4-5":  {"low", "medium", "high"},

	"gemini-2.5-pro":         {"low", "medium", "high"},
	"gemini-2.5-flash":       {"low", "medium", "high"},
	"gemini-2.5-flash-lite":  {"low", "medium", "high"},
	"gemini-3-pro-preview":   {"low", "high"},
	"gemini-3-flash-preview": {"minimal", "low", "medium", "high"},
	"gemini-3.1-pro-preview": {"low", "medium", "high"},
	"gemini-3.5-flash":       {"minimal", "low", "medium", "high"},
	"gemini-3.6-flash":       {"minimal", "low", "medium", "high"},
	"gemini-3.7-flash":       {"low", "medium", "high"},

	"deepseek-r1":         {"low", "medium", "high"},
	"deepseek-r1-zero":    {"medium", "high"},
	"deepseek-r1-distill": {"low", "medium", "high"},
	"deepseek-r2":         {"low", "medium", "high", "xhigh"},
	"deepseek-reasoner":   {"low", "medium", "high"},
	"deepseek-v4":         {"none", "low", "medium", "high", "max"},
	"deepseek-v4-flash":   {"none", "low", "medium", "high", "max"},
	"deepseek-v4-pro":     {"none", "low", "medium", "high", "max"},

	"grok-3":       {"low", "medium", "high"},
	"grok-3-mini":  {"low", "medium", "high"},
	"grok-3-think": {"low", "medium", "high", "max"},
	"grok-4":       {"none", "low", "medium", "high", "xhigh"},

	"qwq-32b":          {"low", "medium", "high"},
	"qwen-3-reasoner":  {"low", "medium", "high"},
	"qwen-3-max-think": {"low", "medium", "high", "xhigh"},

	"llama-4-scout":    {"low", "medium"},
	"llama-4-maverick": {"low", "medium", "high"},
	"llama-4-thinker":  {"low", "medium", "high", "xhigh"},

	"sonar-reasoning":     {"low", "medium", "high"},
	"sonar-reasoning-pro": {"low", "medium", "high", "xhigh"},
}

func reasoningEffortsFor(key string) []string {
	if v, ok := ReasoningEffortVariants[key]; ok {
		return v
	}
	model := key
	if idx := strings.LastIndex(key, "/"); idx >= 0 {
		model = key[idx+1:]
	}
	if v, ok := ReasoningEffortVariants[model]; ok {
		return v
	}
	bases := make([]string, 0, len(ReasoningEffortVariants))
	for base := range ReasoningEffortVariants {
		bases = append(bases, base)
	}
	sort.Slice(bases, func(i, j int) bool { return len(bases[i]) > len(bases[j]) })
	for _, base := range bases {
		if strings.HasPrefix(model, base) {
			return ReasoningEffortVariants[base]
		}
	}
	return nil
}

func FetchPrices(ctx context.Context) (map[string]ModelConfig, error) {
	return fetchPricesFrom(ctx, LitellmPricesURL)
}

func fetchPricesFrom(ctx context.Context, url string) (map[string]ModelConfig, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("llm: create prices request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm: fetch prices: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("llm: fetch prices: status %d", resp.StatusCode)
	}

	dec := json.NewDecoder(resp.Body)
	if _, err := dec.Token(); err != nil {
		return nil, fmt.Errorf("llm: decode prices: %w", err)
	}

	out := make(map[string]ModelConfig)
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("llm: decode prices key: %w", err)
		}
		name, _ := key.(string)
		if name == "sample_spec" {
			var skip any
			if err := dec.Decode(&skip); err != nil {
				return nil, fmt.Errorf("llm: skip prices entry %q: %w", name, err)
			}
			continue
		}
		var cfg ModelConfig
		if err := dec.Decode(&cfg); err != nil {
			return nil, fmt.Errorf("llm: decode prices entry %q: %w", name, err)
		}
		if cfg.Mode != "chat" {
			continue
		}
		if efforts := reasoningEffortsFor(name); len(efforts) > 0 {
			cfg.ReasoningEfforts = efforts
		}
		out[name] = cfg
	}
	return out, nil
}

func lookupModelConfig(prices map[string]ModelConfig, provider, model string) (*ModelConfig, error) {
	if cfg, ok := prices[provider+"/"+model]; ok {
		return &cfg, nil
	}
	if cfg, ok := prices[model]; ok {
		return &cfg, nil
	}
	keys := make([]string, 0, len(prices))
	for key := range prices {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if strings.Contains(key, model) {
			cfg := prices[key]
			return &cfg, nil
		}
	}
	return nil, ErrModelConfigNotFound
}
