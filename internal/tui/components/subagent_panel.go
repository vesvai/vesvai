package components

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type SubagentEntry struct {
	ID        string
	Name      string
	Prompt    string
	Status    string
	Content   string
	Reasoning string
	ToolCount int
}

type SubagentPanel struct {
	entries   []SubagentEntry
	visible   bool
	selected  int
	scrollOff int
}

func NewSubagentPanel() *SubagentPanel {
	return &SubagentPanel{}
}

func (sp *SubagentPanel) IsVisible() bool { return sp.visible }
func (sp *SubagentPanel) Show()           { sp.visible = true }
func (sp *SubagentPanel) Hide()           { sp.visible = false }
func (sp *SubagentPanel) Toggle()         { sp.visible = !sp.visible }
func (sp *SubagentPanel) SetEntries(e []SubagentEntry) {
	sp.entries = e
	if sp.selected >= len(sp.entries) {
		sp.selected = len(sp.entries) - 1
	}
	if sp.selected < 0 {
		sp.selected = 0
	}
}
func (sp *SubagentPanel) Entries() []SubagentEntry { return sp.entries }

func (sp *SubagentPanel) HandleEvent(ev tcell.Event) bool {
	if !sp.visible {
		return false
	}
	if ke, ok := ev.(*tcell.EventKey); ok {
		switch ke.Key() {
		case tcell.KeyUp:
			if sp.selected > 0 {
				sp.selected--
			}
			return true
		case tcell.KeyDown:
			if sp.selected < len(sp.entries)-1 {
				sp.selected++
			}
			return true
		case tcell.KeyEscape:
			sp.Hide()
			return true
		}
	}
	return false
}

func statusGlyph(status string) (string, tcell.Color) {
	switch status {
	case "running":
		return "◍", theme.AccentCyan
	case "completed":
		return "●", theme.AccentGreen
	case "failed":
		return "×", theme.AccentRed
	case "cancelled":
		return "⊘", theme.TextDim
	default:
		return "○", theme.TextDim
	}
}

func nameIcon(name string) string {
	switch name {
	case "explorer":
		return "✦"
	case "planner":
		return "◇"
	case "orchestrator":
		return "◆"
	default:
		return "▣"
	}
}

func (sp *SubagentPanel) Height(width int) int {
	if len(sp.entries) == 0 {
		return 3
	}
	h := 3
	for range sp.entries {
		h += 2
	}
	return 20
}

func (sp *SubagentPanel) Draw(s tcell.Screen, y, width int) {
	panelWidth := 32
	if panelWidth > width-4 {
		panelWidth = width - 4
	}
	startX := width - panelWidth - 1
	panelHeight := sp.Height(width)

	bgStyle := tcell.StyleDefault.Background(theme.BgOverlay)
	render.FillArea(s, startX, y, panelWidth, panelHeight, bgStyle)

	headerStyle := tcell.StyleDefault.
		Foreground(theme.AccentPurple).
		Background(theme.BgOverlay).
		Bold(true)
	render.DrawText(s, startX+1, y, "▣ Subagents", headerStyle)

	countStr := fmt.Sprintf("(%d)", len(sp.entries))
	countStyle := tcell.StyleDefault.
		Foreground(theme.TextDim).
		Background(theme.BgOverlay)
	render.DrawText(s, startX+panelWidth-len(countStr)-1, y, countStr, countStyle)

	sepStyle := tcell.StyleDefault.
		Foreground(theme.BorderMuted).
		Background(theme.BgOverlay)
	for i := 0; i < panelWidth; i++ {
		s.SetContent(startX+i, y+1, '─', nil, sepStyle)
	}

	if len(sp.entries) == 0 {
		emptyStyle := tcell.StyleDefault.
			Foreground(theme.TextDim).
			Background(theme.BgOverlay)
		render.DrawText(s, startX+2, y+2, "No subagents active", emptyStyle)
		return
	}

	entryY := y + 2
	for i, entry := range sp.entries {
		glyph, glyphColor := statusGlyph(entry.Status)
		icon := nameIcon(entry.Name)

		entryStyle := tcell.StyleDefault.Foreground(theme.TextPrimary).Background(theme.BgOverlay)
		if i == sp.selected {
			entryStyle = entryStyle.Background(theme.BgTertiary)
			highlightStyle := tcell.StyleDefault.Background(theme.BgTertiary)
			render.FillArea(s, startX, entryY, panelWidth, 2, highlightStyle)
		}

		nameStyle := entryStyle.Bold(true)
		render.DrawText(s, startX+1, entryY, icon+" "+entry.Name, nameStyle)

		glyphStyle := tcell.StyleDefault.Foreground(glyphColor).Background(theme.BgOverlay)
		if i == sp.selected {
			glyphStyle = glyphStyle.Background(theme.BgTertiary)
		}
		render.DrawText(s, startX+panelWidth-3, entryY, glyph, glyphStyle)

		prompt := entry.Prompt
		maxLen := panelWidth - 6
		if len(prompt) > maxLen {
			prompt = prompt[:maxLen-1] + "…"
		}
		promptStyle := tcell.StyleDefault.
			Foreground(theme.TextDim).
			Background(theme.BgOverlay)
		if i == sp.selected {
			promptStyle = promptStyle.Background(theme.BgTertiary)
		}
		render.DrawText(s, startX+3, entryY+1, prompt, promptStyle)

		entryY += 2
	}

	if sp.selected >= 0 && sp.selected < len(sp.entries) {
		entry := sp.entries[sp.selected]
		detailY := y + 2 + len(sp.entries)*2 + 1

		for i := 0; i < panelWidth; i++ {
			s.SetContent(startX+i, detailY, '─', nil, sepStyle)
		}
		detailY++

		contentLines := strings.Split(entry.Content, "\n")
		maxLines := panelHeight - (detailY - y) - 1
		for i, line := range contentLines {
			if i >= maxLines || i >= 5 {
				break
			}
			truncated := line
			if len(truncated) > panelWidth-4 {
				truncated = truncated[:panelWidth-5] + "…"
			}
			contentStyle := tcell.StyleDefault.
				Foreground(theme.TextSecondary).
				Background(theme.BgOverlay)
			render.DrawText(s, startX+2, detailY+i, truncated, contentStyle)
		}

		if entry.ToolCount > 0 {
			tcStr := fmt.Sprintf("tools: %d", entry.ToolCount)
			tcStyle := tcell.StyleDefault.
				Foreground(theme.AccentAmber).
				Background(theme.BgOverlay)
			render.DrawText(s, startX+2, detailY+min(len(contentLines), 5)+1, tcStr, tcStyle)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
