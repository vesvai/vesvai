package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vesvai/vesvai/internal/agent"
	"github.com/vesvai/vesvai/internal/agents"
	"github.com/vesvai/vesvai/internal/bootstrap"
	"github.com/vesvai/vesvai/internal/filesystem"
	"github.com/vesvai/vesvai/internal/skill"
	"github.com/vesvai/vesvai/internal/tui"
	"github.com/vesvai/vesvai/internal/tui/layouts"
)

func TestMentionAgentsFromRegistry(t *testing.T) {
	reg := agents.NewRegistry(agents.Config{})
	app := &bootstrap.App{AgentRegistry: reg}
	d := &AgentDriver{app: app}

	got := d.MentionAgents()
	if len(got) != 3 {
		t.Fatalf("agents = %d, want 3 (planner/explorer/orchestrator): %v", len(got), got)
	}
	want := map[string]bool{"planner": true, "explorer": true, "orchestrator": true}
	for _, m := range got {
		if m.Kind != "agent" {
			t.Fatalf("entry %q kind = %q, want agent", m.ID, m.Kind)
		}
		delete(want, m.ID)
	}
	if len(want) != 0 {
		t.Fatalf("missing agents: %v", want)
	}
}

func TestSlashCatalogFromSkillsOnly(t *testing.T) {
	toolReg := agent.NewToolRegistry()
	toolReg.Register(&readTool{})
	toolReg.Register(&streamTool{})
	app := &bootstrap.App{ToolRegistry: toolReg}

	d := &AgentDriver{app: app}
	got := d.SlashCatalog()
	if len(got) != 0 {
		t.Fatalf("slash catalog = %v, want empty (no tools, no mock skills)", got)
	}

	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	fsys, err := filesystem.New(filesystem.Config{RootDir: root})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".vesvai", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(root, ".vesvai", "skills", "graphify.md")
	if err := os.WriteFile(skillPath, []byte("# graphify\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr, err := skill.NewManager(root, fsys)
	if err != nil {
		t.Fatal(err)
	}
	d2 := &AgentDriver{app: app, skills: mgr}
	got = d2.SlashCatalog()
	if len(got) != 1 {
		t.Fatalf("slash catalog = %v, want exactly [graphify]", got)
	}
	if got[0].ID != "graphify" || got[0].Kind != "skill" {
		t.Fatalf("slash catalog entry = %+v, want skill graphify", got[0])
	}
}

func TestWireCatalogsUsesBackend(t *testing.T) {
	b := &stubBackend{}
	a := NewWithDriver(&stubDriver{stubBackend: b})
	a.model = tui.NewModel("demo")
	a.layout = layouts.NewMainLayout(a.model, tui.DefaultDark())
	a.backend = b

	a.wireCatalogs()

	mentions := a.layout.Textarea().MentionCatalog()
	foundAgent := false
	foundFile := false
	for _, m := range mentions {
		if m.ID == "explorer" && m.Kind == "agent" {
			foundAgent = true
		}
		if m.Kind == "file" || m.Kind == "dir" {
			foundFile = true
		}
	}
	if !foundAgent {
		t.Fatalf("hook agents missing from @ catalog: %v", mentions)
	}
	if !foundFile {
		t.Fatalf("file entries missing from @ catalog: %v", mentions)
	}

	slash := a.layout.Textarea().SkillCatalog()
	if len(slash) != 1 || slash[0].ID != "graphify" || slash[0].Kind != "skill" {
		t.Fatalf("slash catalog = %v, want only the skill graphify", slash)
	}
}
