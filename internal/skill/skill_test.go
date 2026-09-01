package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSkill(t *testing.T, dir, name, content string) {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, SkillFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseSKILL(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "pdf-tools", `---
name: pdf-tools
description: Extract text from PDF files. Use when handling PDFs.
license: MIT
compatibility: Requires pdfplumber
metadata:
  author: example-org
  version: "1.0"
allowed-tools: "Bash(python:*) Read"
---

# PDF Tools

Extract text with pdfplumber.
`)
	dir := filepath.Join(root, "pdf-tools")
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scripts", "extract.py"), []byte("print('x')"), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := parseSKILL(filepath.Join(dir, SkillFileName), "test")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "pdf-tools" || s.Description == "" {
		t.Fatalf("name/description = %q / %q", s.Name, s.Description)
	}
	if s.License != "MIT" || s.Compatibility == "" {
		t.Fatalf("license/compatibility = %q / %q", s.License, s.Compatibility)
	}
	if s.Metadata["author"] != "example-org" || s.Metadata["version"] != "1.0" {
		t.Fatalf("metadata = %v", s.Metadata)
	}
	if len(s.AllowedTools) != 2 || s.AllowedTools[0] != "Bash(python:*)" {
		t.Fatalf("allowed-tools = %v", s.AllowedTools)
	}
	if strings.Contains(s.Instructions, "name:") || !strings.Contains(s.Instructions, "Extract text with pdfplumber") {
		t.Fatalf("instructions must be body without frontmatter:\n%s", s.Instructions)
	}
	if !s.HasScripts || !strings.HasSuffix(s.ScriptsPath, "scripts") {
		t.Fatalf("scripts: has=%v path=%q", s.HasScripts, s.ScriptsPath)
	}
}

func TestParseSKILLNoFrontmatter(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "bad-skill", "# No frontmatter here\n")
	if _, err := parseSKILL(filepath.Join(root, "bad-skill", SkillFileName), "test"); err == nil {
		t.Fatal("want error for missing frontmatter")
	}
}

func TestParseSKILLNameFallback(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "fallback-name", `---
description: No name field
---

Body here.
`)
	s, err := parseSKILL(filepath.Join(root, "fallback-name", SkillFileName), "test")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "fallback-name" {
		t.Fatalf("name = %q, want dir name", s.Name)
	}
}

func TestRegistryErrors(t *testing.T) {
	if err := Register(nil); !isErr(err, ErrNilSkill) {
		t.Fatalf("err = %v, want ErrNilSkill", err)
	}
	if err := Register(&Skill{Name: ""}); !isErr(err, ErrEmptyName) {
		t.Fatalf("err = %v, want ErrEmptyName", err)
	}
	if err := Register(&Skill{Name: "Bad_Name"}); !isErr(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	if err := Register(&Skill{Name: "ok-skill"}); err != nil {
		t.Fatal(err)
	}
	if err := Register(&Skill{Name: "ok-skill"}); !isErr(err, ErrDuplicate) {
		t.Fatalf("err = %v, want ErrDuplicate", err)
	}
}

func isErr(err error, want error) bool {
	return err != nil && strings.Contains(err.Error(), want.Error())
}

func TestLoadDirAndPrecedence(t *testing.T) {
	low := t.TempDir()
	high := t.TempDir()
	writeSkill(t, low, "shared", "---\nname: shared\ndescription: low version\n---\nLow body.\n")
	writeSkill(t, low, "unique-low", "---\nname: unique-low\ndescription: only in low\n---\nLow.\n")
	writeSkill(t, high, "shared", "---\nname: shared\ndescription: high version\n---\nHigh body.\n")
	writeSkill(t, high, "broken", "no frontmatter")

	if err := LoadDirs(low, high); err != nil {
		t.Fatal(err)
	}

	s, ok := Get("shared")
	if !ok {
		t.Fatal("shared must be registered")
	}
	if s.Source != high {
		t.Fatalf("source = %q, want high-precedence dir %q", s.Source, high)
	}
	if !strings.Contains(s.Instructions, "High body") {
		t.Fatalf("instructions = %q, want high version", s.Instructions)
	}
	if _, ok := Get("unique-low"); !ok {
		t.Fatal("unique-low must be registered")
	}
	if Has("broken") {
		t.Fatal("broken skill must be skipped")
	}
}

func TestLoadDirMissingIsNoop(t *testing.T) {
	if err := LoadDir(filepath.Join(t.TempDir(), "nope"), "x"); err != nil {
		t.Fatalf("missing dir must be a noop, got %v", err)
	}
}

func TestExpandMessage(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "executing-plans", `---
name: executing-plans
description: Use when executing a written plan.
---

# Executing Plans

Load the plan, review it, execute tasks.
`)
	if err := LoadDirs(root); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		in   string
		want string
	}{
		{"Use the /executing-plans skill", "Executing Plans"},
		{"Unknown /no-such-skill stays", "/no-such-skill"},
		{"URL https://example.com/foo must survive", "https://example.com/foo"},
		{"Path a/b/c must survive", "a/b/c"},
		{"No skills here", "No skills here"},
	}
	for _, tc := range cases {
		got := ExpandMessage(tc.in)
		if !strings.Contains(got, tc.want) {
			t.Fatalf("ExpandMessage(%q) = %q, want it to contain %q", tc.in, got, tc.want)
		}
	}

	got := ExpandMessage("Use /executing-plans now")
	if strings.Contains(got, "---\nname:") {
		t.Fatalf("expanded content must not contain frontmatter:\n%s", got)
	}
	if strings.Contains(got, "Use /executing-plans") {
		t.Fatalf("token must be replaced:\n%s", got)
	}
	if !strings.Contains(got, "<skill:executing-plans>") {
		t.Fatalf("skill block missing:\n%s", got)
	}
	if got != ExpandMessage(got) {
		t.Fatalf("expansion must be idempotent")
	}
}
