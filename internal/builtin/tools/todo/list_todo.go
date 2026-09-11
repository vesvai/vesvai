package todo

import (
	"context"
	"fmt"

	"github.com/vesvai/vesvai/internal/agent/prompt"
	"github.com/vesvai/vesvai/internal/agent/tool"
	"github.com/vesvai/vesvai/internal/vfs"
)

func generateListTodoToolPrompt() (string, error) {
	sys, err := listTodoToolPromptBuilder().
		Build(prompt.FormatMarkdown)
	if err != nil {
		return "", err
	}
	return sys, nil
}

func listTodoTool(fs *vfs.VFS) tool.Tool {
	prompt, err := generateListTodoToolPrompt()
	if err != nil {
		panic(fmt.Sprintf("failed to generate list todo tool prompt: %v", err))
	}

	return tool.NewSpec(
		"list-todo",
		prompt,
		map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []string{},
		},
		func(ctx context.Context, args string) (string, error) {
			all, err := store.all(ctx)
			if err != nil {
				return "", fmt.Errorf("list-todo: %w", err)
			}
			result := formatTodoList(all)
			return result, nil
		},
	)
}
