package components

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

const maxModelRows = 10

type ModelPicker struct {
	*Base
	models []tui.ModelInfo
	query  []rune
	cursor int
	offset int

	OnSelect func(m tui.ModelInfo)
	OnBack   func()
}

func NewModelPicker(models []tui.ModelInfo) *ModelPicker {
	p := &ModelPicker{Base: NewBase("models"), models: models}
	p.SetDraw(p.draw)
	return p
}

func (p *ModelPicker) SetModels(models []tui.ModelInfo) {
	p.models = models
	if p.cursor >= len(p.models) {
		p.cursor = len(p.models) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
	p.RequestRender()
}

func (p *ModelPicker) Models() []tui.ModelInfo { return p.models }

func (p *ModelPicker) Reset() {
	p.query = nil
	p.cursor = 0
	p.offset = 0
}

func (p *ModelPicker) DesiredHeight() int {
	n := len(p.filtered())
	if n > maxModelRows {
		n = maxModelRows
	}
	return 4 + n
}

func (p *ModelPicker) filtered() []tui.ModelInfo {
	q := strings.ToLower(string(p.query))
	if q == "" {
		return p.models
	}
	var out []tui.ModelInfo
	for _, m := range p.models {
		if strings.Contains(strings.ToLower(m.Name+" "+m.Provider), q) {
			out = append(out, m)
		}
	}
	return out
}

func formatWindow(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%dM", n/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%dK", n/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func (p *ModelPicker) draw(s tcell.Screen, pal *tui.Palette) {
	rect := p.Bounds()
	if rect.Dx() < 4 || rect.Dy() < 4 {
		return
	}
	inner := drawModalBox(s, rect, "Switch model", pal)
	ix0, iy0 := inner.Min.X, inner.Min.Y
	iw := inner.Dx()

	searchStyle := pal.Style(pal.Foreground, pal.Surface)
	search := tui.Line{{R: '❯', S: pal.Style(pal.Accent, pal.Surface).Bold(true)}, {R: ' ', S: searchStyle}}
	for _, r := range p.query {
		search = append(search, tui.Cell{R: r, S: searchStyle})
	}
	tui.DrawLine(s, ix0, iy0, search)
	if p.Focused() {
		cx := ix0 + 2 + tui.DisplayWidth(string(p.query))
		if cx < inner.Max.X-1 {
			s.SetContent(cx, iy0, ' ', nil, pal.Style(pal.Background, pal.Accent))
		}
	}

	list := p.filtered()
	if p.cursor >= len(list) {
		p.cursor = len(list) - 1
	}
	if p.cursor < p.offset {
		p.offset = p.cursor
	}
	if p.cursor >= p.offset+inner.Dy()-2 {
		p.offset = p.cursor - inner.Dy() + 3
	}
	if p.offset < 0 {
		p.offset = 0
	}

	selStyle := pal.Style(pal.Foreground, pal.Selection)
	normalStyle := pal.Style(pal.Foreground, pal.Surface)

	for row := 0; row < inner.Dy()-2; row++ {
		ei := p.offset + row
		if ei >= len(list) {
			break
		}
		m := list[ei]
		selected := ei == p.cursor
		style := normalStyle
		if selected {
			style = selStyle
		}
		metaStyle := style.Foreground(pal.Muted)

		line := tui.Line{{R: ' ', S: style}}
		if selected {
			line = append(line, tui.Cell{R: '▸', S: style})
		} else {
			line = append(line, tui.Cell{R: ' ', S: style})
		}
		line = append(line, tui.Cell{R: ' ', S: style})
		nameStyle := style
		if selected {
			nameStyle = style.Foreground(pal.Accent)
		}
		for _, r := range m.Name {
			line = append(line, tui.Cell{R: r, S: nameStyle})
		}
		meta := fmt.Sprintf("  %s · %s · %s", m.Provider, m.Effort, formatWindow(m.ContextWindow))
		for _, r := range meta {
			line = append(line, tui.Cell{R: r, S: metaStyle})
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

func (p *ModelPicker) HandleEvent(ev tcell.Event) bool {
	switch e := ev.(type) {
	case *tcell.EventKey:
		return p.handleKey(e)
	}
	return false
}

func (p *ModelPicker) handleKey(e *tcell.EventKey) bool {
	switch e.Key() {
	case tcell.KeyEsc:
		if p.OnBack != nil {
			p.OnBack()
		}
		return true
	case tcell.KeyEnter:
		list := p.filtered()
		if p.cursor >= 0 && p.cursor < len(list) {
			if p.OnSelect != nil {
				p.OnSelect(list[p.cursor])
			}
		}
		return true
	case tcell.KeyUp:
		if p.cursor > 0 {
			p.cursor--
			p.RequestRender()
		}
		return true
	case tcell.KeyDown:
		if p.cursor < len(p.filtered())-1 {
			p.cursor++
			p.RequestRender()
		}
		return true
	case tcell.KeyPgUp:
		p.cursor -= 5
		if p.cursor < 0 {
			p.cursor = 0
		}
		p.RequestRender()
		return true
	case tcell.KeyPgDn:
		p.cursor += 5
		if p.cursor >= len(p.filtered()) {
			p.cursor = len(p.filtered()) - 1
		}
		p.RequestRender()
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
