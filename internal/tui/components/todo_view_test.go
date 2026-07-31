package components

import "testing"

func TestTodoView_ParseListResult(t *testing.T) {
	tv := NewTodoView()
	result := "Todos (2 total, 1 pending, 1 done):\n\n[ ] todo-123 - Fix tests\n[x] todo-456 - Write docs\n"
	tv.HandleToolResult("list-todos", result)

	if tv.TotalCount() != 2 {
		t.Errorf("total = %d, want 2", tv.TotalCount())
	}
	if tv.PendingCount() != 1 {
		t.Errorf("pending = %d, want 1", tv.PendingCount())
	}
	if tv.DoneCount() != 1 {
		t.Errorf("done = %d, want 1", tv.DoneCount())
	}

	if tv.Items()[0].ID != "todo-123" || tv.Items()[0].Status != "pending" {
		t.Errorf("item 0 = %+v", tv.Items()[0])
	}
	if tv.Items()[1].ID != "todo-456" || tv.Items()[1].Status != "done" {
		t.Errorf("item 1 = %+v", tv.Items()[1])
	}
}

func TestTodoView_ParseListResult_Empty(t *testing.T) {
	tv := NewTodoView()
	tv.HandleToolResult("list-todos", "No todos found.\n")
	if tv.TotalCount() != 0 {
		t.Errorf("total = %d, want 0", tv.TotalCount())
	}
}

func TestTodoView_ParseSetResult(t *testing.T) {
	tv := NewTodoView()
	result := "Todo created successfully.\nID: todo-999\nDescription: Fix bug\nStatus: pending\n"
	tv.HandleToolResult("set-todo", result)

	if tv.TotalCount() != 1 {
		t.Fatalf("total = %d, want 1", tv.TotalCount())
	}
	item := tv.Items()[0]
	if item.ID != "todo-999" || item.Description != "Fix bug" || item.Status != "pending" {
		t.Errorf("item = %+v", item)
	}
}

func TestTodoView_ParseUpdateResult(t *testing.T) {
	tv := NewTodoView()
	tv.HandleToolResult("set-todo", "Todo created successfully.\nID: todo-1\nDescription: Old desc\nStatus: pending\n")

	updateResult := "Todo updated successfully.\nID: todo-1\nDescription: New desc\nStatus: done\n"
	tv.HandleToolResult("update-todo", updateResult)

	if tv.TotalCount() != 1 {
		t.Fatalf("total = %d, want 1", tv.TotalCount())
	}
	item := tv.Items()[0]
	if item.Description != "New desc" {
		t.Errorf("description = %q, want New desc", item.Description)
	}
	if item.Status != "done" {
		t.Errorf("status = %q, want done", item.Status)
	}
}

func TestTodoView_ParseDeleteResult(t *testing.T) {
	tv := NewTodoView()
	tv.HandleToolResult("set-todo", "Todo created successfully.\nID: todo-1\nDescription: Task A\nStatus: pending\n")
	tv.HandleToolResult("set-todo", "Todo created successfully.\nID: todo-2\nDescription: Task B\nStatus: pending\n")

	if tv.TotalCount() != 2 {
		t.Fatalf("total = %d, want 2", tv.TotalCount())
	}

	tv.HandleToolResult("delete-todo", "Todo deleted successfully.\nID: todo-1\n")

	if tv.TotalCount() != 1 {
		t.Fatalf("after delete, total = %d, want 1", tv.TotalCount())
	}
	if tv.Items()[0].ID != "todo-2" {
		t.Errorf("remaining item ID = %q, want todo-2", tv.Items()[0].ID)
	}
}

func TestTodoView_Toggle(t *testing.T) {
	tv := NewTodoView()
	if tv.IsVisible() {
		t.Error("should start hidden")
	}
	tv.Toggle()
	if !tv.IsVisible() {
		t.Error("should be visible after toggle")
	}
	tv.Toggle()
	if tv.IsVisible() {
		t.Error("should be hidden after second toggle")
	}
}

func TestTodoView_NonTodoToolIgnored(t *testing.T) {
	tv := NewTodoView()
	tv.HandleToolResult("bash", "some output")
	if tv.TotalCount() != 0 {
		t.Error("non-todo tools should be ignored")
	}
}

func TestTodoView_SetResultExistingItem(t *testing.T) {
	tv := NewTodoView()
	tv.HandleToolResult("set-todo", "Todo created.\nID: t1\nDescription: A\nStatus: pending\n")
	tv.HandleToolResult("set-todo", "Todo created.\nID: t1\nDescription: A updated\nStatus: done\n")
	if tv.TotalCount() != 1 {
		t.Fatalf("total = %d, want 1 (should update in place)", tv.TotalCount())
	}
	if tv.Items()[0].Description != "A updated" || tv.Items()[0].Status != "done" {
		t.Errorf("item not updated: %+v", tv.Items()[0])
	}
}

func TestExtractField(t *testing.T) {
	s := "Line 1\nID: abc123\nDescription: hello world\nStatus: pending\n"
	if got := extractField(s, "ID:"); got != "abc123" {
		t.Errorf("ID = %q", got)
	}
	if got := extractField(s, "Description:"); got != "hello world" {
		t.Errorf("Description = %q", got)
	}
	if got := extractField(s, "Status:"); got != "pending" {
		t.Errorf("Status = %q", got)
	}
	if got := extractField(s, "Missing:"); got != "" {
		t.Errorf("Missing = %q, want empty", got)
	}
}
