package developer

import (
	"slices"
	"strings"
	"testing"

	"github.com/vesvai/vesvai/internal/agent/tools"
	"github.com/vesvai/vesvai/internal/builtin/tools/shell"
	"github.com/vesvai/vesvai/internal/builtin/tools/todo"
	"github.com/vesvai/vesvai/internal/builtin/tools/web"
	"github.com/vesvai/vesvai/internal/vfs"
)

func newTestFS(t *testing.T) *vfs.VFS {
	t.Helper()
	root := t.TempDir()
	fs, err := vfs.New(root, vfs.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Write("main.go", []byte("package main\n")); err != nil {
		t.Fatal(err)
	}
	return fs
}

func TestDeveloperAgentTools(t *testing.T) {
	fs := newTestFS(t)
	todo.TodoTools(fs)
	shell.ShellTools(fs)
	web.WebTools(fs)

	a, err := newDeveloperAgent(fs)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"read", "write", "edit", "list", "glob", "grep"} {
		if _, ok := a.Tools.Get(name); !ok {
			t.Fatalf("developer missing file tool %q", name)
		}
	}
	for _, name := range []string{"bash", "web-fetch", "web-search", "list-todo", "update-todo"} {
		if !slices.Contains(a.ToolNames, name) {
			t.Fatalf("developer missing tool name %q", name)
		}
		if _, ok := tools.Get(name); !ok {
			t.Fatalf("tool %q not registered in global registry", name)
		}
	}
}

func TestDeveloperPromptRenders(t *testing.T) {
	sys, err := generateDeveloperPrompt()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"software engineer", ".vesvai/plans", "update-todo", "in_progress", "Report"} {
		if !strings.Contains(sys, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}
}
