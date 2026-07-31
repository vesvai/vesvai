package components

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type ToolStatus int

const (
	ToolPending ToolStatus = iota
	ToolRunning
	ToolComplete
	ToolFailed
)

const maxPreviewLines = 3

type ToolDisplay struct {
	Name     string
	Args     map[string]any
	Status   ToolStatus
	Result   string
	Error    error
	Duration int64
}

func NewToolDisplay(name string, args map[string]any) *ToolDisplay {
	return &ToolDisplay{
		Name:   name,
		Args:   args,
		Status: ToolPending,
	}
}

func (td *ToolDisplay) SetRunning() {
	td.Status = ToolRunning
}

func (td *ToolDisplay) SetComplete(result string, duration int64) {
	td.Status = ToolComplete
	td.Result = result
	td.Duration = duration
}

func (td *ToolDisplay) SetFailed(err error, duration int64) {
	td.Status = ToolFailed
	td.Error = err
	td.Duration = duration
}

func (td *ToolDisplay) statusGlyph() string {
	switch td.Status {
	case ToolRunning:
		return "◌"
	case ToolComplete:
		return "●"
	case ToolFailed:
		return "×"
	default:
		return "○"
	}
}

func (td *ToolDisplay) cardWidth(screenWidth int) (startX, width int) {
	startX = 2
	width = screenWidth - 4
	if width < 20 {
		width = 20
	}
	return startX, width
}

func (td *ToolDisplay) outputText() string {
	if td.Status == ToolFailed && td.Error != nil {
		return td.Error.Error()
	}
	return td.Result
}

func (td *ToolDisplay) previewLines(screenWidth int) []string {
	text := td.outputText()
	if text == "" {
		return nil
	}
	_, width := td.cardWidth(screenWidth)
	contentWidth := width - 4

	text = strings.ReplaceAll(text, "\r", "")
	rawLines := strings.Split(text, "\n")

	var out []string
	for _, ln := range rawLines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		for _, wl := range render.WrapText(ln, contentWidth) {
			out = append(out, wl)
			if len(out) >= maxPreviewLines {
				out = append(out, "…")
				return out
			}
		}
	}
	return out
}

func (td *ToolDisplay) Height(screenWidth int) int {
	h := 2
	h += len(td.previewLines(screenWidth))
	return h + 1
}

func (td *ToolDisplay) FormatArgs() string {
	if td.Args == nil {
		return ""
	}
	parts := make([]string, 0, len(td.Args))
	for k, v := range td.Args {
		val := fmt.Sprintf("%v", v)
		if len(val) > 40 {
			val = val[:40] + "…"
		}
		parts = append(parts, k+"="+val)
	}
	return strings.Join(parts, "  ")
}

func (td *ToolDisplay) Draw(s tcell.Screen, y, screenWidth int) {
	startX, width := td.cardWidth(screenWidth)
	bodyLines := td.previewLines(screenWidth)
	bodyHeight := 1 + len(bodyLines)

	headerStyle := tcell.StyleDefault.Background(theme.BgTertiary)
	bodyStyle := tcell.StyleDefault.Background(theme.BgSecondary)

	render.FillArea(s, startX, y, width, 1, headerStyle)

	dot := td.statusGlyph()
	dotStyle := tcell.StyleDefault.Foreground(theme.TextSecondary).Background(theme.BgTertiary)
	s.SetContent(startX+2, y, []rune(dot)[0], nil, dotStyle)

	nameStyle := tcell.StyleDefault.
		Foreground(theme.TextPrimary).
		Background(theme.BgTertiary).
		Bold(true)
	render.DrawText(s, startX+4, y, td.Name, nameStyle)

	rightBits := []string{}
	if td.Status == ToolComplete && td.Duration > 0 {
		rightBits = append(rightBits, fmt.Sprintf("%dms", td.Duration))
	}
	if td.Status == ToolFailed {
		rightBits = append(rightBits, "failed")
	}
	right := strings.Join(rightBits, "  ")
	if right != "" {
		rightStyle := tcell.StyleDefault.
			Foreground(theme.TextDim).
			Background(theme.BgTertiary)
		x := startX + width - len(right) - 2
		render.DrawText(s, x, y, right, rightStyle)
	}

	render.FillArea(s, startX, y+1, width, bodyHeight, bodyStyle)

	argsStr := td.FormatArgs()
	if argsStr != "" {
		argsStyle := tcell.StyleDefault.
			Foreground(theme.TextDim).
			Background(theme.BgSecondary)
		render.DrawTextLimited(s, startX+2, y+1, width-4, argsStr, argsStyle)
	}

	lineStyle := tcell.StyleDefault.
		Foreground(theme.TextSecondary).
		Background(theme.BgSecondary)
	for i, line := range bodyLines {
		render.DrawTextLimited(s, startX+2, y+2+i, width-4, line, lineStyle)
	}
}
