package ollama

import (
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/llm/providers/compat"
)

const ProviderName = "ollama"

func RegisterHooks(hooks *hook.Hooks) {
	compat.Register(hooks, ProviderName, "http://localhost:11434/v1", compat.Options{AllowNoKey: true})
}
