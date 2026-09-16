package file

import (
	"testing"
)

func TestEditTool(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"hello.txt": "hello world\nfoo bar\nhello again\n",
	})
	_, err := fs.Read("hello.txt")
	if err != nil {
		t.Fatal(err)
	}

	tool := editTool(fs)
	_, err = tool.Execute(t.Context(), `{"filePath": "hello.txt", "oldString": "hello world", "newString": "hi world"}`)
	if err != nil {
		t.Fatal(err)
	}

	result, err := fs.Read("hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, result, "hi world") || !contains(t, result, "foo bar") || !contains(t, result, "hello again") {
		t.Errorf("expected 'hi world\\nfoo bar\\nhello again' in output, got %q", result)
	}
}

func TestEditToolReplaceAll(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"hello.txt": "hello world\nfoo bar\nhello again\n",
	})
	_, err := fs.Read("hello.txt")
	if err != nil {
		t.Fatal(err)
	}

	tool := editTool(fs)
	_, err = tool.Execute(t.Context(), `{"filePath": "hello.txt", "oldString": "hello", "newString": "hi", "replaceAll": true}`)
	if err != nil {
		t.Fatal(err)
	}

	result, err := fs.Read("hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, result, "hi world") || !contains(t, result, "foo bar") || !contains(t, result, "hi again") {
		t.Errorf("expected 'hi world\\nfoo bar\\nhi again' in output, got %q", result)
	}
}

func TestEditToolRequiresRead(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"hello.txt": "hello world",
	})
	tool := editTool(fs)

	_, err := tool.Execute(t.Context(), `{"filePath": "hello.txt", "oldString": "hello", "newString": "hi"}`)
	if err == nil {
		t.Fatal("expected error: file must be read before editing")
	}
}

func TestEditToolNoMatch(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"hello.txt": "hello world",
	})
	_, err := fs.Read("hello.txt")
	if err != nil {
		t.Fatal(err)
	}

	tool := editTool(fs)
	_, err = tool.Execute(t.Context(), `{"filePath": "hello.txt", "oldString": "nonexistent", "newString": "hi"}`)
	if err == nil {
		t.Fatal("expected error: pattern not found")
	}
}

func TestEditToolMissingPath(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{})
	tool := editTool(fs)

	_, err := tool.Execute(t.Context(), `{"oldString": "a", "newString": "b"}`)
	if err == nil {
		t.Fatal("expected error for missing filePath")
	}
}
