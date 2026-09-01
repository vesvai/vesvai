package todo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vesvai/vesvai/internal/agent/tool"
	"github.com/vesvai/vesvai/internal/vfs"
)

func listTodoTool(fs *vfs.VFS) tool.Tool {
	return tool.NewSpec(
		"list-todo",
		"List todos from the project's persistent todo list (stored in .vesvai/todos.json). Returns all todos sorted by creation date (newest first), showing status, ID, title, priority, description, and dependencies. Optionally filter by status (pending, in_progress, completed, cancelled) and/or priority (high, medium, low). Use this tool to view what work items exist and their current state.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"status": map[string]any{
					"type":        "string",
					"description": "Filter by status: 'pending', 'in_progress', 'completed', or 'cancelled'. Omit to show all.",
				},
				"priority": map[string]any{
					"type":        "string",
					"description": "Filter by priority: 'high', 'medium', or 'low'. Omit to show all.",
				},
			},
			"required": []string{},
		},
		func(ctx context.Context, args string) (string, error) {
			var params struct {
				Status   string `json:"status"`
				Priority string `json:"priority"`
			}
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", fmt.Errorf("list-todo: invalid arguments: %w", err)
			}
			if params.Status != "" && !validStatus(params.Status) {
				return "", fmt.Errorf("list-todo: invalid status %q (must be: pending, in_progress, completed, cancelled)", params.Status)
			}
			if params.Priority != "" && !validPriority(params.Priority) {
				return "", fmt.Errorf("list-todo: invalid priority %q (must be: high, medium, low)", params.Priority)
			}

			all, err := store.all()
			if err != nil {
				return "", fmt.Errorf("list-todo: %w", err)
			}

			var filtered []*Todo
			for _, t := range all {
				if params.Status != "" && t.Status != params.Status {
					continue
				}
				if params.Priority != "" && t.Priority != params.Priority {
					continue
				}
				filtered = append(filtered, t)
			}

			header := "Todos:"
			if params.Status != "" {
				header += fmt.Sprintf(" status=%s", params.Status)
			}
			if params.Priority != "" {
				header += fmt.Sprintf(" priority=%s", params.Priority)
			}

			result := formatTodoList(filtered)
			return header + "\n" + result, nil
		},
	)
}
