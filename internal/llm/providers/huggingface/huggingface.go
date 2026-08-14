package huggingface

import (
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/llm/providers/compat"
)

const ProviderName = "huggingface"

func RegisterHooks(hooks *hook.Hooks) {
	compat.Register(hooks, ProviderName, "https://router.huggingface.co/v1", compat.Options{})
}
