package file

import (
	"testing"
)

func TestGlobTool(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"a.go":     "package a",
		"b.go":     "package b",
		"sub/c.go": "package c",
		"d.txt":    "text",
	})
	tool := globTool(fs)

	out, err := tool.Execute(t.Context(), `{"pattern": "*.go"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "a.go") {
		t.Errorf("expected output to contain 'a.go', got:\n%s", out)
	}
	if !contains(t, out, "b.go") {
		t.Errorf("expected output to contain 'b.go', got:\n%s", out)
	}
	if !contains(t, out, "sub/c.go") {
		t.Errorf("expected output to contain 'sub/c.go' (unanchored glob walks subdirs), got:\n%s", out)
	}
	if contains(t, out, "d.txt") {
		t.Errorf("expected output NOT to contain 'd.txt', got:\n%s", out)
	}
}

func TestGlobToolRecursive(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"a.go":     "package a",
		"sub/c.go": "package c",
	})
	tool := globTool(fs)

	out, err := tool.Execute(t.Context(), `{"pattern": "**/*.go"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "a.go") {
		t.Errorf("expected output to contain 'a.go', got:\n%s", out)
	}
	if !contains(t, out, "sub/c.go") {
		t.Errorf("expected output to contain 'sub/c.go', got:\n%s", out)
	}
}

func TestGlobToolNoMatch(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"a.go": "package a",
	})
	tool := globTool(fs)

	out, err := tool.Execute(t.Context(), `{"pattern": "*.py"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "No files matched") {
		t.Errorf("expected 'No files matched', got:\n%s", out)
	}
}

func TestGlobToolMissingPattern(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{})
	tool := globTool(fs)

	_, err := tool.Execute(t.Context(), `{}`)
	if err == nil {
		t.Fatal("expected error for missing pattern")
	}
}
