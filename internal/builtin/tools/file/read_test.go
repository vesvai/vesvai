package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vesvai/vesvai/internal/vfs"
)

func setupTestVFS(t *testing.T, files map[string]string) *vfs.VFS {
	t.Helper()
	dir := t.TempDir()
	for path, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	fs, err := vfs.New(dir, vfs.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return fs
}

func TestReadTool(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"hello.txt": "line one\nline two\nline three\n",
	})
	tool := readTool(fs)

	out, err := tool.Execute(t.Context(), `{"filePath": "hello.txt"}`)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.ToSlash(out) != "Path: hello.txt | Hash: 3b0bcfb848c53c9e5b3f462c738e5eefb208c0a1a1b0f60e0e6b0e5e3b0bcfb | Size: 26 bytes | Lines: 3\n---\n     1: line one\n     2: line two\n     3: line three\n" {
	}
	if !contains(t, out, "hello.txt") {
		t.Errorf("expected output to contain 'hello.txt', got:\n%s", out)
	}
	if !contains(t, out, "line two") {
		t.Errorf("expected output to contain 'line two', got:\n%s", out)
	}
}

func TestReadToolRange(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"hello.txt": "line one\nline two\nline three\nline four\n",
	})
	tool := readTool(fs)

	out, err := tool.Execute(t.Context(), `{"filePath": "hello.txt", "offset": 2, "limit": 2}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "line two") {
		t.Errorf("expected output to contain 'line two', got:\n%s", out)
	}
	if !contains(t, out, "line three") {
		t.Errorf("expected output to contain 'line three', got:\n%s", out)
	}
	if contains(t, out, "line one") {
		t.Errorf("expected output NOT to contain 'line one', got:\n%s", out)
	}
}

func TestReadToolNotFound(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{})
	tool := readTool(fs)

	_, err := tool.Execute(t.Context(), `{"filePath": "nonexistent.txt"}`)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestReadToolMissingPath(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{})
	tool := readTool(fs)

	_, err := tool.Execute(t.Context(), `{}`)
	if err == nil {
		t.Fatal("expected error for missing filePath")
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
