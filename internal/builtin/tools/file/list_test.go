package file

import (
	"testing"
)

func TestListTool(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"a.txt":     "aaa",
		"b.txt":     "bbb",
		"sub/c.txt": "ccc",
	})
	tool := listTool(fs)

	out, err := tool.Execute(t.Context(), `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "a.txt") {
		t.Errorf("expected output to contain 'a.txt', got:\n%s", out)
	}
	if !contains(t, out, "b.txt") {
		t.Errorf("expected output to contain 'b.txt', got:\n%s", out)
	}
	if !contains(t, out, "sub") {
		t.Errorf("expected output to contain 'sub', got:\n%s", out)
	}
}

func TestListToolSubdir(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"sub/c.txt":   "ccc",
		"sub/d.txt":   "ddd",
		"other/e.txt": "eee",
	})
	tool := listTool(fs)

	out, err := tool.Execute(t.Context(), `{"path": "sub"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "c.txt") {
		t.Errorf("expected output to contain 'c.txt', got:\n%s", out)
	}
	if !contains(t, out, "d.txt") {
		t.Errorf("expected output to contain 'd.txt', got:\n%s", out)
	}
	if contains(t, out, "other") {
		t.Errorf("expected output NOT to contain 'other', got:\n%s", out)
	}
}

func TestListToolEmpty(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{})
	tool := listTool(fs)

	out, err := tool.Execute(t.Context(), `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "(empty)") {
		t.Errorf("expected empty directory indicator, got:\n%s", out)
	}
}

func TestListToolNonexistent(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{})
	tool := listTool(fs)

	_, err := tool.Execute(t.Context(), `{"path": "nonexistent"}`)
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}
