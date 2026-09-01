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

func TestUpdateTodoToolCreate(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	out, err := tool.Execute(t.Context(), `{"title": "fix bug", "description": "fix the critical bug", "priority": "high"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "Created") {
		t.Errorf("expected 'Created', got:\n%s", out)
	}
	if !contains(t, out, "fix bug") {
		t.Errorf("expected 'fix bug', got:\n%s", out)
	}
}

func TestUpdateTodoToolCreateDefaultStatus(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	out, err := tool.Execute(t.Context(), `{"title": "default task"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "[ ]") {
		t.Errorf("expected pending status, got:\n%s", out)
	}
}

func TestUpdateTodoToolCreateMissingTitle(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	_, err := tool.Execute(t.Context(), `{}`)
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestUpdateTodoToolCreateInvalidStatus(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	_, err := tool.Execute(t.Context(), `{"title": "task", "status": "invalid"}`)
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestUpdateTodoToolCreateInvalidPriority(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	_, err := tool.Execute(t.Context(), `{"title": "task", "priority": "urgent"}`)
	if err == nil {
		t.Fatal("expected error for invalid priority")
	}
}

func TestUpdateTodoToolUpdate(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	tool.Execute(t.Context(), `{"title": "task one"}`)

	out, err := tool.Execute(t.Context(), `{"id": "todo-1", "title": "task one updated", "status": "in_progress"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "Updated") {
		t.Errorf("expected 'Updated', got:\n%s", out)
	}

	listTool := listTodoTool(nil)
	listOut, err := listTool.Execute(t.Context(), `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, listOut, "task one updated") {
		t.Errorf("expected 'task one updated', got:\n%s", out)
	}
	if !contains(t, listOut, "[~]") {
		t.Errorf("expected in_progress status, got:\n%s", listOut)
	}
}

func TestUpdateTodoToolComplete(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	tool.Execute(t.Context(), `{"title": "task to complete"}`)

	out, err := tool.Execute(t.Context(), `{"id": "todo-1", "status": "completed"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "Updated") {
		t.Errorf("expected 'Updated', got:\n%s", out)
	}

	listTool := listTodoTool(nil)
	listOut, err := listTool.Execute(t.Context(), `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, listOut, "[x]") {
		t.Errorf("expected completed status, got:\n%s", listOut)
	}
}

func TestUpdateTodoToolDelete(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	tool.Execute(t.Context(), `{"title": "to delete"}`)

	out, err := tool.Execute(t.Context(), `{"id": "todo-1", "action": "delete"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "Deleted") {
		t.Errorf("expected 'Deleted', got:\n%s", out)
	}

	listTool := listTodoTool(nil)
	listOut, err := listTool.Execute(t.Context(), `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, listOut, "No todos") {
		t.Errorf("expected 'No todos', got:\n%s", listOut)
	}
}

func TestUpdateTodoToolDeleteMissingID(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	_, err := tool.Execute(t.Context(), `{"action": "delete"}`)
	if err == nil {
		t.Fatal("expected error for missing id on delete")
	}
}

func TestUpdateTodoToolDeleteNonexistent(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	_, err := tool.Execute(t.Context(), `{"id": "todo-999", "action": "delete"}`)
	if err == nil {
		t.Fatal("expected error for nonexistent id")
	}
}

func TestUpdateTodoToolCreateWithCustomID(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	out, err := tool.Execute(t.Context(), `{"id": "SPEC-42", "title": "custom id task"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "SPEC-42") {
		t.Errorf("expected custom id in output, got:\n%s", out)
	}

	all, err := store.all()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].ID != "SPEC-42" {
		t.Errorf("expected one todo with id SPEC-42, got %+v", all)
	}
}

func TestUpdateTodoToolCustomIDHasPriority(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	if _, err := tool.Execute(t.Context(), `{"title": "first task"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := tool.Execute(t.Context(), `{"id": "CUSTOM-1", "title": "custom"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := tool.Execute(t.Context(), `{"title": "third task"}`); err != nil {
		t.Fatal(err)
	}

	all, err := store.all()
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, t := range all {
		ids[t.ID] = true
	}
	if !ids["todo-1"] || !ids["CUSTOM-1"] || !ids["todo-2"] {
		t.Errorf("expected todo-1, CUSTOM-1, todo-2 (auto ids must skip custom ones), got %v", ids)
	}
}

func TestUpdateTodoToolUpdateNonexistentWithoutTitle(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	if _, err := tool.Execute(t.Context(), `{"id": "todo-999"}`); err == nil {
		t.Fatal("expected error for nonexistent id without title")
	}
}

func TestUpdateTodoToolWithDependsOn(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	tool.Execute(t.Context(), `{"title": "first task"}`)
	tool.Execute(t.Context(), `{"title": "second task"}`)

	_, err := tool.Execute(t.Context(), `{"id": "todo-2", "dependsOn": ["todo-1"]}`)
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

func TestListTodoToolFilterStatus(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)
	listTool := listTodoTool(nil)

	tool.Execute(t.Context(), `{"title": "pending task"}`)
	tool.Execute(t.Context(), `{"id": "todo-1", "status": "completed"}`)
	tool.Execute(t.Context(), `{"title": "another pending"}`)

	out, err := listTool.Execute(t.Context(), `{"status": "completed"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "[x]") {
		t.Errorf("expected completed icon, got:\n%s", out)
	}
	if contains(t, out, "another pending") {
		t.Errorf("expected NOT to see pending tasks, got:\n%s", out)
	}
}

func TestListTodoToolFilterPriority(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)
	listTool := listTodoTool(nil)

	tool.Execute(t.Context(), `{"title": "high priority", "priority": "high"}`)
	tool.Execute(t.Context(), `{"title": "medium priority"}`)

	out, err := listTool.Execute(t.Context(), `{"priority": "high"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "high priority") {
		t.Errorf("expected 'high priority', got:\n%s", out)
	}
	if contains(t, out, "medium priority") {
		t.Errorf("expected NOT to see medium priority, got:\n%s", out)
	}
}

func TestListTodoToolInvalidStatus(t *testing.T) {
	setupTodoTest(t)
	tool := listTodoTool(nil)

	_, err := tool.Execute(t.Context(), `{"status": "invalid"}`)
	if err == nil {
		t.Fatal("expected error for invalid status filter")
	}
}

func TestListTodoToolInvalidPriority(t *testing.T) {
	setupTodoTest(t)
	tool := listTodoTool(nil)

	_, err := tool.Execute(t.Context(), `{"priority": "invalid"}`)
	if err == nil {
		t.Fatal("expected error for invalid priority filter")
	}
}

func TestTodoPersistence(t *testing.T) {
	setupTodoTest(t)
	tool := updateTodoTool(nil)

	tool.Execute(t.Context(), `{"title": "persistent task"}`)

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

func TestNextTodoID(t *testing.T) {
	todos := []*Todo{{ID: "todo-1"}, {ID: "todo-5"}, {ID: "todo-3"}}
	id := nextTodoID(todos)
	if id != "todo-6" {
		t.Errorf("expected todo-6, got %s", id)
	}
}

func TestNextTodoIDEmpty(t *testing.T) {
	id := nextTodoID(nil)
	if id != "todo-1" {
		t.Errorf("expected todo-1, got %s", id)
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

func TestValidStatus(t *testing.T) {
	if !validStatus("pending") {
		t.Error("pending should be valid")
	}
	if !validStatus("completed") {
		t.Error("completed should be valid")
	}
	if validStatus("invalid") {
		t.Error("invalid should not be valid")
	}
}

func TestValidPriority(t *testing.T) {
	if !validPriority("high") {
		t.Error("high should be valid")
	}
	if validPriority("urgent") {
		t.Error("urgent should not be valid")
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
