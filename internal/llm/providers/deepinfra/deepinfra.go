package deepinfra

import (
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/llm/providers/compat"
)

const ProviderName = "deepinfra"

func RegisterHooks(hooks *hook.Hooks) {
	compat.Register(hooks, ProviderName, "https://api.deepinfra.com/v1/openai", compat.Options{})
}
