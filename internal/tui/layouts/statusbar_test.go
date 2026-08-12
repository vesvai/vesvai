package layouts

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func statusbarRow(s tcell.SimulationScreen, l *MainLayout) string {
	rows := renderRaw(s, l)
	for _, r := range rows {
		if strings.Contains(r, "idle") || strings.Contains(r, "running") ||
			strings.Contains(r, "· step") || strings.Contains(r, "tok") {
			return r
		}
	}
	return strings.Join(rows, "\n")
}

func TestStatusbarShowsModelProviderEffort(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("Opus 4.5")
	model.Provider = "Anthropic"
	model.Effort = "max"
	model.ContextWindow = 200_000
	l := NewMainLayout(model, tui.DefaultDark())
	renderFrame(t, l, s)

	row := statusbarRow(s, l)
	if !strings.Contains(row, "Opus 4.5") {
		t.Fatalf("model name missing: %q", row)
	}
	if !strings.Contains(row, "Anthropic") {
		t.Fatalf("provider missing: %q", row)
	}
	if !strings.Contains(row, "max") {
		t.Fatalf("effort missing: %q", row)
	}

	cells, _, _ := s.GetContents()
	w, _ := s.Size()
	y := -1
	for i := 0; i < 25; i++ {
		if strings.Contains(stringOfRow(cells, w, i), "max") {
			y = i
			break
		}
	}
	if y < 0 {
		t.Fatal("statusbar row not found")
	}
	rowStr := stringOfRow(cells, w, y)
	x := strings.Index(rowStr, "max")
	fg, _, _ := cells[y*w+x].Style.Decompose()
	if fg != tui.DefaultDark().Error {
		t.Fatalf("effort fg = %v, want error color %v", fg, tui.DefaultDark().Error)
	}
}

func TestStatusbarShowsTokenTotalWithPercentage(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("deepseek-v4")
	model.ContextWindow = 1_000_000
	model.Usage = tui.Usage{PromptTokens: 300_000, CompletionTokens: 109_200, TotalTokens: 409_200}
	l := NewMainLayout(model, tui.DefaultDark())
	renderFrame(t, l, s)

	row := statusbarRow(s, l)
	if !strings.Contains(row, "409.2K") {
		t.Fatalf("token total missing: %q", row)
	}
	if !strings.Contains(row, "41%") {
		t.Fatalf("context percentage missing: %q", row)
	}
}

func TestStatusbarBackgroundMatchesScreenBackground(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	renderFrame(t, l, s)

	cells, _, _ := s.GetContents()
	w, h := s.Size()

	bg := tui.DefaultDark().Background
	for x := 0; x < w; x++ {
		_, cbg, _ := cells[(h-1)*w+x].Style.Decompose()
		if cbg != bg {
			t.Fatalf("statusbar cell %d bg = %v, want background %v", x, cbg, bg)
		}
	}
	_, taBg, _ := cells[(h-5)*w+10].Style.Decompose()
	if taBg == bg {
		t.Fatal("statusbar and textarea backgrounds must differ")
	}
}

func stringOfRow(cells []tcell.SimCell, w, y int) string {
	var sb strings.Builder
	for x := 0; x < w; x++ {
		c := cells[y*w+x]
		if c.Runes != nil && len(c.Runes) > 0 {
			sb.WriteRune(c.Runes[0])
		} else {
			sb.WriteByte(' ')
		}
	}
	return sb.String()
}
