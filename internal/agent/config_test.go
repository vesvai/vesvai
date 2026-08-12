package agent

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxSteps != DefaultMaxSteps {
		t.Errorf("MaxSteps = %d, want %d", cfg.MaxSteps, DefaultMaxSteps)
	}
	if cfg.Temperature != 0.7 {
		t.Errorf("Temperature = %f, want 0.7", cfg.Temperature)
	}
	if cfg.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want 4096", cfg.MaxTokens)
	}
	if cfg.SystemPrompt != "" {
		t.Errorf("SystemPrompt = %q, want empty", cfg.SystemPrompt)
	}
	if cfg.Model != "" {
		t.Errorf("Model = %q, want empty", cfg.Model)
	}
	if cfg.ToolChoice != "" {
		t.Errorf("ToolChoice = %q, want empty", cfg.ToolChoice)
	}
}

func TestMaxStepsFromEnv(t *testing.T) {
	t.Setenv("VESVAI_MAX_STEPS", "")
	if got := MaxStepsFromEnv(); got != DefaultMaxSteps {
		t.Errorf("unset env: MaxStepsFromEnv = %d, want %d", got, DefaultMaxSteps)
	}
	t.Setenv("VESVAI_MAX_STEPS", "500")
	if got := MaxStepsFromEnv(); got != 500 {
		t.Errorf("env 500: MaxStepsFromEnv = %d, want 500", got)
	}
	t.Setenv("VESVAI_MAX_STEPS", "0")
	if got := MaxStepsFromEnv(); got != 0 {
		t.Errorf("env 0 (unlimited): MaxStepsFromEnv = %d, want 0", got)
	}
	t.Setenv("VESVAI_MAX_STEPS", "abc")
	if got := MaxStepsFromEnv(); got != DefaultMaxSteps {
		t.Errorf("invalid env: MaxStepsFromEnv = %d, want %d", got, DefaultMaxSteps)
	}
}

func TestWithModel(t *testing.T) {
	cfg := DefaultConfig()
	ApplyOptions(&cfg, WithModel("gpt-4"))

	if cfg.Model != "gpt-4" {
		t.Errorf("Model = %q, want %q", cfg.Model, "gpt-4")
	}
}

func TestWithMaxSteps(t *testing.T) {
	cfg := DefaultConfig()
	ApplyOptions(&cfg, WithMaxSteps(20))

	if cfg.MaxSteps != 20 {
		t.Errorf("MaxSteps = %d, want 20", cfg.MaxSteps)
	}
}

func TestWithTemperature(t *testing.T) {
	cfg := DefaultConfig()
	ApplyOptions(&cfg, WithTemperature(0.0))

	if cfg.Temperature != 0.0 {
		t.Errorf("Temperature = %f, want 0.0", cfg.Temperature)
	}
}

func TestWithMaxTokens(t *testing.T) {
	cfg := DefaultConfig()
	ApplyOptions(&cfg, WithMaxTokens(8192))

	if cfg.MaxTokens != 8192 {
		t.Errorf("MaxTokens = %d, want 8192", cfg.MaxTokens)
	}
}

func TestWithSystemPrompt(t *testing.T) {
	cfg := DefaultConfig()
	ApplyOptions(&cfg, WithSystemPrompt("You are helpful."))

	if cfg.SystemPrompt != "You are helpful." {
		t.Errorf("SystemPrompt = %q, want %q", cfg.SystemPrompt, "You are helpful.")
	}
}

func TestApplyOptions_Multiple(t *testing.T) {
	cfg := DefaultConfig()
	ApplyOptions(&cfg,
		WithModel("claude-3"),
		WithMaxSteps(5),
		WithTemperature(1.0),
		WithMaxTokens(2048),
		WithSystemPrompt("Be concise."),
	)

	if cfg.Model != "claude-3" {
		t.Errorf("Model = %q, want claude-3", cfg.Model)
	}
	if cfg.MaxSteps != 5 {
		t.Errorf("MaxSteps = %d, want 5", cfg.MaxSteps)
	}
	if cfg.Temperature != 1.0 {
		t.Errorf("Temperature = %f, want 1.0", cfg.Temperature)
	}
	if cfg.MaxTokens != 2048 {
		t.Errorf("MaxTokens = %d, want 2048", cfg.MaxTokens)
	}
	if cfg.SystemPrompt != "Be concise." {
		t.Errorf("SystemPrompt = %q, want %q", cfg.SystemPrompt, "Be concise.")
	}
}

func TestApplyOptions_Empty(t *testing.T) {
	cfg := DefaultConfig()
	original := cfg

	ApplyOptions(&cfg)

	if cfg.Model != original.Model {
		t.Error("ApplyOptions with no options changed Model")
	}
	if cfg.MaxSteps != original.MaxSteps {
		t.Error("ApplyOptions with no options changed MaxSteps")
	}
	if cfg.Temperature != original.Temperature {
		t.Error("ApplyOptions with no options changed Temperature")
	}
	if cfg.MaxTokens != original.MaxTokens {
		t.Error("ApplyOptions with no options changed MaxTokens")
	}
	if cfg.SystemPrompt != original.SystemPrompt {
		t.Error("ApplyOptions with no options changed SystemPrompt")
	}
	if cfg.ToolChoice != original.ToolChoice {
		t.Error("ApplyOptions with no options changed ToolChoice")
	}
}

func TestAgentConfig_Fields(t *testing.T) {
	cfg := AgentConfig{
		ID:           "agent-1",
		Model:        "gpt-4",
		MaxSteps:     15,
		Temperature:  0.5,
		MaxTokens:    1024,
		SystemPrompt: "System",
		ToolChoice:   "auto",
	}

	if cfg.ID != "agent-1" {
		t.Errorf("ID = %q, want agent-1", cfg.ID)
	}
	if cfg.ToolChoice != "auto" {
		t.Errorf("ToolChoice = %q, want auto", cfg.ToolChoice)
	}
}
