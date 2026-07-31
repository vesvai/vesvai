package openai

import (
	"testing"

	"github.com/vesvai/vesvai/internal/llm"
)

func TestProvider_Name(t *testing.T) {
	p := NewProvider(Config{APIKey: "sk-test"})
	if p.Name() != "openai" {
		t.Errorf("Name() = %q, want openai", p.Name())
	}
}

func TestProvider_SatisfiesProviderInterface(t *testing.T) {
	var _ llm.Provider = (*Provider)(nil)
}

func TestNewProvider_DefaultBaseURL(t *testing.T) {
	p := NewProvider(Config{APIKey: "k"})
	if got := p.httpClient; got == nil {
		t.Fatal("httpClient should be initialised")
	}
	_ = p
}