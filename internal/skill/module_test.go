package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vesvai/vesvai/internal/agent/prompt"
)

func TestMaterializeTo(t *testing.T) {
	root := t.TempDir()
	if err := MaterializeTo(root); err != nil {
		t.Fatal(err)
	}
	skillMD := filepath.Join(root, "init", SkillFileName)
	data, err := os.ReadFile(skillMD)
	if err != nil {
		t.Fatalf("SKILL.md not materialized: %v", err)
	}
	if !strings.Contains(string(data), "init") {
		t.Fatalf("SKILL.md content unexpected:\n%s", data)
	}

	if err := MaterializeTo(root); err != nil {
		t.Fatalf("materialize must be idempotent: %v", err)
	}
}

func TestMaterializedSkillParsesAndExpands(t *testing.T) {
	root := t.TempDir()
	if err := MaterializeTo(root); err != nil {
		t.Fatal(err)
	}
	if err := LoadDirs(root); err != nil {
		t.Fatal(err)
	}

	s, ok := Get("init")
	if !ok {
		t.Fatal("init must be registered")
	}

	out := ExpandMessage("implement it /init")
	if !strings.Contains(out, "<skill:init>") {
		t.Fatalf("expansion failed:\n%s", out)
	}

	infos := []prompt.SkillInfo{{Name: s.Name, Description: s.Description}}
	md, err := prompt.New().Skills(infos).Build(prompt.FormatMarkdown)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "init") {
		t.Fatalf("prompt listing missing skill: %q", md)
	}
}

func TestSkillModule(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := SkillModule(); err != nil {
		t.Fatalf("SkillModule: %v", err)
	}
	if err := SkillModule(); err != nil {
		t.Fatalf("SkillModule must be idempotent: %v", err)
	}
}
