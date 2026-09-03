package skills

import (
	"github.com/vesvai/vesvai/internal/agent/prompt"
)

func BatchSkill() *prompt.Prompt {
	return prompt.New().
		Hr(3).
		Paragraph("name: batch").
		Paragraph("description: Instructs the /batch command. Instructions for orchestrating a large, parallelizable change across a codebase. Use it to make a change to many files at once, such as building an application from scratch, or performing a major refactoring. The /batch command is designed for large-scale code modifications that can be executed in parallel, ensuring efficiency and consistency across the codebase.").
		Hr(3)
}
