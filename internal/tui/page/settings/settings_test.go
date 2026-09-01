package settings

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/tui/components"
	"github.com/vesvai/vesvai/internal/tui/layout"
	"github.com/vesvai/vesvai/internal/tui/styles"
)

func drawSettings(t *testing.T, w, h int, s *Settings) {
	t.Helper()
	screen := newSim(t, w, h)
	s.Draw(screen, layout.Region{Left: 0, Top: 0, Width: w, Height: h}, true)
	screen.Show()
}

func newSim(t *testing.T, w, h int) tcell.Screen {
	t.Helper()
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("init simulation screen: %v", err)
	}
	t.Cleanup(s.Fini)
	s.SetSize(w, h)
	return s
}

func TestSettingsDrawSmoke(t *testing.T) {
	styles.RegisterDefaults()
	styles.Set("dark")
	for _, size := range [][2]int{{80, 24}, {120, 40}, {50, 15}, {200, 50}} {
		drawSettings(t, size[0], size[1], New(Deps{}))
	}
}

func TestSettingsTabs(t *testing.T) {
	s := New(Deps{})
	if s.tab != tabGeneral {
		t.Fatalf("initial tab = %v, want general", s.tab)
	}
	s.HandleKey(tcell.NewEventKey(tcell.KeyTab, 0, 0))
	if s.tab != tabSession {
		t.Errorf("after Tab tab = %v, want Session", s.tab)
	}
	s.HandleKey(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	if s.tab != tabMCP {
		t.Errorf("after Right tab = %v, want MCP", s.tab)
	}
	s.HandleKey(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	if s.tab != tabSkills {
		t.Errorf("after Right tab = %v, want Skills", s.tab)
	}
	s.HandleKey(tcell.NewEventKey(tcell.KeyRight, 0, 0))
	if s.tab != tabGeneral {
		t.Errorf("after wrap tab = %v, want General", s.tab)
	}
	s.HandleKey(tcell.NewEventKey(tcell.KeyLeft, 0, 0))
	if s.tab != tabSkills {
		t.Errorf("after Left tab = %v, want Skills", s.tab)
	}
}

func TestSettingsEscCloses(t *testing.T) {
	s := New(Deps{})
	closed := false
	s.SetOnClose(func() { closed = true })
	s.HandleKey(tcell.NewEventKey(tcell.KeyEsc, 0, 0))
	if !closed {
		t.Error("Esc should fire onClose")
	}
}

func TestSettingsGeneralEnterOpensProviderSub(t *testing.T) {
	s := New(Deps{})
	s.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	if s.sub == nil {
		t.Fatal("Enter on Provider row should open a sub-modal")
	}
	s.HandleKey(tcell.NewEventKey(tcell.KeyEsc, 0, 0))
	if s.sub != nil {
		t.Error("Esc should close the sub-modal")
	}
}

func TestSettingsSubOpensModelPickerAndBack(t *testing.T) {
	s := New(Deps{})
	s.HandleKey(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	s.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	if s.sub == nil {
		t.Fatal("Enter on Model row should open the picker")
	}
	s.HandleKey(tcell.NewEventKey(tcell.KeyEsc, 0, 0))
	if s.sub != nil {
		t.Error("Esc should close the picker")
	}
}

func TestSettingsGeneralRowSelection(t *testing.T) {
	s := New(Deps{})
	g := s.general
	if g.index != 0 {
		t.Fatalf("initial index = %d", g.index)
	}
	g.HandleKey(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	if g.index != 1 {
		t.Errorf("after Down index = %d, want 1", g.index)
	}
	g.HandleKey(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	g.HandleKey(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	if g.index != 2 {
		t.Errorf("index should clamp at 2, got %d", g.index)
	}
}

func TestListModalHandleKey(t *testing.T) {
	back := false
	m := &listModal{title: "x", list: components.NewList("x"), onBack: func() { back = true }}
	m.HandleKey(tcell.NewEventKey(tcell.KeyEsc, 0, 0))
	if !back {
		t.Error("Esc on listModal should trigger onBack")
	}
}

func TestSettingsAllTabsDraw(t *testing.T) {
	styles.RegisterDefaults()
	styles.Set("dark")
	s := New(Deps{})
	for i := 0; i < len(tabNames); i++ {
		s.tab = tabKind(i)
		drawSettings(t, 80, 24, s)
		drawSettings(t, 110, 32, s)
	}
}

func TestSettingsSubModalsDraw(t *testing.T) {
	styles.RegisterDefaults()
	styles.Set("dark")
	s := New(Deps{})
	s.general.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	drawSettings(t, 80, 24, s)
	s.HandleKey(tcell.NewEventKey(tcell.KeyEsc, 0, 0))
	s.general.HandleKey(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	s.general.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	drawSettings(t, 80, 24, s)
	s.HandleKey(tcell.NewEventKey(tcell.KeyEsc, 0, 0))
	s.general.HandleKey(tcell.NewEventKey(tcell.KeyDown, 0, 0))
	s.general.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, 0))
	drawSettings(t, 80, 24, s)
}

func TestSettingsSetSelectedModel(t *testing.T) {
	s := New(Deps{})
	s.SetSelectedModel("provider-x", llm.Model{ID: "model-y", Name: "Model Y"})
	if got := s.modelDisplay(); got != "provider-x/Model Y" {
		t.Errorf("modelDisplay = %q, want provider-x/Model Y", got)
	}
}

func TestSettingsSelectedModelEmptyDisplay(t *testing.T) {
	s := New(Deps{})
	if got := s.modelDisplay(); got != "—" {
		t.Errorf("empty modelDisplay = %q, want em dash", got)
	}
}

func TestSettingsSetSelectedModelUsesID(t *testing.T) {
	s := New(Deps{})
	s.SetSelectedModel("p", llm.Model{ID: "m1"})
	if got := s.modelDisplay(); got != "p/m1" {
		t.Errorf("modelDisplay = %q, want p/m1", got)
	}
}
