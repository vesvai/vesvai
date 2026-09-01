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
	skillMD := filepath.Join(root, "go-development", SkillFileName)
	data, err := os.ReadFile(skillMD)
	if err != nil {
		t.Fatalf("SKILL.md not materialized: %v", err)
	}
	if !strings.Contains(string(data), "go-development") {
		t.Fatalf("SKILL.md content unexpected:\n%s", data)
	}
	script := filepath.Join(root, "go-development", "scripts", "run-tests.sh")
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("scripts not materialized: %v", err)
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

	s, ok := Get("go-development")
	if !ok {
		t.Fatal("go-development must be registered")
	}
	if !s.HasScripts || s.ScriptsPath == "" {
		t.Fatalf("scripts info missing: %+v", s)
	}

	out := ExpandMessage("implement it /go-development")
	if !strings.Contains(out, "<skill:go-development>") {
		t.Fatalf("expansion failed:\n%s", out)
	}
	if !strings.Contains(out, "Verification") {
		t.Fatalf("skill body missing from expansion:\n%s", out)
	}

	infos := []prompt.SkillInfo{{Name: s.Name, Description: s.Description}}
	md, err := prompt.New().Skills(infos).Build(prompt.FormatMarkdown)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "go-development") {
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
