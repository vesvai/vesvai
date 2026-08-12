package components

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

type Block struct {
	ID    string
	Kind  string // "thinking" | "tool" | "subagent"
	Start int
	End   int // exclusive
}

type MessageView struct {
	md       *MarkdownRenderer
	thinking *ThinkingView
}

func NewMessageView() *MessageView {
	return &MessageView{
		md:       NewMarkdownRenderer(),
		thinking: NewThinkingView(),
	}
}

func (v *MessageView) Render(m *tui.Message, width int, pal *tui.Palette) bool {
	changed := false
	if m.Role == tui.RoleAssistant {
		for i := range m.Parts {
			p := &m.Parts[i]
			switch p.Kind {
			case tui.PartThinking:
				if p.Thinking != "" && p.ThinkingDirty() {
					p.SetRenderedThinking(tui.WrapText(p.Thinking, pal.Style(pal.Reasoning, pal.Background), width))
					changed = true
				}
			case tui.PartContent:
				if p.ContentDirty() {
					p.SetRenderedContent(v.md.Render(p.Content, width, pal))
					changed = true
				}
			}
		}
	}
	if m.Role == tui.RoleUser && m.ContentDirty() {
		inner := width - 4
		if inner < 4 {
			inner = 4
		}
		m.SetRenderedContent(tui.WrapText(m.Content, pal.Style(pal.Foreground, pal.UserBg), inner))
		changed = true
	}
	return changed
}

func (v *MessageView) Lines(m *tui.Message, width int, pal *tui.Palette, now time.Duration) ([]tui.Line, []Block) {
	var lines []tui.Line
	var blocks []Block

	if m.Role == tui.RoleUser {
		lines = append(lines, userCardLines(m, width, pal)...)
		return lines, blocks
	}

	m.EnsureParts()
	active := m.Status == tui.StatusRunning
	th, tj, si := 0, 0, 0

	for i := range m.Parts {
		p := &m.Parts[i]
		switch p.Kind {
		case tui.PartThinking:
			start := len(lines)
			lines = append(lines, v.thinking.Lines(p, width, pal, active, now)...)
			blocks = append(blocks, Block{ID: tui.ThinkingPartID(m, th), Kind: "thinking", Start: start, End: len(lines)})
			th++

		case tui.PartContent:
			lines = append(lines, p.RenderedContent()...)

		case tui.PartTool:
			start := len(lines)
			lines = append(lines, ToolLines(p.Tool, width, pal, now)...)
			blocks = append(blocks, Block{ID: tui.ToolPartID(m, tj), Kind: "tool", Start: start, End: len(lines)})
			tj++

		case tui.PartSubagent:
			start := len(lines)
			lines = append(lines, SubagentStatusLine(p.Subagent, pal, now))
			blocks = append(blocks, Block{ID: tui.SubagentBlockID(m, si), Kind: "subagent", Start: start, End: len(lines)})
			si++
		}
	}

	if m.Status == tui.StatusError && m.Error != nil {
		lines = append(lines, tui.LineFromSegments([]tui.Segment{
			{Text: "✖ " + m.Error.Error(), Style: pal.Style(pal.Error, pal.Background)},
		}, width))
	}

	return lines, blocks
}

func (v *MessageView) SubagentTranscript(sa *tui.Subagent, width int, pal *tui.Palette, now time.Duration) ([]tui.Line, []Block) {
	var lines []tui.Line
	var blocks []Block

	label := "✦ " + sa.Name
	style := pal.Style(pal.Subagent, pal.Background).Bold(true)
	labelCells := make(tui.Line, 0, len(label))
	for _, r := range label {
		labelCells = append(labelCells, tui.Cell{R: r, S: style})
	}
	lines = append(lines, labelCells)

	if sa.Prompt != "" {
		lines = append(lines, tui.LineFromSegments([]tui.Segment{
			{Text: "→ " + sa.Prompt, Style: pal.Style(pal.TextDim, pal.Background)},
		}, width))
	}
	lines = append(lines, SubagentStatusLine(sa, pal, now))
	lines = append(lines, nil)

	if len(sa.Parts) == 0 && (sa.Thinking != "" || sa.Content != "" || len(sa.Tools) > 0) {
		sa.RebuildParts()
	}
	active := sa.Status == tui.StatusRunning
	th, tj := 0, 0
	for i := range sa.Parts {
		p := &sa.Parts[i]
		switch p.Kind {
		case tui.PartThinking:
			start := len(lines)
			lines = append(lines, v.thinking.Lines(p, width, pal, active, now)...)
			blocks = append(blocks, Block{
				ID:    fmt.Sprintf("subthink:%d", th),
				Kind:  "subthink",
				Start: start,
				End:   len(lines),
			})
			th++

		case tui.PartContent:
			if p.ContentDirty() {
				p.SetRenderedContent(v.md.Render(p.Content, width, pal))
			}
			lines = append(lines, p.RenderedContent()...)

		case tui.PartTool:
			start := len(lines)
			lines = append(lines, ToolLines(p.Tool, width, pal, now)...)
			blocks = append(blocks, Block{
				ID:    fmt.Sprintf("subtool:%d", tj),
				Kind:  "tool",
				Start: start,
				End:   len(lines),
			})
			tj++
		}
	}

	return lines, blocks
}

func userCardLines(m *tui.Message, width int, pal *tui.Palette) []tui.Line {
	if width < 8 {
		return m.RenderedContent()
	}
	border := pal.Style(pal.Border, pal.UserBg)

	var lines []tui.Line
	lines = append(lines, cardEdge('┌', '┐', width, border))

	content := m.RenderedContent()
	if len(content) == 0 {
		content = []tui.Line{{}}
	}
	for _, ln := range content {
		lines = append(lines, cardRow(ln, width, border, pal))
	}
	for _, a := range m.Attachments {
		lines = append(lines, cardRow(attachmentChipLine(a, width-4, pal), width, border, pal))
	}

	lines = append(lines, cardEdge('└', '┘', width, border))
	return lines
}

func cardEdge(left, right rune, width int, border tcell.Style) tui.Line {
	line := tui.Line{{R: left, S: border}}
	for len(line) < width-1 {
		line = append(line, tui.Cell{R: '─', S: border})
	}
	line = append(line, tui.Cell{R: right, S: border})
	return line
}

func cardRow(ln tui.Line, width int, border tcell.Style, pal *tui.Palette) tui.Line {
	bg := pal.Style(pal.Foreground, pal.UserBg)
	row := tui.Line{{R: '│', S: border}, {R: ' ', S: bg}}
	row = append(row, ln...)
	for len(row) < width-1 {
		row = append(row, tui.Cell{R: ' ', S: bg})
	}
	row = append(row, tui.Cell{R: '│', S: border})
	return row
}

func attachmentChipLine(a *tui.Attachment, width int, pal *tui.Palette) tui.Line {
	var icon rune
	switch a.Kind {
	case "image":
		icon = '🖼'
	case "video":
		icon = '🎬'
	case "pdf":
		icon = '📄'
	default:
		icon = '📎'
	}
	bg := pal.Style(pal.Foreground, pal.UserBg)
	nameStyle := pal.Style(pal.TextDim, pal.UserBg)
	line := tui.Line{
		{R: ' ', S: bg},
		{R: icon, S: bg},
		{R: ' ', S: bg},
	}
	name := a.Name
	for line.Width()+tui.DisplayWidth(name)+tui.DisplayWidth(tui.FormatSize(a.Size))+4 > width {
		if len(name) <= 1 {
			break
		}
		name = name[:len(name)-1]
	}
	if a.Name != name {
		name += "…"
	}
	for _, r := range name {
		line = append(line, tui.Cell{R: r, S: nameStyle})
	}
	size := pal.Style(pal.Muted, pal.UserBg)
	line = append(line,
		tui.Cell{R: ' ', S: nameStyle},
		tui.Cell{R: '·', S: size},
		tui.Cell{R: ' ', S: size},
	)
	for _, r := range tui.FormatSize(a.Size) {
		line = append(line, tui.Cell{R: r, S: size})
	}
	return line
}
