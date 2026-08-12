package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

type ConfirmModal struct {
	*Base
	title   string
	message string

	OnConfirm func()
	OnCancel  func()
}

func NewConfirmModal(title, message string) *ConfirmModal {
	c := &ConfirmModal{
		Base:    NewBase("confirm"),
		title:   title,
		message: message,
	}
	c.SetDraw(c.draw)
	return c
}

func (c *ConfirmModal) DesiredHeight() int {
	return 6
}

func (c *ConfirmModal) draw(s tcell.Screen, pal *tui.Palette) {
	rect := c.Bounds()
	if rect.Dx() < 12 || rect.Dy() < 5 {
		return
	}
	inner := drawModalBox(s, rect, c.title, pal)
	ix0, iy0 := inner.Min.X, inner.Min.Y
	iw := inner.Dx()

	textStyle := pal.Style(pal.Foreground, pal.Surface)
	line := tui.Line{{R: ' ', S: textStyle}}
	for _, r := range c.message {
		line = append(line, tui.Cell{R: r, S: textStyle})
	}
	if len(line) > iw {
		line = line[:iw]
	}
	for len(line) < iw {
		line = append(line, tui.Cell{R: ' ', S: textStyle})
	}
	tui.DrawLine(s, ix0, iy0+1, line)

	hintStyle := pal.Style(pal.Muted, pal.Surface)
	hint := " [y] yes · [n] no · [esc] cancel"
	hintLine := tui.Line{{R: ' ', S: hintStyle}}
	for _, r := range hint {
		hintLine = append(hintLine, tui.Cell{R: r, S: hintStyle})
	}
	if len(hintLine) > iw {
		hintLine = hintLine[:iw]
	}
	tui.DrawLine(s, ix0, iy0+inner.Dy()-1, hintLine)
}

func (c *ConfirmModal) HandleEvent(ev tcell.Event) bool {
	e, ok := ev.(*tcell.EventKey)
	if !ok {
		return false
	}
	switch e.Key() {
	case tcell.KeyEsc, tcell.KeyRune:
		if e.Key() == tcell.KeyRune {
			switch e.Rune() {
			case 'y', 'Y':
				if c.OnConfirm != nil {
					c.OnConfirm()
				}
				return true
			case 'n', 'N':
				if c.OnCancel != nil {
					c.OnCancel()
				}
				return true
			}
		}
		if c.OnCancel != nil {
			c.OnCancel()
		}
		return true
	}
	return false
}
