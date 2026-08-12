package agent

import (
	"os"
	"strconv"
)

type AgentConfig struct {
	ID string

	Model string

	MaxSteps int

	Temperature float64

	MaxTokens int

	SystemPrompt   string
	PromptVariants []PromptVariant

	ToolChoice string

	ToolNames []string
}

// DefaultMaxSteps caps agent loops: each step is one LLM call plus its tool
// executions, so 100 is effectively unbounded for real work.
const DefaultMaxSteps = 100

// MaxStepsFromEnv returns the step cap honoring the VESVAI_MAX_STEPS
// environment override (0 = unlimited); falls back to DefaultMaxSteps.
func MaxStepsFromEnv() int {
	if v := os.Getenv("VESVAI_MAX_STEPS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return DefaultMaxSteps
}

func DefaultConfig() AgentConfig {
	return AgentConfig{
		Model:        "",
		MaxSteps:     MaxStepsFromEnv(),
		Temperature:  0.7,
		MaxTokens:    4096,
		SystemPrompt: "",
	}
}

type AgentOption func(*AgentConfig)

func WithModel(model string) AgentOption {
	return func(c *AgentConfig) {
		c.Model = model
	}
}

func WithMaxSteps(n int) AgentOption {
	return func(c *AgentConfig) {
		c.MaxSteps = n
	}
}

func WithTemperature(temp float64) AgentOption {
	return func(c *AgentConfig) {
		c.Temperature = temp
	}
}

func WithMaxTokens(n int) AgentOption {
	return func(c *AgentConfig) {
		c.MaxTokens = n
	}
}

func WithSystemPrompt(prompt string) AgentOption {
	return func(c *AgentConfig) {
		c.SystemPrompt = prompt
	}
}

func WithToolNames(names ...string) AgentOption {
	return func(c *AgentConfig) {
		c.ToolNames = names
	}
}

func WithPromptVariants(variants ...PromptVariant) AgentOption {
	return func(c *AgentConfig) {
		c.PromptVariants = variants
	}
}

func ApplyOptions(config *AgentConfig, opts ...AgentOption) {
	for _, opt := range opts {
		opt(config)
	}
}
