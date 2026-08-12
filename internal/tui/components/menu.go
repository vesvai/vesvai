package components

import (
	"image"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

type MenuItem struct {
	ID    string
	Label string
	Run   func()
}

type Menu struct {
	*Base
	items  []MenuItem
	cursor int

	OnClose func()
}

func NewMenu(items []MenuItem) *Menu {
	m := &Menu{Base: NewBase("menu"), items: items}
	m.SetDraw(m.draw)
	return m
}

func (m *Menu) OpenAt(x, y, sw, sh int) {
	w := m.desiredWidth()
	h := len(m.items) + 2
	if x+w > sw {
		x = sw - w
	}
	if y+h > sh {
		y = sh - h
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	m.Layout(image.Rect(x, y, x+w, y+h))
	m.MarkDirty()
}

func (m *Menu) desiredWidth() int {
	w := 6
	for _, it := range m.items {
		if l := len(it.Label) + 4; l > w {
			w = l
		}
	}
	return w + 2
}

func (m *Menu) draw(s tcell.Screen, pal *tui.Palette) {
	rect := m.Bounds()
	w, h := rect.Dx(), rect.Dy()
	if w < 3 || h < 3 {
		return
	}
	x0, y0 := rect.Min.X, rect.Min.Y

	border := pal.Style(pal.Border, pal.Surface)
	bg := pal.Style(pal.Foreground, pal.Surface)

	top := tui.Line{{R: '┌', S: border}}
	for len(top) < w-1 {
		top = append(top, tui.Cell{R: '─', S: border})
	}
	top = append(top, tui.Cell{R: '┐', S: border})
	tui.DrawLine(s, x0, y0, top)

	for y := 1; y < h-1; y++ {
		s.SetContent(x0, y0+y, '│', nil, border)
		s.SetContent(x0+w-1, y0+y, '│', nil, border)
		for x := 1; x < w-1; x++ {
			s.SetContent(x0+x, y0+y, ' ', nil, bg)
		}
	}

	bottom := tui.Line{{R: '└', S: border}}
	for len(bottom) < w-1 {
		bottom = append(bottom, tui.Cell{R: '─', S: border})
	}
	bottom = append(bottom, tui.Cell{R: '┘', S: border})
	tui.DrawLine(s, x0, y0+h-1, bottom)

	selStyle := pal.Style(pal.Foreground, pal.Selection)
	normalStyle := pal.Style(pal.Foreground, pal.Surface)
	for i, it := range m.items {
		selected := i == m.cursor
		style := normalStyle
		if selected {
			style = selStyle
		}
		line := tui.Line{{R: ' ', S: style}}
		if selected {
			line = append(line, tui.Cell{R: '▸', S: style})
		} else {
			line = append(line, tui.Cell{R: ' ', S: style})
		}
		line = append(line, tui.Cell{R: ' ', S: style})
		labelStyle := style
		if selected {
			labelStyle = style.Foreground(pal.Accent)
		}
		for _, r := range it.Label {
			line = append(line, tui.Cell{R: r, S: labelStyle})
		}
		for len(line) < w-2 {
			line = append(line, tui.Cell{R: ' ', S: style})
		}
		tui.DrawLine(s, x0+1, y0+1+i, line)
	}
}

func (m *Menu) HandleEvent(ev tcell.Event) bool {
	switch e := ev.(type) {
	case *tcell.EventKey:
		switch e.Key() {
		case tcell.KeyEsc:
			if m.OnClose != nil {
				m.OnClose()
			}
			return true
		case tcell.KeyUp:
			if m.cursor > 0 {
				m.cursor--
				m.RequestRender()
			}
			return true
		case tcell.KeyDown:
			if m.cursor < len(m.items)-1 {
				m.cursor++
				m.RequestRender()
			}
			return true
		case tcell.KeyEnter:
			m.run()
			return true
		}
	case *tcell.EventMouse:
		if e.Buttons()&tcell.Button1 != 0 {
			x, y := e.Position()
			b := m.Bounds()
			if x >= b.Min.X && x < b.Max.X && y >= b.Min.Y && y < b.Max.Y {
				idx := y - b.Min.Y - 1
				if idx >= 0 && idx < len(m.items) {
					m.cursor = idx
					m.run()
				}
			} else if m.OnClose != nil {
				m.OnClose()
			}
			return true
		}
	}
	return false
}

func (m *Menu) run() {
	if m.cursor >= 0 && m.cursor < len(m.items) && m.items[m.cursor].Run != nil {
		m.items[m.cursor].Run()
		return
	}
	if m.OnClose != nil {
		m.OnClose()
	}
}
