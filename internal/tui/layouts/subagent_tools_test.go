package layouts

import (
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func buildSubagentWithTool(t *testing.T) (*MainLayout, *tui.Model, tcell.SimulationScreen) {
	t.Helper()
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	t.Cleanup(s.Fini)

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("main user message")
	model.Apply(tui.StreamEvent{Kind: tui.EventStart})
	model.Apply(tui.StreamEvent{
		Kind:     tui.EventSubagentStart,
		Subagent: &tui.SubagentInfo{Name: "planner", Prompt: "plan the work"},
	})
	model.Apply(tui.StreamEvent{
		Kind: tui.EventToolCall,
		ToolCall: &tui.ToolCallInfo{
			ToolName:   "bash",
			Args:       map[string]any{"command": "ls -la"},
			SubagentID: "planner",
		},
	})
	model.Apply(tui.StreamEvent{
		Kind: tui.EventToolResult,
		ToolResult: &tui.ToolResultInfo{
			ToolName:   "bash",
			Result:     "drwxrwxr-x  obuntu\n-rw-rw-r--  x",
			Duration:   200 * time.Millisecond,
			SubagentID: "planner",
		},
	})
	model.Apply(tui.StreamEvent{Kind: tui.EventSubagentChunk, SubagentID: "planner", Content: "the plan: inspect then edit"})
	model.Apply(tui.StreamEvent{
		Kind:           tui.EventSubagentDone,
		SubagentID:     "planner",
		SubagentResult: &tui.SubagentResult{Name: "planner", Duration: 900 * time.Millisecond},
	})
	return l, model, s
}

func openSubagentView(t *testing.T, l *MainLayout, s tcell.SimulationScreen) int {
	t.Helper()
	rows := renderFrame(t, l, s)
	for i, r := range rows {
		if strings.Contains(r, "planner") && strings.Contains(r, "✔") {
			click(l, 10, i)
			return i
		}
	}
	t.Fatalf("subagent line not found:\n%s", frameText(rows))
	return -1
}

func buildSubagentWithParts(t *testing.T) (*MainLayout, *tui.Model, tcell.SimulationScreen) {
	t.Helper()
	s := tcell.NewSimulationScreen("UTF-8")
	s.Init()
	t.Cleanup(s.Fini)

	model := tui.NewModel("demo")
	l := NewMainLayout(model, tui.DefaultDark())
	model.Conv.AddUser("main user message")
	model.Apply(tui.StreamEvent{Kind: tui.EventStart})
	model.Apply(tui.StreamEvent{
		Kind:     tui.EventSubagentStart,
		Subagent: &tui.SubagentInfo{Name: "planner", Prompt: "plan the work"},
	})
	model.Apply(tui.StreamEvent{Kind: tui.EventSubagentChunk, SubagentID: "planner", Content: "thinking step one", SubagentReasoning: true})
	model.Apply(tui.StreamEvent{
		Kind: tui.EventToolCall,
		ToolCall: &tui.ToolCallInfo{
			ToolName:   "bash",
			Args:       map[string]any{"command": "ls -la"},
			SubagentID: "planner",
		},
	})
	model.Apply(tui.StreamEvent{
		Kind: tui.EventToolResult,
		ToolResult: &tui.ToolResultInfo{
			ToolName:   "bash",
			Result:     "drwxrwxr-x  obuntu\n-rw-rw-r--  x",
			Duration:   200 * time.Millisecond,
			SubagentID: "planner",
		},
	})
	model.Apply(tui.StreamEvent{Kind: tui.EventSubagentChunk, SubagentID: "planner", Content: "the plan: inspect then edit"})
	model.Apply(tui.StreamEvent{
		Kind:           tui.EventSubagentDone,
		SubagentID:     "planner",
		SubagentResult: &tui.SubagentResult{Name: "planner", Duration: 900 * time.Millisecond},
	})
	return l, model, s
}

func TestSubagentViewOrderedParts(t *testing.T) {
	l, _, s := buildSubagentWithParts(t)
	openSubagentView(t, l, s)

	rows := renderFrame(t, l, s)
	joined := frameText(rows)

	thinkRow, toolRow, contentRow := -1, -1, -1
	for i, r := range rows {
		if strings.Contains(r, "Thinking") {
			thinkRow = i
		}
		if strings.Contains(r, "⚙ bash") {
			toolRow = i
		}
		if strings.Contains(r, "the plan: inspect then edit") {
			contentRow = i
		}
	}
	if thinkRow < 0 {
		t.Fatalf("thinking header missing from transcript:\n%s", joined)
	}
	if toolRow < 0 {
		t.Fatalf("tool card missing from transcript:\n%s", joined)
	}
	if contentRow < 0 {
		t.Fatalf("content missing from transcript:\n%s", joined)
	}
	if !(thinkRow < toolRow && toolRow < contentRow) {
		t.Fatalf("parts out of order (think=%d tool=%d content=%d):\n%s", thinkRow, toolRow, contentRow, joined)
	}
	if strings.Contains(joined, "thinking step one") {
		t.Fatalf("thinking body must be collapsed by default:\n%s", joined)
	}
	if strings.Contains(joined, "drwxrwxr-x") {
		t.Fatalf("collapsed tool result leaked into transcript:\n%s", joined)
	}
}

func TestSubagentViewThinkingExpands(t *testing.T) {
	l, _, s := buildSubagentWithParts(t)
	openSubagentView(t, l, s)

	click(l, 10, 4)

	rows := renderFrame(t, l, s)
	joined := frameText(rows)
	if !strings.Contains(joined, "thinking step one") {
		t.Fatalf("thinking body missing after header click:\n%s", joined)
	}
	click(l, 10, 4)
	rows = renderFrame(t, l, s)
	if strings.Contains(frameText(rows), "thinking step one") {
		t.Fatalf("thinking body still visible after collapse:\n%s", frameText(rows))
	}
}

func TestSubagentViewShowsToolCards(t *testing.T) {
	l, _, s := buildSubagentWithTool(t)
	openSubagentView(t, l, s)

	rows := renderFrame(t, l, s)
	joined := frameText(rows)
	if !strings.Contains(joined, "bash") || !strings.Contains(joined, `"ls -la"`) {
		t.Fatalf("tool card missing from subagent transcript:\n%s", joined)
	}
	if strings.Contains(joined, "drwxrwxr-x") {
		t.Fatalf("collapsed tool result leaked into transcript:\n%s", joined)
	}
}

func TestSubagentViewToolClickExpands(t *testing.T) {
	l, _, s := buildSubagentWithTool(t)
	openSubagentView(t, l, s)

	click(l, 10, 4)

	rows := renderFrame(t, l, s)
	joined := frameText(rows)
	if !strings.Contains(joined, "drwxrwxr-x") {
		t.Fatalf("expanded tool result missing:\n%s", joined)
	}
	if !strings.Contains(joined, "the plan: inspect then edit") {
		t.Fatalf("transcript content missing after expand:\n%s", joined)
	}

	click(l, 10, 4)
	rows = renderFrame(t, l, s)
	if strings.Contains(frameText(rows), "drwxrwxr-x") {
		t.Fatalf("collapsed tool result still visible:\n%s", frameText(rows))
	}
}
