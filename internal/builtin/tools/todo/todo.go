package todo

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vesvai/vesvai/internal/agent/tools"
	"github.com/vesvai/vesvai/internal/core/config"
	"github.com/vesvai/vesvai/internal/vfs"
)

type Todo struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	DependsOn   []string  `json:"dependsOn,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type todoStore struct {
	mu     sync.Mutex
	file   string
	todos  map[string]*Todo
	loaded bool
}

var store = &todoStore{}

func (s *todoStore) init() error {
	if s.loaded {
		return nil
	}
	if err := config.EnsureProjectConfigDir(); err != nil {
		return fmt.Errorf("todo: ensure config dir: %w", err)
	}
	file, err := config.GetProjectConfigPath("todos.json")
	if err != nil {
		return fmt.Errorf("todo: config path: %w", err)
	}
	s.file = file
	s.todos = make(map[string]*Todo)

	data, err := os.ReadFile(s.file)
	if err != nil {
		if os.IsNotExist(err) {
			s.loaded = true
			return nil
		}
		return fmt.Errorf("todo: read file: %w", err)
	}
	var list []*Todo
	if err := json.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("todo: parse: %w", err)
	}
	for _, t := range list {
		s.todos[t.ID] = t
	}
	s.loaded = true
	return nil
}

func TodoTools(fs *vfs.VFS) {
	tools.Register(listTodoTool(fs))
	tools.Register(updateTodoTool(fs))
}

func (s *todoStore) save() error {
	list := make([]*Todo, 0, len(s.todos))
	for _, t := range s.todos {
		list = append(list, t)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.After(list[j].CreatedAt) })
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("todo: marshal: %w", err)
	}
	if err := os.WriteFile(s.file, data, 0o644); err != nil {
		return fmt.Errorf("todo: write: %w", err)
	}
	return nil
}

func (s *todoStore) all() ([]*Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.init(); err != nil {
		return nil, err
	}
	out := make([]*Todo, 0, len(s.todos))
	for _, t := range s.todos {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *todoStore) add(t *Todo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.init(); err != nil {
		return err
	}
	if _, ok := s.todos[t.ID]; ok {
		return fmt.Errorf("todo: duplicate id %q", t.ID)
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	s.todos[t.ID] = t
	return s.save()
}

func (s *todoStore) exists(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.init(); err != nil {
		return false, err
	}
	_, ok := s.todos[id]
	return ok, nil
}

func (s *todoStore) update(id string, fn func(*Todo)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.init(); err != nil {
		return err
	}
	t, ok := s.todos[id]
	if !ok {
		return fmt.Errorf("todo: not found: %q", id)
	}
	fn(t)
	t.UpdatedAt = time.Now()
	return s.save()
}

func (s *todoStore) delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.init(); err != nil {
		return err
	}
	if _, ok := s.todos[id]; !ok {
		return fmt.Errorf("todo: not found: %q", id)
	}
	delete(s.todos, id)
	return s.save()
}

func nextTodoID(todos []*Todo) string {
	max := 0
	for _, t := range todos {
		var n int
		if _, err := fmt.Sscanf(t.ID, "todo-%d", &n); err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("todo-%d", max+1)
}

func formatTodoList(todos []*Todo) string {
	if len(todos) == 0 {
		return "No todos.\n"
	}
	var b strings.Builder
	for _, t := range todos {
		status := statusIcon(t.Status)
		priority := priorityLabel(t.Priority)
		fmt.Fprintf(&b, "%s [%s] %s (priority: %s)\n", status, t.ID, t.Title, priority)
		if t.Description != "" {
			fmt.Fprintf(&b, "   %s\n", t.Description)
		}
		if len(t.DependsOn) > 0 {
			fmt.Fprintf(&b, "   depends on: %s\n", strings.Join(t.DependsOn, ", "))
		}
	}
	return b.String()
}

func statusIcon(status string) string {
	switch status {
	case "completed":
		return "[x]"
	case "in_progress":
		return "[~]"
	case "cancelled":
		return "[-]"
	default:
		return "[ ]"
	}
}

func priorityLabel(p string) string {
	switch p {
	case "high":
		return "high"
	case "low":
		return "low"
	default:
		return "medium"
	}
}

func validStatus(s string) bool {
	switch s {
	case "pending", "in_progress", "completed", "cancelled":
		return true
	}
	return false
}

func validPriority(p string) bool {
	switch p {
	case "high", "medium", "low":
		return true
	}
	return false
}
