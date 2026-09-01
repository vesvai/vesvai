package file

import (
	"testing"
)

func TestWriteTool(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{})
	tool := writeTool(fs)

	out, err := tool.Execute(t.Context(), `{"filePath": "newfile.txt", "content": "hello world"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "newfile.txt") {
		t.Errorf("expected output to contain 'newfile.txt', got:\n%s", out)
	}
	if !contains(t, out, "written") {
		t.Errorf("expected output to contain 'written', got:\n%s", out)
	}

	result, err := fs.Read("newfile.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, result, "hello world") {
		t.Errorf("expected content 'hello world' in output, got %q", result)
	}
}

func TestWriteToolMissingPath(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{})
	tool := writeTool(fs)

	_, err := tool.Execute(t.Context(), `{"content": "hello"}`)
	if err == nil {
		t.Fatal("expected error for missing filePath")
	}
}

func TestWriteToolOverwrite(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"existing.txt": "old content",
	})
	tool := writeTool(fs)

	_, err := tool.Execute(t.Context(), `{"filePath": "existing.txt", "content": "new content"}`)
	if err != nil {
		t.Fatal(err)
	}

	result, err := fs.Read("existing.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, result, "new content") {
		t.Errorf("expected content 'new content' in output, got %q", result)
	}
}
