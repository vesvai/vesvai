package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManager_List(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global", "skills")
	projectDir := filepath.Join(tmpDir, "project", ".vesvai", "skills")

	os.MkdirAll(globalDir, 0755)
	os.MkdirAll(projectDir, 0755)

	os.WriteFile(filepath.Join(globalDir, "global-skill.md"), []byte("---\ndescription: A global skill\n---\n\nGlobal content"), 0644)
	os.WriteFile(filepath.Join(projectDir, "project-skill.md"), []byte("---\ndescription: A project skill\n---\n\nProject content"), 0644)
	os.WriteFile(filepath.Join(projectDir, "global-skill.md"), []byte("---\ndescription: Override global\n---\n\nOverride content"), 0644)

	m := &Manager{
		globalDir:   globalDir,
		projectDir:  projectDir,
		projectRoot: filepath.Join(tmpDir, "project"),
	}

	skills, err := m.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(skills) != 2 {
		t.Fatalf("List() returned %d skills, want 2", len(skills))
	}

	for _, s := range skills {
		if s.Name == "global-skill" {
			if s.Location != LocationProject {
				t.Errorf("global-skill should be from project, got %v", s.Location)
			}
			if s.Description != "Override global" {
				t.Errorf("global-skill description = %q, want %q", s.Description, "Override global")
			}
		}
		if s.Name == "project-skill" {
			if s.Location != LocationProject {
				t.Errorf("project-skill should be from project, got %v", s.Location)
			}
		}
	}
}

func TestManager_Read(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global", "skills")
	projectDir := filepath.Join(tmpDir, "project", ".vesvai", "skills")

	os.MkdirAll(globalDir, 0755)
	os.MkdirAll(projectDir, 0755)

	os.WriteFile(filepath.Join(globalDir, "test-skill.md"), []byte("Global skill content"), 0644)
	os.WriteFile(filepath.Join(projectDir, "test-skill.md"), []byte("Project skill content"), 0644)

	m := &Manager{
		globalDir:   globalDir,
		projectDir:  projectDir,
		projectRoot: filepath.Join(tmpDir, "project"),
	}

	skill, err := m.Read("test-skill")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if skill.Location != LocationProject {
		t.Errorf("Read() location = %v, want LocationProject", skill.Location)
	}
	if skill.Content != "Project skill content" {
		t.Errorf("Read() content = %q, want %q", skill.Content, "Project skill content")
	}

	_, err = m.Read("nonexistent")
	if err == nil {
		t.Error("Read() should return error for non-existent skill")
	}
}

func TestManager_Create(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global", "skills")
	projectDir := filepath.Join(tmpDir, "project", ".vesvai", "skills")

	os.MkdirAll(globalDir, 0755)
	os.MkdirAll(projectDir, 0755)

	m := &Manager{
		globalDir:   globalDir,
		projectDir:  projectDir,
		projectRoot: filepath.Join(tmpDir, "project"),
	}

	skill, err := m.Create("my-skill", "This is my skill content", LocationProject)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if skill.Name != "my-skill" {
		t.Errorf("Create() name = %q, want %q", skill.Name, "my-skill")
	}
	if skill.Location != LocationProject {
		t.Errorf("Create() location = %v, want LocationProject", skill.Location)
	}

	path := filepath.Join(projectDir, "my-skill.md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Create() did not create skill file")
	}

	_, err = m.Create("my-skill", "Duplicate", LocationProject)
	if err == nil {
		t.Error("Create() should return error for duplicate skill")
	}
}

func TestManager_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global", "skills")
	projectDir := filepath.Join(tmpDir, "project", ".vesvai", "skills")

	os.MkdirAll(globalDir, 0755)
	os.MkdirAll(projectDir, 0755)

	os.WriteFile(filepath.Join(globalDir, "exists-global.md"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(projectDir, "exists-project.md"), []byte("content"), 0644)

	m := &Manager{
		globalDir:   globalDir,
		projectDir:  projectDir,
		projectRoot: filepath.Join(tmpDir, "project"),
	}

	if !m.Exists("exists-global") {
		t.Error("Exists() should find global skill")
	}
	if !m.Exists("exists-project") {
		t.Error("Exists() should find project skill")
	}
	if m.Exists("nonexistent") {
		t.Error("Exists() should not find non-existent skill")
	}
}

func TestManager_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global", "skills")
	projectDir := filepath.Join(tmpDir, "project", ".vesvai", "skills")

	os.MkdirAll(globalDir, 0755)
	os.MkdirAll(projectDir, 0755)

	os.WriteFile(filepath.Join(projectDir, "to-delete.md"), []byte("content"), 0644)

	m := &Manager{
		globalDir:   globalDir,
		projectDir:  projectDir,
		projectRoot: filepath.Join(tmpDir, "project"),
	}

	err := m.Delete("to-delete", LocationProject)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if m.Exists("to-delete") {
		t.Error("Delete() did not remove skill")
	}

	err = m.Delete("nonexistent", LocationProject)
	if err == nil {
		t.Error("Delete() should return error for non-existent skill")
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"My Skill", "my-skill"},
		{"UPPERCASE", "uppercase"},
		{"special!@#chars", "specialchars"},
		{"with_underscores", "with_underscores"},
		{"  lots   of   spaces  ", "lots-of-spaces"},
	}

	for _, tt := range tests {
		got := sanitizeName(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractDescription(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "frontmatter",
			content: "---\ndescription: This is a test skill\n---\n\nContent here",
			want:    "This is a test skill",
		},
		{
			name:    "no frontmatter",
			content: "This is the first line of the skill",
			want:    "This is the first line of the skill",
		},
		{
			name:    "empty",
			content: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDescription(tt.content)
			if got != tt.want {
				t.Errorf("extractDescription() = %q, want %q", got, tt.want)
			}
		})
	}
}
