package orchestrator

import (
	"slices"
	"strings"
	"testing"

	"github.com/vesvai/vesvai/internal/agent/tools"
	"github.com/vesvai/vesvai/internal/builtin/tools/subagent"
	"github.com/vesvai/vesvai/internal/builtin/tools/todo"
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

func TestOrchestratorAgentTools(t *testing.T) {
	fs := newTestFS(t)
	todo.TodoTools(fs)
	subagent.SubAgentTools(nil)

	a, err := newOrchestratorAgent(fs)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"read", "list", "glob"} {
		if _, ok := a.Tools.Get(name); !ok {
			t.Fatalf("orchestrator missing file tool %q", name)
		}
	}
	for _, name := range []string{"subagent", "wait-for-subagents", "subagents-status", "subagent-message", "list-todo", "update-todo"} {
		if !slices.Contains(a.ToolNames, name) {
			t.Fatalf("orchestrator missing tool name %q", name)
		}
		if _, ok := tools.Get(name); !ok {
			t.Fatalf("tool %q not registered in global registry", name)
		}
	}
}

func TestOrchestratorPromptRenders(t *testing.T) {
	sys, err := generateOrchestratorPrompt()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"master orchestrator", "explorer", "planner", "developer", "subagent", "wait-for-subagents", "list-todo", "update-todo", "task_id", ".vesvai/plans", "Todo Status", "Skills", "/<skill-name>"} {
		if !strings.Contains(sys, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}
}