package components

import (
	"github.com/gdamore/tcell/v2"

	"github.com/vesvai/vesvai/internal/tui/layout"
	"github.com/vesvai/vesvai/internal/tui/styles"
	"github.com/vesvai/vesvai/internal/utils/search"
)

type ListItem struct {
	Label    string
	Detail   string
	Data     any
	Marked   bool
	Disabled bool
}

type List struct {
	title    string
	all      []ListItem
	filtered []int
	index    int
	scroll   int
	filter   []rune
	search   bool
	onSelect func(index int, item ListItem)
}

func NewList(title string) *List {
	return &List{title: title}
}

func (l *List) SetTitle(title string) { l.title = title }

func (l *List) Title() string { return l.title }

func (l *List) SetSearchable(b bool) { l.search = b }

func (l *List) SetOnSelect(fn func(index int, item ListItem)) { l.onSelect = fn }

func (l *List) SetItems(items []ListItem) {
	l.all = items
	l.index = 0
	l.scroll = 0
	l.filter = nil
	l.rebuild()
}

func (l *List) Items() []ListItem { return l.all }

func (l *List) Count() int { return len(l.filtered) }

func (l *List) FilterText() string { return string(l.filter) }

func (l *List) Selected() (ListItem, bool) {
	if len(l.filtered) == 0 {
		return ListItem{}, false
	}
	return l.all[l.filtered[l.index]], true
}

func (l *List) SelectedIndex() int {
	if len(l.filtered) == 0 {
		return -1
	}
	return l.filtered[l.index]
}

func (l *List) rebuild() {
	q := string(l.filter)
	l.filtered = l.filtered[:0]
	if q == "" {
		for i := range l.all {
			l.filtered = append(l.filtered, i)
		}
	} else {
		for i, it := range l.all {
			if search.Score(q, it.Label) >= 0 || search.Score(q, it.Detail) >= 0 {
				l.filtered = append(l.filtered, i)
			}
		}
	}
	if l.index >= len(l.filtered) {
		l.index = 0
	}
}

func (l *List) MoveUp() {
	if len(l.filtered) == 0 {
		return
	}
	if l.index > 0 {
		l.index--
	}
}

func (l *List) MoveDown() {
	if len(l.filtered) == 0 {
		return
	}
	if l.index < len(l.filtered)-1 {
		l.index++
	}
}

func (l *List) MovePageUp(page int) {
	l.index -= page
	if l.index < 0 {
		l.index = 0
	}
}

func (l *List) MovePageDown(page int) {
	l.index += page
	if l.index >= len(l.filtered) {
		l.index = len(l.filtered) - 1
	}
}

func (l *List) Home() { l.index = 0 }

func (l *List) End() {
	if len(l.filtered) > 0 {
		l.index = len(l.filtered) - 1
	}
}

func (l *List) HandleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyUp:
		l.MoveUp()
		return true
	case tcell.KeyDown:
		l.MoveDown()
		return true
	case tcell.KeyPgUp:
		l.MovePageUp(10)
		return true
	case tcell.KeyPgDn:
		l.MovePageDown(10)
		return true
	case tcell.KeyHome:
		l.Home()
		return true
	case tcell.KeyEnd:
		l.End()
		return true
	case tcell.KeyEnter:
		if l.onSelect != nil {
			if item, ok := l.Selected(); ok {
				l.onSelect(l.SelectedIndex(), item)
			}
		}
		return true
	}
	if l.search {
		switch ev.Key() {
		case tcell.KeyRune:
			if r := ev.Rune(); r != 0 && r >= ' ' {
				l.filter = append(l.filter, r)
				l.rebuild()
				return true
			}
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if len(l.filter) > 0 {
				l.filter = l.filter[:len(l.filter)-1]
				l.rebuild()
				return true
			}
		case tcell.KeyEscape:
			if len(l.filter) > 0 {
				l.filter = nil
				l.rebuild()
				return true
			}
		}
	}
	return false
}

func (l *List) Draw(s tcell.Screen, bounds layout.Region, _ bool) {
	th := styles.Current()
	style := th.Base().Background(th.InputBg)
	FillRegion(s, bounds, ' ', style)

	top := bounds.Top
	if l.search && len(l.filter) > 0 {
		DrawText(s, bounds.Left+1, top, "search: "+string(l.filter), th.Base().Foreground(th.Accent).Background(th.InputBg))
		top++
	}

	if len(l.filtered) == 0 {
		DrawText(s, bounds.Left+1, top, "(no items)", th.Base().Foreground(th.Placeholder).Background(th.InputBg))
		return
	}

	visible := bounds.Bottom() - top
	if l.index < l.scroll {
		l.scroll = l.index
	}
	if l.index >= l.scroll+visible {
		l.scroll = l.index - visible + 1
	}

	for i := 0; i < visible; i++ {
		idx := l.scroll + i
		if idx >= len(l.filtered) {
			break
		}
		item := l.all[l.filtered[idx]]
		y := top + i
		rowStyle := style
		if idx == l.index {
			rowStyle = th.Base().Foreground(th.InputText).Background(th.Selection)
		} else if item.Disabled {
			rowStyle = th.Base().Foreground(th.Muted).Background(th.InputBg)
		}
		label := item.Label
		if item.Marked {
			label = "● " + label
		}
		DrawText(s, bounds.Left+1, y, TruncateTo(label, bounds.Width-2), rowStyle)
		if item.Detail != "" {
			d := TruncateTo(item.Detail, bounds.Width/2)
			DrawText(s, bounds.Right()-len(d)-1, y, d, rowStyle)
		}
	}
}
