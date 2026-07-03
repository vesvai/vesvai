package agent

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxSteps != 10 {
		t.Errorf("MaxSteps = %d, want 10", cfg.MaxSteps)
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

	if cfg != original {
		t.Error("ApplyOptions with no options changed config")
	}
}

func TestAgentConfig_Fields(t *testing.T) {
	cfg := AgentConfig{
		ID:            "agent-1",
		Model:         "gpt-4",
		MaxSteps:      15,
		Temperature:   0.5,
		MaxTokens:     1024,
		SystemPrompt:  "System",
		ToolChoice:    "auto",
	}

	if cfg.ID != "agent-1" {
		t.Errorf("ID = %q, want agent-1", cfg.ID)
	}
	if cfg.ToolChoice != "auto" {
		t.Errorf("ToolChoice = %q, want auto", cfg.ToolChoice)
	}
}
