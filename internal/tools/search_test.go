package tools

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vesvai/vesvai/internal/filesystem"
)

func TestGlobTool_Success(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	fs.Write(context.Background(), "file1.go", []byte("package main"))
	fs.Write(context.Background(), "file2.go", []byte("package main"))
	fs.Write(context.Background(), "file3.txt", []byte("text"))

	tool := newGlobTool(fs)
	ctx := context.Background()

	result, err := tool.Handle(ctx, map[string]any{
		"pattern": "*.go",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestGlobTool_NoMatches(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	tool := newGlobTool(fs)
	ctx := context.Background()

	result, err := tool.Handle(ctx, map[string]any{
		"pattern": "*.xyz",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != "No files found matching pattern\n" {
		t.Errorf("expected 'No files found matching pattern\\n', got %q", result)
	}
}

func TestGlobTool_WithPath(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	fs.Write(context.Background(), "subdir/file.go", []byte("package main"))

	tool := newGlobTool(fs)
	ctx := context.Background()

	result, err := tool.Handle(ctx, map[string]any{
		"pattern": "*.go",
		"path":    filepath.Join(tmpDir, "subdir"),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestGrepTool_Content(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	fs.Write(context.Background(), "test.go", []byte("package main\n\nfunc main() {}\n"))

	tool := newGrepTool(fs)
	ctx := context.Background()

	result, err := tool.Handle(ctx, map[string]any{
		"pattern":     "func main",
		"output_mode": "content",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestGrepTool_FilesOnly(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	fs.Write(context.Background(), "test.go", []byte("package main"))

	tool := newGrepTool(fs)
	ctx := context.Background()

	result, err := tool.Handle(ctx, map[string]any{
		"pattern":     "package",
		"output_mode": "files_with_matches",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestGrepTool_Count(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	fs.Write(context.Background(), "test.go", []byte("line1\nline2\nline3\n"))

	tool := newGrepTool(fs)
	ctx := context.Background()

	result, err := tool.Handle(ctx, map[string]any{
		"pattern":     "line",
		"output_mode": "count",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestGrepTool_Include(t *testing.T) {
	tmpDir := t.TempDir()

	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	fs.Write(context.Background(), "test.go", []byte("package main"))
	fs.Write(context.Background(), "test.txt", []byte("package main"))

	tool := newGrepTool(fs)
	ctx := context.Background()

	result, err := tool.Handle(ctx, map[string]any{
		"pattern": "package",
		"include": "*.go",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestGlobTool_EmptyPathRecursive(t *testing.T) {
	tmpDir := t.TempDir()
	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}
	fs.Write(context.Background(), "root.go", []byte("package main\n"))
	fs.Write(context.Background(), "cmd/sub/deep.go", []byte("package sub\n"))
	fs.Write(context.Background(), "cmd/other.go", []byte("package cmd\n"))

	tool := newGlobTool(fs)
	result, err := tool.Handle(context.Background(), map[string]any{
		"path":    "",
		"pattern": "*.go",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(result, "cmd/sub/deep.go") || !strings.Contains(result, "root.go") {
		t.Errorf("expected recursive matches in result, got:\n%s", result)
	}
}

func TestGlobTool_PathWithPatternRecursive(t *testing.T) {
	tmpDir := t.TempDir()
	fs, err := filesystem.New(filesystem.Config{RootDir: tmpDir})
	if err != nil {
		t.Fatal(err)
	}
	fs.Write(context.Background(), "src/a.go", []byte("package a\n"))
	fs.Write(context.Background(), "src/deep/b.go", []byte("package b\n"))
	fs.Write(context.Background(), "other/c.go", []byte("package c\n"))

	tool := newGlobTool(fs)
	result, err := tool.Handle(context.Background(), map[string]any{
		"path":    "src",
		"pattern": "*.go",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(result, "src/deep/b.go") || strings.Contains(result, "other/c.go") {
		t.Errorf("expected recursive matches under src only, got:\n%s", result)
	}
}
