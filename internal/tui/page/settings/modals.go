package settings

import (
	"github.com/gdamore/tcell/v2"

	"github.com/vesvai/vesvai/internal/tui/components"
	"github.com/vesvai/vesvai/internal/tui/layout"
)

type listModal struct {
	title  string
	list   *components.List
	onBack func()
}

func (m *listModal) HandleKey(ev *tcell.EventKey) bool {
	if ev.Key() == tcell.KeyEsc {
		if m.onBack != nil {
			m.onBack()
		}
		return true
	}
	return m.list.HandleKey(ev)
}

func (m *listModal) Draw(s tcell.Screen, bounds layout.Region, focused bool) {
	w, h := bounds.Width-8, bounds.Height-6
	if w > 64 {
		w = 64
	}
	inner := components.DrawCenteredBox(s, bounds, w, h, m.title)
	listRegion := layout.Region{Left: inner.Left, Top: inner.Top, Width: inner.Width, Height: inner.Height - 1}
	m.list.Draw(s, listRegion, focused)
	components.DrawFooter(s, inner, "↑/↓ navigate  Enter select  Esc back")
}

type fieldModal struct {
	title string
	field *components.Field
}

func (m *fieldModal) HandleKey(ev *tcell.EventKey) bool {
	return m.field.HandleKey(ev)
}

func (m *fieldModal) Draw(s tcell.Screen, bounds layout.Region, focused bool) {
	w, h := 60, 9
	if bounds.Width-8 < w {
		w = bounds.Width - 8
	}
	inner := components.DrawCenteredBox(s, bounds, w, h, m.title)
	m.field.Draw(s, inner, focused)
}
