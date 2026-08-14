package azure

import (
	"net/url"

	"github.com/vesvai/vesvai/internal/hook"
	"github.com/vesvai/vesvai/internal/llm/providers/compat"
)

const ProviderName = "azure"

const apiVersion = "2024-06-01"

func RegisterHooks(hooks *hook.Hooks) {
	compat.Register(hooks, ProviderName, "", compat.Options{
		KeyHeader:      "api-key",
		ExampleBaseURL: "https://my-resource.openai.azure.com/openai/v1",
		PathFor: func(endpoint, model string) string {
			switch endpoint {
			case "/models":
				return "/deployments?api-version=" + apiVersion
			default:
				return "/deployments/" + url.PathEscape(model) + "/chat/completions?api-version=" + apiVersion
			}
		},
	})
}
