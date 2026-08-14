package lmstudio

import (
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/llm/providers/compat"
)

const ProviderName = "lmstudio"

func RegisterHooks(hooks *hook.Hooks) {
	compat.Register(hooks, ProviderName, "http://localhost:1234/v1", compat.Options{AllowNoKey: true})
}
