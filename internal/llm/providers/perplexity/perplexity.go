package perplexity

import (
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/llm/providers/compat"
)

const ProviderName = "perplexity"

func RegisterHooks(hooks *hook.Hooks) {
	compat.Register(hooks, ProviderName, "https://api.perplexity.ai", compat.Options{})
}
