package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Manager struct {
	globalDir   string
	projectDir  string
	projectRoot string
}

func NewManager(projectRoot string) (*Manager, error) {
	globalDir, err := GlobalSkillsDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get global skills dir: %w", err)
	}

	var projectDir string
	if projectRoot != "" {
		projectDir = ProjectSkillsDir(projectRoot)
	}

	return &Manager{
		globalDir:   globalDir,
		projectDir:  projectDir,
		projectRoot: projectRoot,
	}, nil
}

func (m *Manager) List() ([]Skill, error) {
	skillMap := make(map[string]Skill)

	globalSkills, err := m.loadSkillsFromDir(m.globalDir, LocationGlobal)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to list global skills: %w", err)
	}
	for _, s := range globalSkills {
		skillMap[s.Name] = s
	}

	if m.projectDir != "" {
		projectSkills, err := m.loadSkillsFromDir(m.projectDir, LocationProject)
		if err != nil && !os.IsNotExist(err) {
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
		if data, err := os.ReadFile(path); err == nil {
			return &Skill{
				Name:        name,
				Location:    LocationProject,
				Path:        path,
				Content:     string(data),
				Description: extractDescription(string(data)),
			}, nil
		}
	}

	path := filepath.Join(m.globalDir, name+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("skill not found: %s", name)
	}

	return &Skill{
		Name:        name,
		Location:    LocationGlobal,
		Path:        path,
		Content:     string(data),
		Description: extractDescription(string(data)),
	}, nil
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

	path := filepath.Join(m.globalDir, name+".md")
	_, err := os.Stat(path)
	return err == nil
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
