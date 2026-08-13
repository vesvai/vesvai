package skill

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vesvai/vesvai/internal/config"
	"github.com/vesvai/vesvai/internal/filesystem"
)

type Manager struct {
	globalDir   string
	extraDirs   []string
	projectDir  string
	projectRoot string
	fs          *filesystem.FileSystem
}

func NewManager(projectRoot string, fs *filesystem.FileSystem) (*Manager, error) {
	globalDir, err := config.GetSkillsDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get global skills dir: %w", err)
	}

	var projectDir string
	if projectRoot != "" {
		projectDir = ProjectSkillsDir(projectRoot)
	}

	return &Manager{
		globalDir:   globalDir,
		extraDirs:   agentSkillDirs(),
		projectDir:  projectDir,
		projectRoot: projectRoot,
		fs:          fs,
	}, nil
}

func agentSkillDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var dirs []string
	for _, rel := range []string{
		filepath.Join(home, ".agents", "skills"),
		filepath.Join(home, ".agent", "skills"),
		filepath.Join(home, ".claude", "skills"),
		filepath.Join(home, ".config", "opencode", "skills"),
	} {
		if fi, err := os.Stat(rel); err == nil && fi.IsDir() {
			dirs = append(dirs, rel)
		}
	}
	return dirs
}

func (m *Manager) List() ([]Skill, error) {
	skillMap := make(map[string]Skill)

	for _, dir := range append([]string{m.globalDir}, m.extraDirs...) {
		skills, err := m.loadSkillsFromDir(dir, LocationGlobal)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to list global skills: %w", err)
		}
		for _, s := range skills {
			if _, exists := skillMap[s.Name]; !exists {
				skillMap[s.Name] = s
			}
		}
	}

	if m.projectDir != "" {
		projectSkills, err := m.loadProjectSkillsFromDir(m.projectDir, LocationProject)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to list project skills: %w", err)
		}
		for _, s := range projectSkills {
			skillMap[s.Name] = s
		}
	}

	skills := make([]Skill, 0, len(skillMap))
	for _, s := range skillMap {
		skills = append(skills, s)
	}
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return skills, nil
}

func (m *Manager) Read(name string) (*Skill, error) {
	if m.projectDir != "" {
		path := filepath.Join(m.projectDir, name+".md")
		if data, err := m.readProjectFile(path); err == nil {
			return &Skill{
				Name:        name,
				Location:    LocationProject,
				Path:        path,
				Content:     data,
				Description: extractDescription(data),
			}, nil
		}
	}

	for _, dir := range append([]string{m.globalDir}, m.extraDirs...) {
		if s, ok := m.readFromDir(dir, name, LocationGlobal); ok {
			return s, nil
		}
	}

	return nil, fmt.Errorf("skill not found: %s", name)
}

func (m *Manager) readFromDir(dir, name string, location SkillLocation) (*Skill, bool) {
	paths := []string{
		filepath.Join(dir, name+".md"),
		filepath.Join(dir, name, "SKILL.md"),
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		return &Skill{
			Name:        name,
			Location:    location,
			Path:        path,
			Content:     string(data),
			Description: extractDescription(string(data)),
		}, true
	}
	return nil, false
}

func (m *Manager) Create(name, content string, location SkillLocation) (*Skill, error) {
	if name == "" {
		return nil, fmt.Errorf("skill name is required")
	}

	name = sanitizeName(name)
	if name == "" {
		return nil, fmt.Errorf("invalid skill name")
	}

	var targetDir string
	switch location {
	case LocationGlobal:
		targetDir = m.globalDir
	case LocationProject:
		if m.projectDir == "" {
			return nil, fmt.Errorf("project root not set, cannot create project skill")
		}
		targetDir = m.projectDir
	default:
		return nil, fmt.Errorf("invalid location: %d", location)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create skills directory: %w", err)
	}

	path := filepath.Join(targetDir, name+".md")
	if _, err := os.Stat(path); err == nil {
		return nil, fmt.Errorf("skill already exists: %s", name)
	}

	if !strings.HasPrefix(content, "---") {
		content = fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s",
			name, extractDescription(content), content)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write skill file: %w", err)
	}

	return &Skill{
		Name:        name,
		Location:    location,
		Path:        path,
		Content:     content,
		Description: extractDescription(content),
	}, nil
}

func (m *Manager) Exists(name string) bool {
	if m.projectDir != "" {
		path := filepath.Join(m.projectDir, name+".md")
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	for _, dir := range append([]string{m.globalDir}, m.extraDirs...) {
		if _, ok := m.readFromDir(dir, name, LocationGlobal); ok {
			return true
		}
	}
	return false
}

func (m *Manager) Delete(name string, location SkillLocation) error {
	var targetDir string
	switch location {
	case LocationGlobal:
		targetDir = m.globalDir
	case LocationProject:
		if m.projectDir == "" {
			return fmt.Errorf("project root not set")
		}
		targetDir = m.projectDir
	default:
		return fmt.Errorf("invalid location: %d", location)
	}

	path := filepath.Join(targetDir, name+".md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("skill not found: %s", name)
	}

	return os.Remove(path)
}

func (m *Manager) loadSkillsFromDir(dir string, location SkillLocation) ([]Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var skills []Skill
	for _, entry := range entries {
		switch {
		case !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md"):
			name := strings.TrimSuffix(entry.Name(), ".md")
			path := filepath.Join(dir, entry.Name())

			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			skills = append(skills, Skill{
				Name:        name,
				Location:    location,
				Path:        path,
				Content:     string(data),
				Description: extractDescription(string(data)),
			})

		case entry.IsDir():
			path := filepath.Join(dir, entry.Name(), "SKILL.md")
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			skills = append(skills, Skill{
				Name:        entry.Name(),
				Location:    location,
				Path:        path,
				Content:     string(data),
				Description: extractDescription(string(data)),
			})
		}
	}

	return skills, nil
}

func (m *Manager) loadProjectSkillsFromDir(dir string, location SkillLocation) ([]Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var skills []Skill
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".md")
		path := filepath.Join(dir, entry.Name())

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		skills = append(skills, Skill{
			Name:        name,
			Location:    location,
			Path:        path,
			Content:     string(data),
			Description: extractDescription(string(data)),
		})
	}

	return skills, nil
}

func (m *Manager) readProjectFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func extractDescription(content string) string {
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) >= 2 {
			for _, line := range strings.Split(parts[1], "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "description:") {
					return strings.TrimSpace(strings.TrimPrefix(line, "description:"))
				}
			}
		}
	}

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "---") {
			continue
		}
		if len(line) > 200 {
			return line[:200] + "..."
		}
		return line
	}

	return ""
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " ", "-")
	var b strings.Builder
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			b.WriteRune(c)
		}
	}
	result := strings.ToLower(b.String())
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	return strings.Trim(result, "-")
}
