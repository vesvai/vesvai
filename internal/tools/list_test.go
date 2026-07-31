package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/vesvai/vesvai/internal/filesystem"
)

func TestListTool_Success(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	fs.Write(context.Background(), "file1.txt", []byte("content1"))
	fs.Write(context.Background(), "file2.txt", []byte("content2"))

	tool := newListTool(fs)
	ctx := context.Background()

	result, err := tool.Handle(ctx, map[string]any{
		"path": tmpDir,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestListTool_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	tool := newListTool(fs)
	ctx := context.Background()

	result, err := tool.Handle(ctx, map[string]any{
		"path": tmpDir,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != "(empty directory)\n" {
		t.Errorf("expected '(empty directory)\\n', got %q", result)
	}
}

func TestListTool_Subdirectories(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	fs.Write(context.Background(), "dir/file.txt", []byte("content"))
	fs.Write(context.Background(), "file.txt", []byte("content"))

	tool := newListTool(fs)
	ctx := context.Background()

	result, err := tool.Handle(ctx, map[string]any{
		"path": tmpDir,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestListTool_ShowsLineCounts(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	fs.Write(context.Background(), "multi.go", []byte("package main\n\nfunc main() {\n\tprintln()\n}\n"))
	fs.Write(context.Background(), "single.txt", []byte("one line"))
	fs.Write(context.Background(), "empty.go", nil)

	tool := newListTool(fs)
	result, err := tool.Handle(context.Background(), map[string]any{"path": tmpDir})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(result, "multi.go (") || !strings.Contains(result, "5 lines") {
		t.Errorf("expected line count for multi.go in output, got:\n%s", result)
	}
	if !strings.Contains(result, "single.txt (") || !strings.Contains(result, "1 lines") {
		t.Errorf("expected line count for single.txt in output, got:\n%s", result)
	}
	if !strings.Contains(result, "empty.go (") {
		t.Errorf("expected empty.go in output, got:\n%s", result)
	}
}
