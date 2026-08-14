package tgi

import (
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/llm/providers/compat"
)

const ProviderName = "tgi"

func RegisterHooks(hooks *hook.Hooks) {
	compat.Register(hooks, ProviderName, "http://localhost:3000/v1", compat.Options{AllowNoKey: true})
}
