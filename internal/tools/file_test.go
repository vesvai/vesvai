package tools

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vesvai/vesvai/internal/filesystem"
)

func TestReadTool_Success(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	testContent := "line 1\nline 2\nline 3\n"
	if err := fs.Write(context.Background(), "test.txt", []byte(testContent)); err != nil {
		t.Fatal(err)
	}

	tool := newReadTool(fs)
	ctx := context.Background()

	result, err := tool.Handle(ctx, map[string]any{
		"filePath": filepath.Join(tmpDir, "test.txt"),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestReadTool_WithOffset(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	testContent := "line 1\nline 2\nline 3\n"
	if err := fs.Write(context.Background(), "test.txt", []byte(testContent)); err != nil {
		t.Fatal(err)
	}

	tool := newReadTool(fs)
	ctx := context.Background()

	result, err := tool.Handle(ctx, map[string]any{
		"filePath": filepath.Join(tmpDir, "test.txt"),
		"offset":   2,
		"limit":    1,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestReadTool_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	tool := newReadTool(fs)
	ctx := context.Background()

	_, err = tool.Handle(ctx, map[string]any{
		"filePath": filepath.Join(tmpDir, "nonexistent.txt"),
	})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestWriteTool_Success(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	tool := newWriteTool(fs)
	ctx := context.Background()

	result, err := tool.Handle(ctx, map[string]any{
		"filePath": filepath.Join(tmpDir, "new.txt"),
		"content":  "hello world",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}

	content, err := fs.Read(ctx, "new.txt")
	if err != nil {
		t.Fatal(err)
	}

	if content != "hello world\n" {
		t.Errorf("expected 'hello world\\n', got %q", content)
	}
}

func TestWriteTool_CreatesDirs(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	tool := newWriteTool(fs)
	ctx := context.Background()

	result, err := tool.Handle(ctx, map[string]any{
		"filePath": filepath.Join(tmpDir, "nested", "dir", "file.txt"),
		"content":  "content",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}

	content, err := fs.Read(ctx, "nested/dir/file.txt")
	if err != nil {
		t.Fatal(err)
	}

	if content != "content\n" {
		t.Errorf("expected 'content\\n', got %q", content)
	}
}

func TestEditTool_Success(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	testContent := "hello world"
	if err := fs.Write(context.Background(), "test.txt", []byte(testContent)); err != nil {
		t.Fatal(err)
	}

	tool := newEditTool(fs)
	ctx := context.Background()

	result, err := tool.Handle(ctx, map[string]any{
		"filePath":  filepath.Join(tmpDir, "test.txt"),
		"oldString": "world",
		"newString": "universe",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}

	content, err := fs.Read(ctx, "test.txt")
	if err != nil {
		t.Fatal(err)
	}

	if content != "hello universe\n" {
		t.Errorf("expected 'hello universe\\n', got %q", content)
	}
}

func TestEditTool_MultipleMatches(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	testContent := "abc abc abc"
	if err := fs.Write(context.Background(), "test.txt", []byte(testContent)); err != nil {
		t.Fatal(err)
	}

	tool := newEditTool(fs)
	ctx := context.Background()

	_, err = tool.Handle(ctx, map[string]any{
		"filePath":  filepath.Join(tmpDir, "test.txt"),
		"oldString": "abc",
		"newString": "xyz",
	})
	if err == nil {
		t.Fatal("expected error for multiple matches")
	}
}

func TestEditTool_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	testContent := "hello world"
	if err := fs.Write(context.Background(), "test.txt", []byte(testContent)); err != nil {
		t.Fatal(err)
	}

	tool := newEditTool(fs)
	ctx := context.Background()

	_, err = tool.Handle(ctx, map[string]any{
		"filePath":  filepath.Join(tmpDir, "test.txt"),
		"oldString": "nonexistent",
		"newString": "replacement",
	})
	if err == nil {
		t.Fatal("expected error for missing oldString")
	}
}
