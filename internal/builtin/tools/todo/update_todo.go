package todo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vesvai/vesvai/internal/agent/tool"
	"github.com/vesvai/vesvai/internal/vfs"
)

func updateTodoTool(fs *vfs.VFS) tool.Tool {
	return tool.NewSpec(
		"update-todo",
		"Create, update, delete, or change the status of a todo in the project's persistent todo list (stored in .vesvai/todos.json). Todos are used for persistent cross-session task tracking. Actions: set 'id' to an existing ID to update/delete/complete that todo, provide a custom ID to create a new todo with that ID, or omit 'id' to create a new todo with an auto-generated ID like 'todo-1'. Use 'status' to change state (pending, in_progress, completed, cancelled). Use 'action' set to 'delete' to remove a todo.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "Todo ID. Set an existing ID to update/delete that todo; set a new custom ID to create a todo with that ID; omit 'id' to create a new todo with an auto-generated ID (e.g. 'todo-1').",
				},
				"title": map[string]any{
					"type":        "string",
					"description": "Title of the todo (required for new todos).",
				},
				"description": map[string]any{
					"type":        "string",
					"description": "Detailed description of what needs to be done.",
				},
				"status": map[string]any{
					"type":        "string",
					"description": "Set status: 'pending' (default for new), 'in_progress', 'completed', or 'cancelled'.",
				},
				"priority": map[string]any{
					"type":        "string",
					"description": "Priority level: 'high', 'medium' (default), or 'low'.",
				},
				"dependsOn": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "List of todo IDs that this todo depends on (e.g. ['todo-1', 'todo-3']).",
				},
				"action": map[string]any{
					"type":        "string",
					"description": "Set to 'delete' to remove a todo by its ID. When 'delete' is used, only 'id' is required.",
				},
			},
			"required": []string{},
		},
		func(ctx context.Context, args string) (string, error) {
			var params struct {
				ID          string   `json:"id"`
				Title       string   `json:"title"`
				Description string   `json:"description"`
				Status      string   `json:"status"`
				Priority    string   `json:"priority"`
				DependsOn   []string `json:"dependsOn"`
				Action      string   `json:"action"`
			}
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", fmt.Errorf("update-todo: invalid arguments: %w", err)
			}

			if params.Action == "delete" {
				if params.ID == "" {
					return "", fmt.Errorf("update-todo: id is required for delete action")
				}
				if err := store.delete(params.ID); err != nil {
					return "", fmt.Errorf("update-todo: %w", err)
				}
				return fmt.Sprintf("Deleted todo %s.\n", params.ID), nil
			}

			id := params.ID
			if id == "" {
				all, err := store.all()
				if err != nil {
					return "", fmt.Errorf("update-todo: %w", err)
				}
				id = nextTodoID(all)
			} else if exists, err := store.exists(id); err != nil {
				return "", fmt.Errorf("update-todo: %w", err)
			} else if exists {
				if err := store.update(id, func(t *Todo) {
					if params.Title != "" {
						t.Title = params.Title
					}
					if params.Description != "" {
						t.Description = params.Description
					}
					if params.Status != "" {
						t.Status = params.Status
					}
					if params.Priority != "" {
						t.Priority = params.Priority
					}
					if params.DependsOn != nil {
						t.DependsOn = params.DependsOn
					}
				}); err != nil {
					return "", fmt.Errorf("update-todo: %w", err)
				}
				return fmt.Sprintf("Updated todo %s.\n", id), nil
			}

			if params.Title == "" {
				return "", fmt.Errorf("update-todo: title is required for new todos")
			}
			status := params.Status
			if status == "" {
				status = "pending"
			}
			if !validStatus(status) {
				return "", fmt.Errorf("update-todo: invalid status %q", status)
			}
			priority := params.Priority
			if priority == "" {
				priority = "medium"
			}
			if !validPriority(priority) {
				return "", fmt.Errorf("update-todo: invalid priority %q", priority)
			}
			t := &Todo{
				ID:          id,
				Title:       params.Title,
				Description: params.Description,
				Status:      status,
				Priority:    priority,
				DependsOn:   params.DependsOn,
			}
			if err := store.add(t); err != nil {
				return "", fmt.Errorf("update-todo: %w", err)
			}
			return fmt.Sprintf("Created %s [%s] %s\n", statusIcon(t.Status), t.ID, t.Title), nil
		},
	)
}
