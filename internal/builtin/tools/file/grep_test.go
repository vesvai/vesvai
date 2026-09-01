package file

import (
	"testing"
)

func TestGrepTool(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"a.go": `package a
func hello() {
	return "hello"
}
`,
		"b.go": `package b
func goodbye() {
	return "goodbye"
}
`,
	})
	tool := grepTool(fs)

	out, err := tool.Execute(t.Context(), `{"pattern": "hello"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "a.go") {
		t.Errorf("expected output to contain 'a.go', got:\n%s", out)
	}
	if contains(t, out, "b.go") {
		t.Errorf("expected output NOT to contain 'b.go', got:\n%s", out)
	}
}

func TestGrepToolFilesWithMatches(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"a.go": `package a
func hello() {}
`,
		"b.go": `package b
func hello() {}
`,
	})
	tool := grepTool(fs)

	out, err := tool.Execute(t.Context(), `{"pattern": "hello", "mode": "files_with_matches"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "a.go") {
		t.Errorf("expected output to contain 'a.go', got:\n%s", out)
	}
	if !contains(t, out, "b.go") {
		t.Errorf("expected output to contain 'b.go', got:\n%s", out)
	}
}

func TestGrepToolCount(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"a.go": `package a
func hello() {
	return hello()
}
`,
	})
	tool := grepTool(fs)

	out, err := tool.Execute(t.Context(), `{"pattern": "hello", "mode": "count"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "a.go") {
		t.Errorf("expected output to contain 'a.go', got:\n%s", out)
	}
}

func TestGrepToolNoMatch(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"a.go": "package a",
	})
	tool := grepTool(fs)

	out, err := tool.Execute(t.Context(), `{"pattern": "nonexistent"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "No matches") {
		t.Errorf("expected 'No matches', got:\n%s", out)
	}
}

func TestGrepToolMissingPattern(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{})
	tool := grepTool(fs)

	_, err := tool.Execute(t.Context(), `{}`)
	if err == nil {
		t.Fatal("expected error for missing pattern")
	}
}

func TestGrepToolInclude(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"a.go": "package a\nvar x = 1\n",
		"b.ts": "let x = 1\n",
	})
	tool := grepTool(fs)

	out, err := tool.Execute(t.Context(), `{"pattern": "x", "include": ["*.go"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "a.go") {
		t.Errorf("expected output to contain 'a.go', got:\n%s", out)
	}
	if contains(t, out, "b.ts") {
		t.Errorf("expected output NOT to contain 'b.ts', got:\n%s", out)
	}
}
