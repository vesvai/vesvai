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

func TestListToolTreeStructure(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"a.txt":            "aaa",
		"b.txt":            "bbb",
		"sub/c.txt":        "ccc",
		"sub/d.txt":        "ddd",
		"sub/nested/e.txt": "eee",
	})
	tool := listTool(fs)

	out, err := tool.Execute(t.Context(), `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "sub/") {
		t.Errorf("expected tree to show 'sub/' directory, got:\n%s", out)
	}
	if !contains(t, out, "nested/") {
		t.Errorf("expected tree to show 'nested/' directory, got:\n%s", out)
	}
	if !contains(t, out, "c.txt") {
		t.Errorf("expected tree to show 'c.txt', got:\n%s", out)
	}
	if !contains(t, out, "e.txt") {
		t.Errorf("expected tree to show 'e.txt', got:\n%s", out)
	}
}

func TestListToolIgnore(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"a.txt":     "aaa",
		"b.log":     "bbb",
		"sub/c.txt": "ccc",
		"sub/d.log": "ddd",
	})
	tool := listTool(fs)

	out, err := tool.Execute(t.Context(), `{"ignore": ["*.log"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "a.txt") {
		t.Errorf("expected output to contain 'a.txt', got:\n%s", out)
	}
	if !contains(t, out, "c.txt") {
		t.Errorf("expected output to contain 'c.txt', got:\n%s", out)
	}
	if contains(t, out, "b.log") {
		t.Errorf("expected output NOT to contain 'b.log', got:\n%s", out)
	}
	if contains(t, out, "d.log") {
		t.Errorf("expected output NOT to contain 'd.log', got:\n%s", out)
	}
}

func TestListToolIgnoreMultiplePatterns(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"a.txt":     "aaa",
		"b.log":     "bbb",
		"c.tmp":     "ccc",
		"sub/d.txt": "ddd",
	})
	tool := listTool(fs)

	out, err := tool.Execute(t.Context(), `{"ignore": ["*.log", "*.tmp"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "a.txt") {
		t.Errorf("expected output to contain 'a.txt', got:\n%s", out)
	}
	if !contains(t, out, "d.txt") {
		t.Errorf("expected output to contain 'd.txt', got:\n%s", out)
	}
	if contains(t, out, "b.log") {
		t.Errorf("expected output NOT to contain 'b.log', got:\n%s", out)
	}
	if contains(t, out, "c.tmp") {
		t.Errorf("expected output NOT to contain 'c.tmp', got:\n%s", out)
	}
}
