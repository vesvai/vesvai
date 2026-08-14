package sambanova

import (
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/llm/providers/compat"
)

const ProviderName = "sambanova"

func RegisterHooks(hooks *hook.Hooks) {
	compat.Register(hooks, ProviderName, "https://api.sambanova.ai/v1", compat.Options{})
}
