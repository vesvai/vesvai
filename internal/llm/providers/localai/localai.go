package localai

import (
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/llm/providers/compat"
)

const ProviderName = "localai"

func RegisterHooks(hooks *hook.Hooks) {
	compat.Register(hooks, ProviderName, "http://localhost:8080/v1", compat.Options{AllowNoKey: true})
}
