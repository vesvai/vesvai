package agent

type AgentConfig struct {
	ID string

	Model string

	MaxSteps int

	Temperature float64

	MaxTokens int

	SystemPrompt string

	ToolChoice string
}

func DefaultConfig() AgentConfig {
	return AgentConfig{
		Model:        "",
		MaxSteps:     10,
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

func ApplyOptions(config *AgentConfig, opts ...AgentOption) {
	for _, opt := range opts {
		opt(config)
	}
}
