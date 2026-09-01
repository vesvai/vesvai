package tui

import (
	"github.com/gdamore/tcell/v2"

	"github.com/vesvai/vesvai/internal/tui/components"
	"github.com/vesvai/vesvai/internal/tui/layout"
	"github.com/vesvai/vesvai/internal/tui/styles"
)

type UpdateModal struct {
	version  string
	onUpdate func()
	onSkip   func()
	selected int
}

func NewUpdateModal(version string, onUpdate, onSkip func()) *UpdateModal {
	return &UpdateModal{
		version:  version,
		onUpdate: onUpdate,
		onSkip:   onSkip,
		selected: 0,
	}
}

func (m *UpdateModal) HandleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyLeft, tcell.KeyRight:
		m.selected = (m.selected + 1) % 2
		return true
	case tcell.KeyEnter:
		if m.selected == 0 && m.onUpdate != nil {
			m.onUpdate()
		} else if m.selected == 1 && m.onSkip != nil {
			m.onSkip()
		}
		return true
	case tcell.KeyEsc:
		if m.onSkip != nil {
			m.onSkip()
		}
		return true
	}
	return false
}

func (m *UpdateModal) Draw(s tcell.Screen, bounds layout.Region, focused bool) {
	w, h := 50, 8
	if bounds.Width-8 < w {
		w = bounds.Width - 8
	}
	inner := components.DrawCenteredBox(s, bounds, w, h, "Update Available")

	th := styles.Current()
	msg := "A new version (" + m.version + ") is available."
	components.DrawText(s, inner.Left, inner.Top, msg, th.Base())

	components.DrawText(s, inner.Left, inner.Top+2, "Do you want to update?", th.Base())

	btnY := inner.Top + 4
	updateLabel := "[ Update ]"
	skipLabel := "[ Later ]"

	if m.selected == 0 {
		components.DrawText(s, inner.Left+5, btnY, updateLabel, th.Base().Foreground(th.Accent))
		components.DrawText(s, inner.Left+25, btnY, skipLabel, th.Base())
	} else {
		components.DrawText(s, inner.Left+5, btnY, updateLabel, th.Base())
		components.DrawText(s, inner.Left+25, btnY, skipLabel, th.Base().Foreground(th.Accent))
	}

	components.DrawFooter(s, inner, "←/→ select  Enter confirm  Esc skip")
}
