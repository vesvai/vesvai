package layouts

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func collectThinkingColors(s tcell.SimulationScreen) []tcell.Color {
	cells, w, _ := s.GetContents()
	var out []tcell.Color
	for i := 0; i+8 < len(cells); i++ {
		if cells[i].Runes == nil || len(cells[i].Runes) == 0 || cells[i].Runes[0] != 'T' {
			continue
		}
		text := ""
		for j := 0; j < 8; j++ {
			if cells[i+j].Runes != nil && len(cells[i+j].Runes) > 0 {
				text += string(cells[i+j].Runes[0])
			}
		}
		if text != "Thinking" {
			continue
		}
		fg, _, _ := cells[i].Style.Decompose()
		out = append(out, fg)
		_ = w
	}
	return out
}

func TestOnlyCurrentThinkingGlows(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("do it")
	model.Apply(tui.StreamEvent{Kind: tui.EventStart})
	model.Apply(tui.StreamEvent{Kind: tui.EventReasoning, Reasoning: "first thoughts"})
	model.Apply(tui.StreamEvent{Kind: tui.EventToolCall, ToolCall: &tui.ToolCallInfo{ToolName: "read"}})
	model.Apply(tui.StreamEvent{Kind: tui.EventToolResult, ToolResult: &tui.ToolResultInfo{ToolName: "read", Result: "code"}})
	model.Apply(tui.StreamEvent{Kind: tui.EventReasoning, Reasoning: "second thoughts"})

	renderFrame(t, l, s)
	colors := collectThinkingColors(s)

	if len(colors) != 2 {
		t.Fatalf("found %d Thinking labels, want 2:\n%s", len(colors), frameText(renderFrame(t, l, s)))
	}
	pal := tui.DefaultDark()
	if colors[0] != pal.TextDim {
		t.Fatalf("finished thinking block color = %v, want static TextDim %v", colors[0], pal.TextDim)
	}
	if colors[1] == pal.TextDim {
		t.Fatal("active thinking block must glow (not TextDim)")
	}

	model.Apply(tui.StreamEvent{Kind: tui.EventDone})
	renderFrame(t, l, s)
	colors = collectThinkingColors(s)
	if len(colors) != 2 {
		t.Fatalf("found %d Thinking labels after done, want 2", len(colors))
	}
	if colors[0] != pal.TextDim || colors[1] != pal.TextDim {
		t.Fatalf("done message thinking colors = %v, want static TextDim", colors)
	}
}

func TestThinkingLabelExists(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	defer s.Fini()

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("do it")
	model.Apply(tui.StreamEvent{Kind: tui.EventStart})
	model.Apply(tui.StreamEvent{Kind: tui.EventReasoning, Reasoning: "hmm"})

	renderFrame(t, l, s)
	joined := frameText(renderFrame(t, l, s))
	if !strings.Contains(joined, "Thinking") {
		t.Fatalf("thinking header missing:\n%s", joined)
	}
}
