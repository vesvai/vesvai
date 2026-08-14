package skill

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vesvai/vesvai/internal/config"
	"github.com/vesvai/vesvai/internal/event"
	"github.com/vesvai/vesvai/internal/filesystem"
)

const descHeadSize = 8192

type Manager struct {
	mu          sync.RWMutex
	globalDir   string
	extraDirs   []string
	projectDir  string
	projectRoot string
	fs          *filesystem.FileSystem
	eventBus    event.EventBus

	index     map[string]Skill
	dirMTimes map[string]time.Time
	descCache map[string]cachedDesc
	scanned   bool
}

type cachedDesc struct {
	size  int64
	mtime time.Time
	desc  string
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
		index:       make(map[string]Skill),
		dirMTimes:   make(map[string]time.Time),
		descCache:   make(map[string]cachedDesc),
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

func (m *Manager) globalDirs() []string {
	dirs := make([]string, 0, 1+len(m.extraDirs))
	dirs = append(dirs, m.globalDir)
	dirs = append(dirs, m.extraDirs...)
	return dirs
}

func (m *Manager) allDirs() []string {
	dirs := m.globalDirs()
	if m.projectDir != "" {
		dirs = append(dirs, m.projectDir)
	}
	return dirs
}

func (m *Manager) List() ([]Skill, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureScannedLocked(); err != nil {
		return nil, err
	}

	skills := make([]Skill, 0, len(m.index))
	for _, s := range m.index {
		skills = append(skills, s)
	}
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return skills, nil
}

func (m *Manager) ensureScannedLocked() error {
	if m.index == nil {
		m.index = make(map[string]Skill)
		m.dirMTimes = make(map[string]time.Time)
		m.descCache = make(map[string]cachedDesc)
	}

	if m.scanned {
		for _, dir := range m.allDirs() {
			fi, err := os.Stat(dir)
			if err != nil {
				continue
			}
			if last, ok := m.dirMTimes[dir]; !ok || !fi.ModTime().Equal(last) {
				return m.rescanLocked()
			}
		}
		return nil
	}

	return m.rescanLocked()
}

func (m *Manager) rescanLocked() error {
	index := make(map[string]Skill, len(m.index))

	for _, dir := range m.globalDirs() {
		if err := m.scanDir(dir, LocationGlobal, index, true, true); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to list global skills: %w", err)
		}
		m.recordDirMtime(dir)
	}

	if m.projectDir != "" {
		if err := m.scanDir(m.projectDir, LocationProject, index, false, false); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to list project skills: %w", err)
		}
		m.recordDirMtime(m.projectDir)
	}

	m.index = index
	m.scanned = true
	return nil
}

func (m *Manager) recordDirMtime(dir string) {
	if fi, err := os.Stat(dir); err == nil {
		m.dirMTimes[dir] = fi.ModTime()
	}
}

func (m *Manager) scanDir(dir string, location SkillLocation, index map[string]Skill, nested, firstWins bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		switch {
		case !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md"):
			name := strings.TrimSuffix(entry.Name(), ".md")
			path := filepath.Join(dir, entry.Name())
			if firstWins {
				if _, exists := index[name]; exists {
					continue
				}
			}
			index[name] = Skill{
				Name:        name,
				Location:    location,
				Path:        path,
				Description: m.descriptionOf(path),
			}

		case nested && entry.IsDir():
			path := filepath.Join(dir, entry.Name(), "SKILL.md")
			if _, err := os.Stat(path); err != nil {
				continue
			}
			name := entry.Name()
			if firstWins {
				if _, exists := index[name]; exists {
					continue
				}
			}
			index[name] = Skill{
				Name:        name,
				Location:    location,
				Path:        path,
				Description: m.descriptionOf(path),
			}
		}
	}

	return nil
}

func (m *Manager) descriptionOf(path string) string {
	fi, err := os.Stat(path)
	if err != nil {
		return ""
	}
	if c, ok := m.descCache[path]; ok && c.size == fi.Size() && c.mtime.Equal(fi.ModTime()) {
		return c.desc
	}

	desc := m.extractDescriptionHead(path)
	m.descCache[path] = cachedDesc{size: fi.Size(), mtime: fi.ModTime(), desc: desc}
	return desc
}

func (m *Manager) extractDescriptionHead(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, descHeadSize))
	if err != nil {
		return ""
	}
	return extractDescription(string(data))
}

func (m *Manager) Read(name string) (*Skill, error) {
	if m.projectDir != "" {
		path := filepath.Join(m.projectDir, name+".md")
		if data, err := os.ReadFile(path); err == nil {
			return m.skillFromData(name, LocationProject, path, data), nil
		}
	}

	for _, dir := range m.globalDirs() {
		for _, path := range []string{
			filepath.Join(dir, name+".md"),
			filepath.Join(dir, name, "SKILL.md"),
		} {
			if data, err := os.ReadFile(path); err == nil {
				return m.skillFromData(name, LocationGlobal, path, data), nil
			}
		}
	}

	return nil, fmt.Errorf("skill not found: %s", name)
}

func (m *Manager) skillFromData(name string, location SkillLocation, path string, data []byte) *Skill {
	content := string(data)
	return &Skill{
		Name:        name,
		Location:    location,
		Path:        path,
		Content:     content,
		Description: extractDescription(content),
	}
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

	m.Invalidate()
	m.publishSkillChanged(name, "created")

	return &Skill{
		Name:        name,
		Location:    location,
		Path:        path,
		Content:     content,
		Description: extractDescription(content),
	}, nil
}

func (m *Manager) Exists(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureScannedLocked(); err != nil {
		return false
	}
	_, ok := m.index[name]
	return ok
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

	if err := os.Remove(path); err != nil {
		return err
	}

	m.Invalidate()
	m.publishSkillChanged(name, "deleted")

	return nil
}

func (m *Manager) Invalidate() {
	m.mu.Lock()
	m.scanned = false
	m.mu.Unlock()
}

func (m *Manager) SetEventBus(bus event.EventBus) {
	m.mu.Lock()
	m.eventBus = bus
	m.mu.Unlock()

	if bus != nil {
		bus.Subscribe(event.EventType(event.EventSkillChanged), event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
			m.Invalidate()
			return nil
		}))
	}
}

func (m *Manager) publishSkillChanged(name, action string) {
	m.mu.RLock()
	bus := m.eventBus
	m.mu.RUnlock()

	if bus == nil {
		return
	}
	bus.Publish(context.Background(), event.NewSystemEvent(event.EventSkillChanged, map[string]interface{}{
		"name":   name,
		"action": action,
	}))
}

func StripFrontmatter(content string) string {
	trimmed := strings.TrimLeft(content, " \t\r\n")
	if !strings.HasPrefix(trimmed, "---") {
		return content
	}
	rest := trimmed
	if idx := strings.Index(rest, "\n"); idx >= 0 {
		rest = rest[idx+1:]
	}
	if parts := strings.SplitN(rest, "\n---", 2); len(parts) == 2 {
		return strings.TrimLeft(parts[1], "\r\n")
	}
	return content
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
