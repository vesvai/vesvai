package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/filesystem"
)

const todosRelDir = ".vesvai/todos"

type Todo struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TodoStore struct {
	mu       sync.RWMutex
	todos    map[string]*Todo
	relPath  string
	fs       *filesystem.FileSystem
}

func NewTodoStore(sessionID string, fs *filesystem.FileSystem) (*TodoStore, error) {
	relPath := filepath.Join(todosRelDir, sessionID+".json")
	store := &TodoStore{
		todos:   make(map[string]*Todo),
		relPath: relPath,
		fs:      fs,
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *TodoStore) load() error {
	ctx := context.Background()
	data, err := s.fs.Read(ctx, s.relPath)
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			return nil
		}
		return fmt.Errorf("failed to read todos file: %w", err)
	}

	var todos []*Todo
	if err := json.Unmarshal([]byte(data), &todos); err != nil {
		return fmt.Errorf("failed to parse todos file: %w", err)
	}

	for _, t := range todos {
		s.todos[t.ID] = t
	}

	return nil
}

func (s *TodoStore) save() error {
	todos := make([]*Todo, 0, len(s.todos))
	for _, t := range s.todos {
		todos = append(todos, t)
	}

	data, err := json.MarshalIndent(todos, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal todos: %w", err)
	}

	ctx := context.Background()
	if err := s.fs.Write(ctx, s.relPath, data); err != nil {
		return fmt.Errorf("failed to write todos file: %w", err)
	}

	return nil
}

func (s *TodoStore) Set(description string) *Todo {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("todo-%d", time.Now().UnixNano())
	now := time.Now()

	todo := &Todo{
		ID:          id,
		Description: description,
		Status:      "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.todos[id] = todo
	s.save()
	return todo
}

func (s *TodoStore) Get(id string) (*Todo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	todo, ok := s.todos[id]
	return todo, ok
}

func (s *TodoStore) List() []*Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	todos := make([]*Todo, 0, len(s.todos))
	for _, t := range s.todos {
		todos = append(todos, t)
	}
	return todos
}

func (s *TodoStore) Update(id string, status string, description string) (*Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo, ok := s.todos[id]
	if !ok {
		return nil, fmt.Errorf("todo not found: %s", id)
	}

	if status != "" {
		if status != "pending" && status != "done" {
			return nil, fmt.Errorf("invalid status: %s (must be 'pending' or 'done')", status)
		}
		todo.Status = status
	}

	if description != "" {
		todo.Description = description
	}

	todo.UpdatedAt = time.Now()
	s.save()
	return todo, nil
}

func (s *TodoStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.todos[id]; !ok {
		return fmt.Errorf("todo not found: %s", id)
	}

	delete(s.todos, id)
	return s.save()
}

func (s *TodoStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.todos)
}

func (s *TodoStore) PendingCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, t := range s.todos {
		if t.Status == "pending" {
			count++
		}
	}
	return count
}

func (s *TodoStore) DoneCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, t := range s.todos {
		if t.Status == "done" {
			count++
		}
	}
	return count
}

func newSetTodoTool(fs *filesystem.FileSystem) agent.Tool {
	return agent.NewFuncTool(
		"set-todo",
		"Add a new todo item. Returns the created todo with its ID.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"description": map[string]any{
					"type":        "string",
					"description": "The description of the todo item",
				},
			},
			"required": []string{"description"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			description, _ := params["description"].(string)
			if description == "" {
				return "", fmt.Errorf("description is required")
			}

			sessionID := agent.SessionIDFromContext(ctx)
			if sessionID == "" {
				sessionID = "default"
			}

			store, err := NewTodoStore(sessionID, fs)
			if err != nil {
				return "", fmt.Errorf("failed to create todo store: %w", err)
			}

			todo := store.Set(description)

			return fmt.Sprintf("Todo created successfully.\nID: %s\nDescription: %s\nStatus: %s\n", todo.ID, todo.Description, todo.Status), nil
		},
	)
}

func newGetTodoTool(fs *filesystem.FileSystem) agent.Tool {
	return agent.NewFuncTool(
		"get-todo",
		"Get a specific todo item by its ID.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "The todo ID to retrieve",
				},
			},
			"required": []string{"id"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			id, _ := params["id"].(string)
			if id == "" {
				return "", fmt.Errorf("id is required")
			}

			sessionID := agent.SessionIDFromContext(ctx)
			if sessionID == "" {
				sessionID = "default"
			}

			store, err := NewTodoStore(sessionID, fs)
			if err != nil {
				return "", fmt.Errorf("failed to create todo store: %w", err)
			}

			todo, ok := store.Get(id)
			if !ok {
				return "", fmt.Errorf("todo not found: %s", id)
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Todo: %s\n", todo.Description))
			sb.WriteString(fmt.Sprintf("ID: %s\n", todo.ID))
			sb.WriteString(fmt.Sprintf("Status: %s\n", todo.Status))
			sb.WriteString(fmt.Sprintf("Created: %s\n", todo.CreatedAt.Format("2006-01-02 15:04:05")))
			sb.WriteString(fmt.Sprintf("Updated: %s\n", todo.UpdatedAt.Format("2006-01-02 15:04:05")))

			return sb.String(), nil
		},
	)
}

func newListTodosTool(fs *filesystem.FileSystem) agent.Tool {
	return agent.NewFuncTool(
		"list-todos",
		"List all todo items. Returns a summary of all todos with their status.",
		map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			sessionID := agent.SessionIDFromContext(ctx)
			if sessionID == "" {
				sessionID = "default"
			}

			store, err := NewTodoStore(sessionID, fs)
			if err != nil {
				return "", fmt.Errorf("failed to create todo store: %w", err)
			}

			todos := store.List()
			if len(todos) == 0 {
				return "No todos found.\n", nil
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Todos (%d total, %d pending, %d done):\n\n", len(todos), store.PendingCount(), store.DoneCount()))

			for _, todo := range todos {
				statusIcon := "[ ]"
				if todo.Status == "done" {
					statusIcon = "[x]"
				}
				sb.WriteString(fmt.Sprintf("%s %s - %s\n", statusIcon, todo.ID, todo.Description))
			}

			return sb.String(), nil
		},
	)
}

func newUpdateTodoTool(fs *filesystem.FileSystem) agent.Tool {
	return agent.NewFuncTool(
		"update-todo",
		"Update a todo item's status or description.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "The todo ID to update",
				},
				"status": map[string]any{
					"type":        "string",
					"description": "New status ('pending' or 'done')",
					"enum":        []string{"pending", "done"},
				},
				"description": map[string]any{
					"type":        "string",
					"description": "New description (optional)",
				},
			},
			"required": []string{"id"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			id, _ := params["id"].(string)
			if id == "" {
				return "", fmt.Errorf("id is required")
			}

			status, _ := params["status"].(string)
			description, _ := params["description"].(string)

			if status == "" && description == "" {
				return "", fmt.Errorf("at least one of status or description is required")
			}

			sessionID := agent.SessionIDFromContext(ctx)
			if sessionID == "" {
				sessionID = "default"
			}

			store, err := NewTodoStore(sessionID, fs)
			if err != nil {
				return "", fmt.Errorf("failed to create todo store: %w", err)
			}

			todo, err := store.Update(id, status, description)
			if err != nil {
				return "", err
			}

			return fmt.Sprintf("Todo updated successfully.\nID: %s\nDescription: %s\nStatus: %s\n", todo.ID, todo.Description, todo.Status), nil
		},
	)
}

func newDeleteTodoTool(fs *filesystem.FileSystem) agent.Tool {
	return agent.NewFuncTool(
		"delete-todo",
		"Delete a todo item by its ID.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "The todo ID to delete",
				},
			},
			"required": []string{"id"},
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			id, _ := params["id"].(string)
			if id == "" {
				return "", fmt.Errorf("id is required")
			}

			sessionID := agent.SessionIDFromContext(ctx)
			if sessionID == "" {
				sessionID = "default"
			}

			store, err := NewTodoStore(sessionID, fs)
			if err != nil {
				return "", fmt.Errorf("failed to create todo store: %w", err)
			}

			if err := store.Delete(id); err != nil {
				return "", err
			}

			return fmt.Sprintf("Todo deleted successfully.\nID: %s\n", id), nil
		},
	)
}
