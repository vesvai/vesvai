package cloudflare

import (
	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/llm/providers/compat"
)

const ProviderName = "cloudflare"

func RegisterHooks(hooks *hook.Hooks) {
	compat.Register(hooks, ProviderName, "", compat.Options{ExampleBaseURL: "https://api.cloudflare.com/client/v4/accounts/{account_id}/ai/v1"})
}
