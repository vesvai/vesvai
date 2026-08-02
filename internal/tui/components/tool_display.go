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

type ToolDisplay struct {
	Name     string
	Args     map[string]any
	Status   ToolStatus
	Result   string
	Error    error
	Duration int64
	Expanded bool
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

func (td *ToolDisplay) ToggleExpand() {
	td.Expanded = !td.Expanded
}

func (td *ToolDisplay) IsExpanded() bool {
	return td.Expanded
}

func (td *ToolDisplay) toolDisplayName() string {
	name := strings.ToUpper(td.Name)
	name = strings.ReplaceAll(name, "_", " ")
	return name
}

func (td *ToolDisplay) toolPath() string {
	if td.Args == nil {
		return ""
	}
	for _, v := range td.Args {
		val := fmt.Sprintf("%v", v)
		if len(val) > 0 {
			if len(val) > 50 {
				val = val[:47] + "..."
			}
			return val
		}
	}
	return ""
}

func (td *ToolDisplay) Height(screenWidth int) int {
	return 1
}

func (td *ToolDisplay) Draw(s tcell.Screen, y, screenWidth int) {
	startX := 2
	width := screenWidth - 4

	render.FillArea(s, startX, y, width, 1, tcell.StyleDefault.Background(theme.BgPrimary))

	statusStyle := tcell.StyleDefault.Foreground(theme.AccentGold).Background(theme.BgPrimary)
	statusGlyph := td.statusGlyph()
	s.SetContent(startX, y, []rune(statusGlyph)[0], nil, statusStyle)

	nameStyle := tcell.StyleDefault.Foreground(theme.TextSecondary).Background(theme.BgPrimary).Bold(true)
	name := td.toolDisplayName()
	render.DrawText(s, startX+2, y, name, nameStyle)

	path := td.toolPath()
	if path != "" {
		pathStyle := tcell.StyleDefault.Foreground(theme.TextDim).Background(theme.BgPrimary)
		pathX := startX + 2 + len(name) + 1
		render.DrawTextLimited(s, pathX, y, width-10, path, pathStyle)
	}
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
