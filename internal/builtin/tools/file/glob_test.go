package file

import (
	"fmt"
	"strings"
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
	if !contains(t, out, "No files found") {
		t.Errorf("expected 'No files found', got:\n%s", out)
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

func TestGlobToolWithPath(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"root.go":       "package root",
		"dir/nested.go": "package nested",
		"other.go":      "package other",
	})
	tool := globTool(fs)

	out, err := tool.Execute(t.Context(), `{"pattern": "*.go", "path": "dir"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "nested.go") {
		t.Errorf("expected output to contain 'nested.go', got:\n%s", out)
	}
	if contains(t, out, "root.go") {
		t.Errorf("expected output NOT to contain 'root.go', got:\n%s", out)
	}
}

func TestGlobToolTruncation(t *testing.T) {
	files := map[string]string{}
	for i := 0; i < 150; i++ {
		files[fmt.Sprintf("file%03d.txt", i)] = "content"
	}
	fs := setupTestVFS(t, files)
	tool := globTool(fs)

	out, err := tool.Execute(t.Context(), `{"pattern": "*.txt"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "truncated") {
		t.Errorf("expected truncation message, got:\n%s", out)
	}
	count := 0
	for _, line := range splitLines(out) {
		if len(line) > 0 && line != "(Results are truncated. Consider using a more specific path or pattern.)" {
			count++
		}
	}
	if count != 100 {
		t.Errorf("expected 100 file lines, got %d", count)
	}
}

func TestGlobToolUnderLimit(t *testing.T) {
	files := map[string]string{}
	for i := 0; i < 50; i++ {
		files[fmt.Sprintf("file%03d.txt", i)] = "content"
	}
	fs := setupTestVFS(t, files)
	tool := globTool(fs)

	out, err := tool.Execute(t.Context(), `{"pattern": "*.txt"}`)
	if err != nil {
		t.Fatal(err)
	}
	if contains(t, out, "truncated") {
		t.Errorf("expected no truncation message for 50 files, got:\n%s", out)
	}
	count := 0
	for _, line := range splitLines(out) {
		if len(line) > 0 {
			count++
		}
	}
	if count != 50 {
		t.Errorf("expected 50 file lines, got %d", count)
	}
}

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}
