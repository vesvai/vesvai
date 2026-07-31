package tools

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/filesystem"
)

func TestTodoStore_SetAndGet(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	store := &TodoStore{
		todos:   make(map[string]*Todo),
		relPath: filepath.Join(todosRelDir, "test-session.json"),
		fs:      fs,
	}

	todo := store.Set("Test todo")
	if todo == nil {
		t.Fatal("expected todo to be created")
	}
	if todo.Description != "Test todo" {
		t.Errorf("expected description 'Test todo', got %q", todo.Description)
	}
	if todo.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", todo.Status)
	}

	got, ok := store.Get(todo.ID)
	if !ok {
		t.Fatal("expected to find todo")
	}
	if got.ID != todo.ID {
		t.Errorf("expected ID %q, got %q", todo.ID, got.ID)
	}
}

func TestTodoStore_Update(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	store := &TodoStore{
		todos:   make(map[string]*Todo),
		relPath: filepath.Join(todosRelDir, "test-session.json"),
		fs:      fs,
	}

	todo := store.Set("Test todo")

	updated, err := store.Update(todo.ID, "done", "")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "done" {
		t.Errorf("expected status 'done', got %q", updated.Status)
	}

	updated2, err := store.Update(todo.ID, "", "Updated description")
	if err != nil {
		t.Fatal(err)
	}
	if updated2.Description != "Updated description" {
		t.Errorf("expected description 'Updated description', got %q", updated2.Description)
	}
}

func TestTodoStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	store := &TodoStore{
		todos:   make(map[string]*Todo),
		relPath: filepath.Join(todosRelDir, "test-session.json"),
		fs:      fs,
	}

	todo := store.Set("Test todo")

	if err := store.Delete(todo.ID); err != nil {
		t.Fatal(err)
	}

	_, ok := store.Get(todo.ID)
	if ok {
		t.Error("expected todo to be deleted")
	}
}

func TestTodoStore_List(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	store := &TodoStore{
		todos:   make(map[string]*Todo),
		relPath: filepath.Join(todosRelDir, "test-session.json"),
		fs:      fs,
	}

	store.Set("Todo 1")
	store.Set("Todo 2")
	store.Set("Todo 3")

	todos := store.List()
	if len(todos) != 3 {
		t.Errorf("expected 3 todos, got %d", len(todos))
	}
}

func TestTodoTools_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ctx = agent.WithAgentContext(ctx, "test-agent", "test-session")

	setTool := newSetTodoTool(fs)
	getTool := newGetTodoTool(fs)
	listTool := newListTodosTool(fs)
	updateTool := newUpdateTodoTool(fs)
	deleteTool := newDeleteTodoTool(fs)

	setResult, err := setTool.Handle(ctx, map[string]any{"description": "Test todo"})
	if err != nil {
		t.Fatalf("set-todo failed: %v", err)
	}
	t.Logf("Set result: %s", setResult)

	listResult, err := listTool.Handle(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("list-todos failed: %v", err)
	}
	t.Logf("List result: %s", listResult)

	store, _ := NewTodoStore("test-session", fs)
	todos := store.List()
	if len(todos) == 0 {
		t.Fatal("expected at least one todo")
	}

	todoID := todos[0].ID

	getResult, err := getTool.Handle(ctx, map[string]any{"id": todoID})
	if err != nil {
		t.Fatalf("get-todo failed: %v", err)
	}
	t.Logf("Get result: %s", getResult)

	updateResult, err := updateTool.Handle(ctx, map[string]any{"id": todoID, "status": "done"})
	if err != nil {
		t.Fatalf("update-todo failed: %v", err)
	}
	t.Logf("Update result: %s", updateResult)

	deleteResult, err := deleteTool.Handle(ctx, map[string]any{"id": todoID})
	if err != nil {
		t.Fatalf("delete-todo failed: %v", err)
	}
	t.Logf("Delete result: %s", deleteResult)

	store2, _ := NewTodoStore("test-session", fs)
	_, ok := store2.Get(todoID)
	if ok {
		t.Error("expected todo to be deleted")
	}
}
