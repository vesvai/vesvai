package moonshot

import (
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/llm/providers/compat"
)

const ProviderName = "moonshot"

func RegisterHooks(hooks *hook.Hooks) {
	compat.Register(hooks, ProviderName, "https://api.moonshot.ai/v1", compat.Options{})
}
