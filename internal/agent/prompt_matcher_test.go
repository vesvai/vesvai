package agent

import (
	"testing"
)

func TestMatchPattern_Exact(t *testing.T) {
	if !matchPattern("claude-3-opus", "claude-3-opus") {
		t.Error("exact match failed")
	}
	if matchPattern("claude-3-opus", "claude-3-sonnet") {
		t.Error("exact match should not match different value")
	}
}

func TestMatchPattern_Prefix(t *testing.T) {
	if !matchPattern("claude-*", "claude-3-opus") {
		t.Error("prefix match failed for claude-3-opus")
	}
	if !matchPattern("claude-*", "claude-instant") {
		t.Error("prefix match failed for claude-instant")
	}
	if matchPattern("claude-*", "gpt-4") {
		t.Error("prefix match should not match gpt-4")
	}
}

func TestMatchPattern_Suffix(t *testing.T) {
	if !matchPattern("*-opus", "claude-3-opus") {
		t.Error("suffix match failed for claude-3-opus")
	}
	if !matchPattern("*-opus", "gpt-4-opus") {
		t.Error("suffix match failed for gpt-4-opus")
	}
	if matchPattern("*-opus", "claude-3-sonnet") {
		t.Error("suffix match should not match claude-3-sonnet")
	}
}

func TestMatchPattern_Contains(t *testing.T) {
	if !matchPattern("*opus*", "claude-3-opus") {
		t.Error("contains match failed")
	}
	if !matchPattern("*opus*", "my-opus-model") {
		t.Error("contains match failed for my-opus-model")
	}
	if matchPattern("*opus*", "claude-3-sonnet") {
		t.Error("contains match should not match claude-3-sonnet")
	}
}

func TestMatchPattern_Regex(t *testing.T) {
	if !matchPattern("^claude-3-.*", "claude-3-opus") {
		t.Error("regex match failed for claude-3-opus")
	}
	if !matchPattern("^claude-3-.*", "claude-3-sonnet") {
		t.Error("regex match failed for claude-3-sonnet")
	}
	if matchPattern("^claude-3-.*", "claude-2-opus") {
		t.Error("regex match should not match claude-2-opus")
	}
}

func TestMatchPattern_ComplexRegex(t *testing.T) {
	if !matchPattern("^(claude|gpt)-4.*", "claude-4-opus") {
		t.Error("complex regex match failed for claude-4-opus")
	}
	if !matchPattern("^(claude|gpt)-4.*", "gpt-4-turbo") {
		t.Error("complex regex match failed for gpt-4-turbo")
	}
	if matchPattern("^(claude|gpt)-4.*", "gemini-pro") {
		t.Error("complex regex match should not match gemini-pro")
	}
}

func TestPromptMatcher_NoVariants(t *testing.T) {
	matcher := NewPromptMatcher("default prompt", nil)
	result := matcher.Match("any-model")
	if result != "default prompt" {
		t.Errorf("expected default prompt, got %s", result)
	}
}

func TestPromptMatcher_ModelMatch(t *testing.T) {
	variants := []PromptVariant{
		{Pattern: "claude-*", Prompt: "claude prompt"},
		{Pattern: "gemini-*", Prompt: "gemini prompt"},
	}

	matcher := NewPromptMatcher("default prompt", variants)

	if result := matcher.Match("claude-3-opus"); result != "claude prompt" {
		t.Errorf("expected claude prompt, got %s", result)
	}

	if result := matcher.Match("gemini-pro"); result != "gemini prompt" {
		t.Errorf("expected gemini prompt, got %s", result)
	}

	if result := matcher.Match("gpt-4"); result != "default prompt" {
		t.Errorf("expected default prompt, got %s", result)
	}
}

func TestPromptMatcher_FirstMatchWins(t *testing.T) {
	variants := []PromptVariant{
		{Pattern: "claude-*", Prompt: "first match"},
		{Pattern: "claude-3-*", Prompt: "second match"},
	}

	matcher := NewPromptMatcher("default prompt", variants)

	if result := matcher.Match("claude-3-opus"); result != "first match" {
		t.Errorf("expected first match, got %s", result)
	}
}
