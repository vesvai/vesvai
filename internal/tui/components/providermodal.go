package components

import (
	"image"
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

const maxProviderRows = 8

type ProviderModal struct {
	*Base
	providers []string
	query     []rune
	cursor    int
	offset    int
	selected  string
	key       []rune
	step      int
	errMsg    string

	OnSubmit func(name, apiKey string)
	OnClose  func()
}

func NewProviderModal(providers []string) *ProviderModal {
	p := &ProviderModal{Base: NewBase("provider"), providers: providers}
	p.SetDraw(p.draw)
	return p
}

func (p *ProviderModal) Reset() {
	p.step = 0
	p.query = nil
	p.cursor = 0
	p.offset = 0
	p.selected = ""
	p.key = nil
	p.errMsg = ""
}

func (p *ProviderModal) DesiredHeight() int {
	if p.step == 1 {
		return 6
	}
	n := len(p.filtered())
	if n > maxProviderRows {
		n = maxProviderRows
	}
	if n == 0 {
		n = 1
	}
	return 4 + n
}

func (p *ProviderModal) filtered() []string {
	q := strings.ToLower(string(p.query))
	if q == "" {
		return p.providers
	}
	var out []string
	for _, prov := range p.providers {
		if strings.Contains(strings.ToLower(prov), q) {
			out = append(out, prov)
		}
	}
	return out
}

func (p *ProviderModal) draw(s tcell.Screen, pal *tui.Palette) {
	rect := p.Bounds()
	if rect.Dx() < 12 || rect.Dy() < 5 {
		return
	}
	inner := drawModalBox(s, rect, "Connect provider", pal)
	ix0, iy0 := inner.Min.X, inner.Min.Y

	if p.step == 1 {
		p.drawKeyStep(s, pal, ix0, iy0, inner)
		return
	}
	p.drawListStep(s, pal, ix0, iy0, inner)
}

func (p *ProviderModal) drawListStep(s tcell.Screen, pal *tui.Palette, ix0, iy0 int, inner image.Rectangle) {
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
	dimStyle := pal.Style(pal.Muted, pal.Surface)

	for row := 0; row < inner.Dy()-2; row++ {
		ei := p.offset + row
		line := tui.Line{{R: ' ', S: normalStyle}}
		if len(list) == 0 {
			for _, r := range "no providers available" {
				line = append(line, tui.Cell{R: r, S: dimStyle})
			}
			tui.DrawLine(s, ix0, iy0+2+row, line)
			break
		}
		if ei >= len(list) {
			break
		}
		prov := list[ei]
		selected := ei == p.cursor
		style := normalStyle
		if selected {
			style = selStyle
		}
		line = tui.Line{{R: ' ', S: style}}
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
		for _, r := range prov {
			line = append(line, tui.Cell{R: r, S: nameStyle})
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

func (p *ProviderModal) drawKeyStep(s tcell.Screen, pal *tui.Palette, ix0, iy0 int, inner image.Rectangle) {
	iw := inner.Dx()

	headStyle := pal.Style(pal.Accent, pal.Surface).Bold(true)
	head := tui.Line{{R: ' ', S: headStyle}}
	for _, r := range "provider: " + p.selected {
		head = append(head, tui.Cell{R: r, S: headStyle})
	}
	tui.DrawLine(s, ix0, iy0, head)

	labelStyle := pal.Style(pal.Muted, pal.Surface)
	label := tui.Line{{R: ' ', S: labelStyle}}
	for _, r := range "api key" {
		label = append(label, tui.Cell{R: r, S: labelStyle})
	}
	tui.DrawLine(s, ix0, iy0+1, label)

	valueStyle := pal.Style(pal.Foreground, pal.Surface)
	val := tui.Line{{R: ' ', S: valueStyle}, {R: ' ', S: valueStyle}}
	for _, r := range p.maskedKey() {
		val = append(val, tui.Cell{R: r, S: valueStyle})
	}
	if p.Focused() {
		val = append(val, tui.Cell{R: ' ', S: pal.Style(pal.Background, pal.Accent)})
	}
	if len(val) > iw-1 {
		val = val[:iw-1]
	}
	for len(val) < iw {
		val = append(val, tui.Cell{R: ' ', S: valueStyle})
	}
	tui.DrawLine(s, ix0, iy0+2, val)

	hint := " enter submit · esc back"
	if p.errMsg != "" {
		hint = p.errMsg
	}
	hintStyle := pal.Style(pal.Muted, pal.Surface)
	if p.errMsg != "" {
		hintStyle = pal.Style(pal.Error, pal.Surface)
	}
	h := tui.Line{{R: ' ', S: hintStyle}}
	for _, r := range hint {
		h = append(h, tui.Cell{R: r, S: hintStyle})
	}
	tui.DrawLine(s, ix0, iy0+inner.Dy()-1, h)
}

func (p *ProviderModal) maskedKey() []rune {
	out := make([]rune, len(p.key))
	for i := range out {
		out[i] = '•'
	}
	return out
}

func (p *ProviderModal) HandleEvent(ev tcell.Event) bool {
	e, ok := ev.(*tcell.EventKey)
	if !ok {
		return false
	}
	if p.step == 1 {
		return p.handleKeyStep(e)
	}
	return p.handleListStep(e)
}

func (p *ProviderModal) handleListStep(e *tcell.EventKey) bool {
	switch e.Key() {
	case tcell.KeyEsc:
		if p.OnClose != nil {
			p.OnClose()
		}
		return true
	case tcell.KeyEnter:
		list := p.filtered()
		if p.cursor >= 0 && p.cursor < len(list) {
			p.selected = list[p.cursor]
			p.step = 1
			p.errMsg = ""
			p.RequestRender()
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

func (p *ProviderModal) handleKeyStep(e *tcell.EventKey) bool {
	switch e.Key() {
	case tcell.KeyEsc:
		p.step = 0
		p.key = nil
		p.errMsg = ""
		p.RequestRender()
		return true
	case tcell.KeyEnter:
		if strings.TrimSpace(string(p.key)) == "" {
			p.errMsg = "api key required"
			p.RequestRender()
			return true
		}
		if p.OnSubmit != nil {
			p.OnSubmit(p.selected, strings.TrimSpace(string(p.key)))
		}
		return true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(p.key) > 0 {
			p.key = p.key[:len(p.key)-1]
			p.errMsg = ""
			p.RequestRender()
		}
		return true
	case tcell.KeyRune:
		r := e.Rune()
		if r >= 32 && !unicode.IsControl(r) {
			p.key = append(p.key, r)
			p.errMsg = ""
			p.RequestRender()
			return true
		}
	}
	return false
}
