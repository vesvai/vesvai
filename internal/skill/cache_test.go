package skill

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vesvai/vesvai/internal/event"
	"github.com/vesvai/vesvai/internal/filesystem"
)

func newTestManager(t *testing.T) (*Manager, string, string) {
	t.Helper()
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global", "skills")
	projectDir := filepath.Join(tmpDir, "project", ".vesvai", "skills")

	os.MkdirAll(globalDir, 0755)
	os.MkdirAll(projectDir, 0755)

	fs, err := filesystem.New(filesystem.Config{RootDir: filepath.Join(tmpDir, "project")})
	if err != nil {
		t.Fatalf("failed to create filesystem: %v", err)
	}

	m := &Manager{
		globalDir:   globalDir,
		projectDir:  projectDir,
		projectRoot: filepath.Join(tmpDir, "project"),
		fs:          fs,
		index:       make(map[string]Skill),
		dirMTimes:   make(map[string]time.Time),
		descCache:   make(map[string]cachedDesc),
	}

	return m, globalDir, projectDir
}

func TestManager_List_NoContent(t *testing.T) {
	m, globalDir, _ := newTestManager(t)

	os.WriteFile(filepath.Join(globalDir, "lazy.md"),
		[]byte("---\ndescription: lazy skill\n---\n\nFull body here"), 0644)

	skills, err := m.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("List() = %d skills, want 1", len(skills))
	}
	if skills[0].Content != "" {
		t.Errorf("List() should not load content, got %q", skills[0].Content)
	}
	if skills[0].Description != "lazy skill" {
		t.Errorf("description = %q, want %q", skills[0].Description, "lazy skill")
	}

	read, err := m.Read("lazy")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if read.Content != "---\ndescription: lazy skill\n---\n\nFull body here" {
		t.Errorf("Read() content = %q, want full file content", read.Content)
	}
	if read.Description != "lazy skill" {
		t.Errorf("Read() description = %q, want %q", read.Description, "lazy skill")
	}
}

func TestManager_List_CachedIndex(t *testing.T) {
	m, globalDir, _ := newTestManager(t)

	os.WriteFile(filepath.Join(globalDir, "cached.md"), []byte("content"), 0644)

	if _, err := m.List(); err != nil {
		t.Fatalf("List() error = %v", err)
	}

	os.WriteFile(filepath.Join(globalDir, "new.md"), []byte("content"), 0644)

	skills, err := m.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(skills) != 2 {
		t.Errorf("List() = %d skills after new file, want 2", len(skills))
	}
}

func TestManager_Create_Invalidates(t *testing.T) {
	m, _, projectDir := newTestManager(t)

	if _, err := m.Create("invalidate-me", "content", LocationProject); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(projectDir, "invalidate-me.md")); err != nil {
		t.Fatalf("created file missing: %v", err)
	}

	skills, err := m.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	found := false
	for _, s := range skills {
		if s.Name == "invalidate-me" {
			found = true
		}
	}
	if !found {
		t.Error("created skill not visible after Create()")
	}
}

func TestManager_Delete_Invalidates(t *testing.T) {
	m, _, projectDir := newTestManager(t)

	os.WriteFile(filepath.Join(projectDir, "doomed.md"), []byte("content"), 0644)

	if !m.Exists("doomed") {
		t.Fatal("Exists(doomed) = false before delete")
	}

	if err := m.Delete("doomed", LocationProject); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if m.Exists("doomed") {
		t.Error("Exists(doomed) = true after delete")
	}
}

func TestManager_EventInvalidation(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	m, _, _ := newTestManager(t)
	m.SetEventBus(bus)

	if _, err := m.List(); err != nil {
		t.Fatalf("List() error = %v", err)
	}

	m.mu.RLock()
	scanned := m.scanned
	m.mu.RUnlock()
	if !scanned {
		t.Fatal("manager should be scanned after List()")
	}

	bus.Publish(context.Background(), event.NewSystemEvent(event.EventSkillChanged, map[string]interface{}{
		"name":   "any",
		"action": "changed",
	}))

	m.mu.RLock()
	scanned = m.scanned
	m.mu.RUnlock()
	if scanned {
		t.Error("manager should be invalidated after skill:changed event")
	}
}

func TestManager_Create_PublishesEvent(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	m, _, _ := newTestManager(t)
	m.SetEventBus(bus)

	received := make(chan event.Event, 1)
	bus.Subscribe(event.EventType(event.EventSkillChanged), event.EventHandlerFunc(func(ctx context.Context, e event.Event) error {
		received <- e
		return nil
	}))

	if _, err := m.Create("eventful", "content", LocationGlobal); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	select {
	case e := <-received:
		if e.Type() != event.EventType(event.EventSkillChanged) {
			t.Errorf("event type = %q, want %q", e.Type(), event.EventSkillChanged)
		}
	case <-time.After(time.Second):
		t.Fatal("skill:changed event not published on Create()")
	}
}
