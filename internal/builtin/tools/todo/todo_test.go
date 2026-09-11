package todo

import (
	"os"
	"testing"
)

func setupTodoTest(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	store.file = ""
	store.todos = make(map[string]*Todo)
	store.loaded = false
	store.mu.Unlock()
}

func TestListTodoToolEmpty(t *testing.T) {
	setupTodoTest(t)
	tool := listTodoTool(nil)

	out, err := tool.Execute(t.Context(), `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "No todos") {
		t.Errorf("expected 'No todos', got:\n%s", out)
	}
}

func TestUpdateTodoToolSet(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	out, err := tool.Execute(t.Context(), `{"todos": [
		{"id": "todo-1", "title": "fix bug", "description": "fix the critical bug", "status": "pending", "priority": "high"}
	]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "Set 1 todos") {
		t.Errorf("expected 'Set 1 todos', got:\n%s", out)
	}

	all, err := store.all()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(all))
	}
	if all[0].ID != "todo-1" || all[0].Title != "fix bug" || all[0].Priority != "high" {
		t.Errorf("unexpected todo: %+v", all[0])
	}
	if all[0].CreatedAt.IsZero() || all[0].UpdatedAt.IsZero() {
		t.Errorf("expected timestamps to be set: %+v", all[0])
	}
}

func TestUpdateTodoToolReplace(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	if _, err := tool.Execute(t.Context(), `{"todos": [
		{"id": "todo-1", "title": "first", "status": "pending", "priority": "medium"}
	]}`); err != nil {
		t.Fatal(err)
	}

	out, err := tool.Execute(t.Context(), `{"todos": [
		{"id": "todo-1", "title": "first updated", "status": "completed", "priority": "high"},
		{"id": "todo-2", "title": "second", "status": "pending", "priority": "low"}
	]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "Set 2 todos") {
		t.Errorf("expected 'Set 2 todos', got:\n%s", out)
	}

	all, err := store.all()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 todos, got %d", len(all))
	}
}

func TestUpdateTodoToolClear(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	if _, err := tool.Execute(t.Context(), `{"todos": [
		{"id": "todo-1", "title": "first", "status": "pending", "priority": "medium"}
	]}`); err != nil {
		t.Fatal(err)
	}

	if _, err := tool.Execute(t.Context(), `{"todos": []}`); err != nil {
		t.Fatal(err)
	}

	all, err := store.all()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 0 {
		t.Fatalf("expected no todos, got %d", len(all))
	}
}

func TestUpdateTodoToolInvalidJSON(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	if _, err := tool.Execute(t.Context(), `not json`); err == nil {
		t.Fatal("expected error for invalid json")
	}
}

func TestUpdateTodoToolWithDependsOn(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	_, err := tool.Execute(t.Context(), `{"todos": [
		{"id": "todo-1", "title": "first", "status": "completed", "priority": "medium"},
		{"id": "todo-2", "title": "second", "status": "pending", "priority": "medium", "dependsOn": ["todo-1"]}
	]}`)
	if err != nil {
		t.Fatal(err)
	}

	listTool := listTodoTool(nil)
	listOut, err := listTool.Execute(t.Context(), `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, listOut, "depends on: todo-1") {
		t.Errorf("expected dependency info, got:\n%s", listOut)
	}
}

func TestTodoPersistence(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	if _, err := tool.Execute(t.Context(), `{"todos": [
		{"id": "todo-1", "title": "persistent task", "status": "pending", "priority": "medium"}
	]}`); err != nil {
		t.Fatal(err)
	}

	store.mu.Lock()
	store.loaded = false
	store.mu.Unlock()

	listTool := listTodoTool(nil)
	out, err := listTool.Execute(t.Context(), `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "persistent task") {
		t.Errorf("expected todo to persist, got:\n%s", out)
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct{ status, expected string }{
		{"completed", "[x]"},
		{"in_progress", "[~]"},
		{"cancelled", "[-]"},
		{"pending", "[ ]"},
		{"unknown", "[ ]"},
	}
	for _, tt := range tests {
		got := statusIcon(tt.status)
		if got != tt.expected {
			t.Errorf("statusIcon(%q) = %q, want %q", tt.status, got, tt.expected)
		}
	}
}

func contains(t *testing.T, s, substr string) bool {
	t.Helper()
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
