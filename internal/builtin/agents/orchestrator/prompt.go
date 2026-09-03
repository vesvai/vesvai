package orchestrator

import (
	"github.com/vesvai/vesvai/internal/agent/prompt"
	"github.com/vesvai/vesvai/internal/builtin/agents/shared"
)

func generateOrchestratorPrompt() (string, error) {
	sys, err := shared.SharedPromptBuilder().
		Build(prompt.FormatMarkdown)
	if err != nil {
		return "", err
	}
	return sys, nil
}
