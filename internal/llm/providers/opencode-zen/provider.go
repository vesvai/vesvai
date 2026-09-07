package opencodezen

import (
	"maps"
	"time"

	"github.com/vesvai/vesvai/internal/core/config"
	"github.com/vesvai/vesvai/internal/llm"
	openaidriver "github.com/vesvai/vesvai/internal/llm/drivers/openai"
)

const (
	ProviderName   = "opencode-zen"
	DefaultBaseURL = "https://opencode.ai/zen/v1"
)

func init() {
	llm.RegisterProvider(ProviderName, NewFromConfig)
}

func NewFromConfig(cfg config.LLMConfig) (llm.Provider, error) {
	defaultHeaders := map[string]string{
		"User-Agent":         "opencode/1.18.29 ai-sdk/provider-utils/4.0.23 runtime/bun/1.3.14",
		"x-opencode-client":  "vesvai",
		"x-opencode-project": "vesvai",
		"x-opencode-request": "vesvai",
		"x-opencode-session": "vesvai",
	}
	maps.Copy(defaultHeaders, cfg.Headers)
	return openaidriver.NewService(ProviderName, openaidriver.ServiceConfig{
		BaseURL: DefaultBaseURL,
		APIKey:  cfg.APIKey,
		Headers: defaultHeaders,
		Timeout: time.Duration(cfg.Timeout) * time.Second,
	}), nil
}
