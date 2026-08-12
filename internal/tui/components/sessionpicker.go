package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

type Session struct {
	ID      string
	Title   string
	Updated string
}

const maxSessionRows = 8

type SessionPicker struct {
	*Base
	sessions []Session
	cursor   int
	offset   int

	OnSelect func(s *Session)
	OnBack   func()
}

func NewSessionPicker(sessions []Session) *SessionPicker {
	p := &SessionPicker{Base: NewBase("sessions"), sessions: sessions}
	p.SetDraw(p.draw)
	return p
}

func (p *SessionPicker) SetSessions(sessions []Session) {
	p.sessions = sessions
	if p.cursor >= len(p.sessions) {
		p.cursor = len(p.sessions) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
	p.RequestRender()
}

func (p *SessionPicker) Sessions() []Session { return p.sessions }

func (p *SessionPicker) Reset() {
	p.cursor = 0
	p.offset = 0
}

func (p *SessionPicker) DesiredHeight() int {
	n := len(p.sessions)
	if n > maxSessionRows {
		n = maxSessionRows
	}
	return 4 + 2*n
}

func (p *SessionPicker) draw(s tcell.Screen, pal *tui.Palette) {
	rect := p.Bounds()
	if rect.Dx() < 4 || rect.Dy() < 4 {
		return
	}
	inner := drawModalBox(s, rect, "Switch session", pal)
	ix0, iy0 := inner.Min.X, inner.Min.Y
	iw := inner.Dx()
	listH := inner.Dy() - 2

	if p.cursor >= len(p.sessions) {
		p.cursor = len(p.sessions) - 1
	}
	if p.cursor < p.offset {
		p.offset = p.cursor
	}
	if p.cursor >= p.offset+listH/2 {
		p.offset = p.cursor - listH/2 + 1
	}
	if p.offset < 0 {
		p.offset = 0
	}

	selStyle := pal.Style(pal.Foreground, pal.Selection)
	normalStyle := pal.Style(pal.Foreground, pal.Surface)

	row := 0
	for si := p.offset; si < len(p.sessions) && row+1 < listH; si++ {
		sess := &p.sessions[si]
		selected := si == p.cursor

		style := normalStyle
		if selected {
			style = selStyle
		}
		title := tui.Line{{R: ' ', S: style}}
		if selected {
			title = append(title, tui.Cell{R: '▸', S: style})
		} else {
			title = append(title, tui.Cell{R: ' ', S: style})
		}
		title = append(title, tui.Cell{R: ' ', S: style})
		for _, r := range sess.Title {
			title = append(title, tui.Cell{R: r, S: style})
		}
		if len(title) > iw {
			title = title[:iw]
		}
		for len(title) < iw {
			title = append(title, tui.Cell{R: ' ', S: style})
		}
		tui.DrawLine(s, ix0, iy0+1+row, title)
		row++

		meta := tui.Line{{R: ' ', S: style}, {R: ' ', S: style}, {R: ' ', S: style}}
		metaStyle := style.Foreground(pal.Muted)
		for _, r := range sess.Updated {
			meta = append(meta, tui.Cell{R: r, S: metaStyle})
		}
		if len(meta) > iw {
			meta = meta[:iw]
		}
		for len(meta) < iw {
			meta = append(meta, tui.Cell{R: ' ', S: style})
		}
		tui.DrawLine(s, ix0, iy0+1+row, meta)
		row++
	}
}

func (p *SessionPicker) HandleEvent(ev tcell.Event) bool {
	switch e := ev.(type) {
	case *tcell.EventKey:
		switch e.Key() {
		case tcell.KeyEsc:
			if p.OnBack != nil {
				p.OnBack()
			}
			return true
		case tcell.KeyEnter:
			if p.cursor >= 0 && p.cursor < len(p.sessions) {
				if p.OnSelect != nil {
					p.OnSelect(&p.sessions[p.cursor])
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
			if p.cursor < len(p.sessions)-1 {
				p.cursor++
				p.RequestRender()
			}
			return true
		case tcell.KeyPgUp:
			p.cursor -= 4
			if p.cursor < 0 {
				p.cursor = 0
			}
			p.RequestRender()
			return true
		case tcell.KeyPgDn:
			p.cursor += 4
			if p.cursor >= len(p.sessions) {
				p.cursor = len(p.sessions) - 1
			}
			p.RequestRender()
			return true
		}
	}
	return false
}
