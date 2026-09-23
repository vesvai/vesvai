package opencodezen

import (
	"fmt"
	"maps"
	"time"

	"github.com/vesvai/vesvai/internal/core/config"
	"github.com/vesvai/vesvai/internal/llm"
	openaidriver "github.com/vesvai/vesvai/internal/llm/drivers/openai"
	"github.com/vesvai/vesvai/internal/utils/random"
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
		"User-Agent":         "opencode/1.18.31 ai-sdk/provider-utils/4.0.23 runtime/bun/1.3.14",
		"x-opencode-client":  "cli",
		"x-opencode-project": random.GenerateHex(20),
		"x-opencode-request": fmt.Sprintf("msg_%s", random.GenerateAlphanumeric(26)),
		"x-opencode-session": fmt.Sprintf("ses_%s%s", random.GenerateHex(12), random.GenerateAlphanumeric(14)),
	}
	maps.Copy(defaultHeaders, cfg.Headers)
	return openaidriver.NewService(ProviderName, openaidriver.ServiceConfig{
		BaseURL: DefaultBaseURL,
		APIKey:  cfg.APIKey,
		Headers: defaultHeaders,
		Timeout: time.Duration(cfg.Timeout) * time.Second,
	}), nil
}
