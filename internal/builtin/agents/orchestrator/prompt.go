package orchestrator

import (
	"github.com/vesvai/vesvai/internal/agent/prompt"
	"github.com/vesvai/vesvai/internal/builtin/agents/shared"
)

func generateOrchestratorPrompt(providerID, modelID string) (string, error) {
	sys, err := shared.SharedPromptBuilder(providerID, modelID).
		Build(prompt.FormatMarkdown)
	if err != nil {
		return "", err
	}
	return sys, nil
}
