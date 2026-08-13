package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vesvai/vesvai/internal/filesystem"
	"github.com/vesvai/vesvai/internal/tui/components"
)

func TestBuildMentionCatalogUsesFilesystem(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(rel, content string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("src/main.go", "package main\n")
	mustWrite("src/util/helper.go", "package util\n")
	mustWrite("ignored.go", "secret\n")
	mustWrite("secret/data.go", "secret\n")
	mustWrite(".gitignore", "ignored.go\nsecret/\n")
	mustWrite(".hidden.go", "hidden\n")

	fsys, err := filesystem.New(filesystem.Config{RootDir: root, IgnoreDotfiles: true})
	if err != nil {
		t.Fatal(err)
	}

	catalog := buildMentionCatalog(fsys)
	ids := map[string]string{}
	for _, m := range catalog {
		ids[m.ID] = m.Kind
	}

	for _, a := range []string{"orchestrator", "planner", "explorer"} {
		if ids[a] != "agent" {
			t.Fatalf("agent %q missing from catalog", a)
		}
	}
	if ids["src/main.go"] != "file" || ids["src/util/helper.go"] != "file" {
		t.Fatalf("tracked files missing: %v", ids)
	}
	if ids["src"] != "dir" || ids["src/util"] != "dir" {
		t.Fatalf("folders missing: %v", ids)
	}
	for _, gone := range []string{"ignored.go", "secret/data.go", ".hidden.go"} {
		if _, ok := ids[gone]; ok {
			t.Fatalf("ignored/hidden file %q leaked into the catalog", gone)
		}
	}
}

func TestBuildMentionCatalogBounded(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "many"), 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxMentionEntries+50; i++ {
		if err := os.WriteFile(filepath.Join(root, "many", "f"+string(rune('a'+i%26))+string(rune('0'+i/26))+".go"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	fsys, err := filesystem.New(filesystem.Config{RootDir: root})
	if err != nil {
		t.Fatal(err)
	}
	catalog := buildMentionCatalog(fsys)
	if len(catalog) > 3+maxMentionEntries {
		t.Fatalf("catalog not bounded: %d entries", len(catalog))
	}
	_ = components.Mention{}
}

func TestBuildSkillCatalogUsesManager(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	root := t.TempDir()
	skillDir := filepath.Join(root, ".vesvai", "skills")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "refactor.md"),
		[]byte("---\nname: refactor\ndescription: refactor code\n---\n\nDo it.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	fsys, err := filesystem.New(filesystem.Config{RootDir: root})
	if err != nil {
		t.Fatal(err)
	}
	catalog := buildSkillCatalog(fsys)
	got := map[string]bool{}
	for _, m := range catalog {
		if m.Kind != "skill" {
			t.Fatalf("entry %q has kind %q, want skill", m.ID, m.Kind)
		}
		got[m.ID] = true
	}
	if !got["refactor"] {
		t.Fatalf("installed skill not in catalog: %v", got)
	}
	if got["graphify"] {
		t.Fatalf("fallback demo skills leaked in with real skills: %v", got)
	}

	catalog = buildSkillCatalog(nil)
	if len(catalog) != 0 {
		t.Fatalf("expected empty fallback catalog, got %v", catalog)
	}
}
