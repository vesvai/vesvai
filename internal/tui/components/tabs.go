package components

import (
	"github.com/gdamore/tcell/v2"

	"github.com/vesvai/vesvai/internal/tui/styles"
)

type Tabs struct {
	names  []string
	active int
}

func NewTabs(names []string) *Tabs {
	return &Tabs{names: names}
}

func (t *Tabs) SetActive(n int) { t.active = n }

func (t *Tabs) Active() int { return t.active }

func (t *Tabs) Count() int { return len(t.names) }

func (t *Tabs) Draw(s tcell.Screen, x, y int) {
	th := styles.Current()
	cx := x
	for i, name := range t.names {
		style := th.Base().Foreground(th.Hint).Background(th.InputBg)
		if i == t.active {
			style = th.Base().Foreground(th.InputBg).Background(th.Accent)
		}
		DrawText(s, cx, y, " "+name+" ", style)
		cx += len([]rune(name)) + 3
	}
}
