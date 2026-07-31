package components

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type ErrorOverlay struct {
	message     string
	visible     bool
	showTime    time.Time
	autoDismiss time.Duration
}

func NewErrorOverlay() *ErrorOverlay {
	return &ErrorOverlay{
		autoDismiss: 5 * time.Second,
	}
}

func (eo *ErrorOverlay) Show(message string) {
	eo.message = message
	eo.visible = true
	eo.showTime = time.Now()
}

func (eo *ErrorOverlay) Hide() {
	eo.visible = false
	eo.message = ""
}

func (eo *ErrorOverlay) IsVisible() bool {
	return eo.visible
}

func (eo *ErrorOverlay) ShouldAutoDismiss() bool {
	if !eo.visible {
		return false
	}
	return time.Since(eo.showTime) > eo.autoDismiss
}

func (eo *ErrorOverlay) Message() string {
	return eo.message
}

func (eo *ErrorOverlay) Draw(s tcell.Screen, areaY, areaWidth, areaHeight int) {
	if !eo.visible || eo.message == "" {
		return
	}

	if eo.ShouldAutoDismiss() {
		eo.Hide()
		return
	}

	maxWidth := areaWidth - 8
	if maxWidth < 20 {
		maxWidth = 20
	}
	msg := eo.message
	if len(msg) > maxWidth {
		truncateAt := maxWidth - 1
		for truncateAt > maxWidth/2 && msg[truncateAt] != ' ' {
			truncateAt--
		}
		if truncateAt <= maxWidth/2 {
			truncateAt = maxWidth - 1
		}
		msg = msg[:truncateAt] + "…"
	}

	height := 3
	width := len(msg) + 8
	if width > areaWidth-2 {
		width = areaWidth - 2
	}
	startX := render.CenterX(width, areaWidth)
	startY := areaY + 1

	bgStyle := tcell.StyleDefault.
		Foreground(theme.TextPrimary).
		Background(tcell.NewRGBColor(0x3A, 0x12, 0x12))
	render.FillArea(s, startX, startY, width, height, bgStyle)

	borderStyle := tcell.StyleDefault.
		Foreground(theme.AccentRed).
		Background(tcell.NewRGBColor(0x3A, 0x12, 0x12))
	for i := 0; i < width; i++ {
		s.SetContent(startX+i, startY, '─', nil, borderStyle)
		s.SetContent(startX+i, startY+height-1, '─', nil, borderStyle)
	}
	for i := 0; i < height; i++ {
		s.SetContent(startX, startY+i, '│', nil, borderStyle)
		s.SetContent(startX+width-1, startY+i, '│', nil, borderStyle)
	}
	s.SetContent(startX, startY, '╭', nil, borderStyle)
	s.SetContent(startX+width-1, startY, '╮', nil, borderStyle)
	s.SetContent(startX, startY+height-1, '╰', nil, borderStyle)
	s.SetContent(startX+width-1, startY+height-1, '╯', nil, borderStyle)

	iconStyle := tcell.StyleDefault.
		Foreground(theme.AccentRed).
		Background(tcell.NewRGBColor(0x3A, 0x12, 0x12)).
		Bold(true)
	msgStyle := tcell.StyleDefault.
		Foreground(theme.TextPrimary).
		Background(tcell.NewRGBColor(0x3A, 0x12, 0x12))

	render.DrawText(s, startX+2, startY+1, "×", iconStyle)
	render.DrawText(s, startX+4, startY+1, msg, msgStyle)

	hint := " (Esc to dismiss)"
	hintX := startX + 4 + len(msg)
	if hintX+len(hint) < startX+width {
		hintStyle := tcell.StyleDefault.
			Foreground(theme.TextDim).
			Background(tcell.NewRGBColor(0x3A, 0x12, 0x12))
		render.DrawText(s, hintX, startY+1, hint, hintStyle)
	}
}

func (eo *ErrorOverlay) HandleEvent(ev tcell.Event) bool {
	if !eo.visible {
		return false
	}
	if ke, ok := ev.(*tcell.EventKey); ok {
		if ke.Key() == tcell.KeyEscape {
			eo.Hide()
			return true
		}
	}
	return false
}
