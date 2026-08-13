package app

import (
	"context"
	"testing"

	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/session"
	"github.com/vesvai/vesvai/internal/tui"
	"github.com/vesvai/vesvai/internal/tui/components"
	"github.com/vesvai/vesvai/internal/tui/layouts"
)

type stubBackend struct {
	models  []tui.ModelInfo
	chosen  string
	sess    []components.Session
	deleted []string
}

func (s *stubBackend) ListSessions() ([]components.Session, error) { return s.sess, nil }
func (s *stubBackend) LoadSession(id string) (*session.Session, error) {
	return &session.Session{ID: id, Messages: []llm.Message{}}, nil
}
func (s *stubBackend) DeleteSession(id string) error {
	s.deleted = append(s.deleted, id)
	return nil
}
func (s *stubBackend) SaveSession(*session.Session) error             { return nil }
func (s *stubBackend) Models() []tui.ModelInfo                        { return s.models }
func (s *stubBackend) SetModel(name string)                           { s.chosen = name }
func (s *stubBackend) NewSessionID() string                           { return "session_test" }
func (s *stubBackend) CurrentHistory(*tui.Conversation) []llm.Message { return nil }
func (s *stubBackend) ConnectProvider(name, key string) error         { return nil }
func (s *stubBackend) SupportedProviders() []string {
	return []string{"openrouter", "openai"}
}
func (s *stubBackend) MentionAgents() []components.Mention {
	return []components.Mention{
		{ID: "planner", Kind: "agent", Label: "planner"},
		{ID: "explorer", Kind: "agent", Label: "explorer"},
		{ID: "orchestrator", Kind: "agent", Label: "orchestrator"},
	}
}
func (s *stubBackend) SlashCatalog() []components.Mention {
	return []components.Mention{
		{ID: "graphify", Kind: "skill", Label: "graphify"},
	}
}

type stubDriver struct {
	*stubBackend
	cancelled bool
}

func (d *stubDriver) Run(ctx context.Context, req RunRequest, emit func(tui.StreamEvent)) {}
func (d *stubDriver) Cancel()                                                             { d.cancelled = true }

func TestSeedBackendPopulatesModelPicker(t *testing.T) {
	models := []tui.ModelInfo{
		{Name: "qwen/qwen3.7-flash", Provider: "openrouter"},
		{Name: "deepseek/deepseek-v4-flash", Provider: "openrouter"},
		{Name: "anthropic/claude-opus-4.5", Provider: "openrouter"},
	}
	b := &stubBackend{models: models}
	d := &stubDriver{stubBackend: b}
	a := NewWithDriver(d)
	a.model = tui.NewModel("demo")
	a.layout = layouts.NewMainLayout(a.model, tui.DefaultDark())

	a.seedBackend(b)

	if a.model.Model != "qwen/qwen3.7-flash" {
		t.Fatalf("default model = %q, want qwen/qwen3.7-flash", a.model.Model)
	}
	if b.chosen != "qwen/qwen3.7-flash" {
		t.Fatalf("driver model = %q, want qwen/qwen3.7-flash", b.chosen)
	}
	got := a.layout.Models()
	if len(got) != 3 {
		t.Fatalf("picker catalog = %d models, want 3", len(got))
	}
	for i, m := range models {
		if got[i].Name != m.Name {
			t.Fatalf("picker[%d] = %q, want %q", i, got[i].Name, m.Name)
		}
	}
}

func TestSeedBackendEmptyCatalogKeepsPickerEmpty(t *testing.T) {
	b := &stubBackend{}
	d := &stubDriver{stubBackend: b}
	a := NewWithDriver(d)
	a.model = tui.NewModel("demo")
	a.layout = layouts.NewMainLayout(a.model, tui.DefaultDark())

	a.seedBackend(b)

	if a.model.Model != "" {
		t.Fatalf("model = %q, want empty", a.model.Model)
	}
	if len(a.layout.Models()) != 0 {
		t.Fatalf("picker catalog = %d, want 0", len(a.layout.Models()))
	}
}
