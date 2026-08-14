package all

import (
	"context"
	"testing"

	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/llm"
)

func TestRegisterAllProviders(t *testing.T) {
	hooks := hook.New(nil)
	RegisterAll(hooks)

	registry := llm.NewProviderRegistry()
	result := hooks.ApplyFilter(context.Background(), llm.HookProviderRegistry, registry)
	got, ok := result.(*llm.ProviderRegistry)
	if !ok {
		t.Fatal("filter did not return registry")
	}

	names := map[string]bool{}
	for _, n := range got.Supported() {
		names[n] = true
	}

	for _, want := range []string{
		"openai", "openrouter",
		"deepseek", "groq", "mistral", "together", "fireworks", "perplexity",
		"xai", "cerebras", "sambanova", "nvidia", "deepinfra", "lambdalabs",
		"friendliai", "hyperbolic", "featherless", "lepton", "novita",
		"siliconflow", "cloudflare", "huggingface",
		"moonshot", "zhipu", "dashscope", "qianfan", "hunyuan", "volcengine",
		"azure", "github", "gemini", "vertex", "watsonx", "gaudi",
		"ollama", "lmstudio", "localai", "vllm", "tgi",
		"opencode", "zen",
	} {
		if !names[want] {
			t.Errorf("provider %q not registered", want)
		}
	}
}
