package components

import (
	"github.com/gdamore/tcell/v2"

	"github.com/vesvai/vesvai/internal/tui/styles"
)

type Tabs struct {
	names   []string
	active  int
	focused bool
}

func NewTabs(names []string) *Tabs {
	return &Tabs{names: names}
}

func (t *Tabs) SetActive(n int) { t.active = n }

func (t *Tabs) Active() int { return t.active }

func (t *Tabs) Count() int { return len(t.names) }

func (t *Tabs) SetFocused(f bool) { t.focused = f }

func (t *Tabs) Focused() bool { return t.focused }

func (t *Tabs) HandleKey(ev *tcell.EventKey) bool {
	if !t.focused {
		return false
	}
	switch ev.Key() {
	case tcell.KeyLeft:
		if t.active > 0 {
			t.active--
		} else {
			t.active = len(t.names) - 1
		}
		return true
	case tcell.KeyRight:
		if t.active < len(t.names)-1 {
			t.active++
		} else {
			t.active = 0
		}
		return true
	}
	return false
}

func (t *Tabs) Draw(s tcell.Screen, x, y int, focused bool) {
	th := styles.Current()
	cx := x
	for i, name := range t.names {
		style := th.Base().Foreground(th.Hint).Background(th.InputBg)
		if i == t.active {
			if focused {
				style = th.Base().Foreground(th.InputBg).Background(th.Accent)
			} else {
				style = th.Base().Foreground(th.InputBg).Background(th.AccentDim)
			}
		}
		DrawText(s, cx, y, " "+name+" ", style)
		cx += len([]rune(name)) + 3
	}
}
