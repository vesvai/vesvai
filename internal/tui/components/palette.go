package components

import (
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

type Action struct {
	ID       string
	Label    string
	Category string
	Hint     string
	Run      func()
}

type palItem struct {
	isHeader bool
	label    string
	hint     string
	action   *Action
}

const maxPaletteItems = 10

type Palette struct {
	*Base
	actions []Action
	query   []rune
	cursor  int
	offset  int

	OnRun   func(a Action)
	OnClose func()
}

func NewPalette(actions []Action) *Palette {
	p := &Palette{Base: NewBase("palette"), actions: actions}
	p.SetDraw(p.draw)
	return p
}

func (p *Palette) Reset() {
	p.query = nil
	p.cursor = 0
	p.offset = 0
}

func (p *Palette) DesiredHeight() int {
	n := len(p.entries())
	if n > maxPaletteItems+3 {
		n = maxPaletteItems + 3
	}
	return 4 + n
}

func (p *Palette) entries() []palItem {
	q := strings.ToLower(string(p.query))
	var out []palItem
	lastCat := ""
	for i := range p.actions {
		a := &p.actions[i]
		if q != "" && !matchesAction(a, q) {
			continue
		}
		if a.Category != lastCat {
			out = append(out, palItem{isHeader: true, label: a.Category})
			lastCat = a.Category
		}
		out = append(out, palItem{label: a.Label, hint: a.Hint, action: a})
	}
	return out
}

func matchesAction(a *Action, q string) bool {
	hay := strings.ToLower(a.Label + " " + a.Category)
	return strings.Contains(hay, q)
}

func (p *Palette) cursorItem() *Action {
	idx := 0
	for _, e := range p.entries() {
		if e.isHeader {
			continue
		}
		if idx == p.cursor {
			return e.action
		}
		idx++
	}
	return nil
}

func (p *Palette) draw(s tcell.Screen, pal *tui.Palette) {
	rect := p.Bounds()
	if rect.Dx() < 4 || rect.Dy() < 4 {
		return
	}
	inner := drawModalBox(s, rect, "Actions", pal)
	ix0, iy0 := inner.Min.X, inner.Min.Y
	iw := inner.Dx()

	searchStyle := pal.Style(pal.Foreground, pal.Surface)
	search := tui.Line{{R: '❯', S: pal.Style(pal.Accent, pal.Surface).Bold(true)}, {R: ' ', S: searchStyle}}
	for _, r := range p.query {
		search = append(search, tui.Cell{R: r, S: searchStyle})
	}
	if !p.Focused() {
		search = append(search, tui.Cell{R: ' ', S: searchStyle})
	}
	tui.DrawLine(s, ix0, iy0, search)
	if p.Focused() {
		cx := ix0 + 2 + tui.DisplayWidth(string(p.query))
		if cx < inner.Max.X-1 {
			s.SetContent(cx, iy0, ' ', nil, pal.Style(pal.Background, pal.Accent))
		}
	}

	entries := p.entries()

	cursorRow := -1
	{
		idx := 0
		for i, e := range entries {
			if e.isHeader {
				continue
			}
			if idx == p.cursor {
				cursorRow = i
				break
			}
			idx++
		}
	}

	listH := inner.Dy() - 2
	if cursorRow >= 0 {
		if cursorRow < p.offset {
			p.offset = cursorRow
		}
		if cursorRow >= p.offset+listH {
			p.offset = cursorRow - listH + 1
		}
	}
	if p.offset < 0 {
		p.offset = 0
	}

	selStyle := pal.Style(pal.Foreground, pal.Selection)
	normalStyle := pal.Style(pal.Foreground, pal.Surface)
	headerStyle := pal.Style(pal.Muted, pal.Surface)

	itemIdx := 0
	for row := 0; row < listH; row++ {
		ei := p.offset + row
		if ei >= len(entries) {
			break
		}
		e := entries[ei]
		style := normalStyle
		if !e.isHeader && itemIdx == p.cursor {
			style = selStyle
		}
		line := tui.Line{{R: ' ', S: style}}
		if e.isHeader {
			line = append(line, tui.LineFromSegments([]tui.Segment{
				{Text: " " + e.label, Style: headerStyle.Bold(true)},
			}, iw-2)...)
		} else {
			marker := ' '
			if itemIdx == p.cursor {
				marker = '▸'
			}
			line = append(line, tui.Cell{R: marker, S: style}, tui.Cell{R: ' ', S: style})
			labelStyle := style
			if itemIdx == p.cursor {
				labelStyle = style.Foreground(pal.Accent)
			}
			for _, r := range e.label {
				line = append(line, tui.Cell{R: r, S: labelStyle})
			}
			if e.hint != "" {
				hintStyle := style.Foreground(pal.Muted)
				hint := tui.LineFromSegments([]tui.Segment{{Text: " " + e.hint, Style: hintStyle}}, len(e.hint)+1)
				line = append(line, hint...)
			}
			itemIdx++
		}
		if len(line) > iw {
			line = line[:iw]
		}
		for len(line) < iw {
			line = append(line, tui.Cell{R: ' ', S: style})
		}
		tui.DrawLine(s, ix0, iy0+2+row, line)
	}
}

func (p *Palette) HandleEvent(ev tcell.Event) bool {
	switch e := ev.(type) {
	case *tcell.EventKey:
		return p.handleKey(e)
	}
	return false
}

func (p *Palette) handleKey(e *tcell.EventKey) bool {
	switch e.Key() {
	case tcell.KeyEsc:
		if p.OnClose != nil {
			p.OnClose()
		}
		return true
	case tcell.KeyEnter:
		if a := p.cursorItem(); a != nil {
			if a.Run != nil {
				a.Run()
			} else if p.OnRun != nil {
				p.OnRun(*a)
			}
		}
		return true
	case tcell.KeyUp:
		p.moveCursor(-1)
		return true
	case tcell.KeyDown:
		p.moveCursor(1)
		return true
	case tcell.KeyPgUp:
		p.moveCursor(-5)
		return true
	case tcell.KeyPgDn:
		p.moveCursor(5)
		return true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(p.query) > 0 {
			p.query = p.query[:len(p.query)-1]
			p.cursor = 0
			p.offset = 0
			p.RequestRender()
		}
		return true
	case tcell.KeyCtrlU:
		p.query = nil
		p.cursor = 0
		p.offset = 0
		p.RequestRender()
		return true
	case tcell.KeyRune:
		r := e.Rune()
		if r >= 32 && !unicode.IsControl(r) {
			p.query = append(p.query, r)
			p.cursor = 0
			p.offset = 0
			p.RequestRender()
			return true
		}
	}
	return false
}

func (p *Palette) moveCursor(delta int) {
	entries := p.entries()
	if len(entries) == 0 {
		return
	}
	cur := -1
	idx := 0
	for i, e := range entries {
		if e.isHeader {
			continue
		}
		if idx == p.cursor {
			cur = i
			break
		}
		idx++
	}
	if cur < 0 {
		p.cursor = 0
		p.RequestRender()
		return
	}
	step := delta
	if step < 0 {
		step = -step
	}
	for s := 0; s < step; s++ {
		if delta > 0 {
			cur++
			for cur < len(entries) && entries[cur].isHeader {
				cur++
			}
			if cur >= len(entries) {
				cur = -1
				break
			}
		} else {
			cur--
			for cur >= 0 && entries[cur].isHeader {
				cur--
			}
			if cur < 0 {
				break
			}
		}
	}
	if cur < 0 {
		return
	}
	idx = 0
	for i, e := range entries {
		if e.isHeader {
			continue
		}
		if i == cur {
			p.cursor = idx
			break
		}
		idx++
	}
	p.RequestRender()
}
