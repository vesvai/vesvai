package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

const attachmentsPerPage = 3

type AttachmentBar struct {
	*Base
	attachments []*tui.Attachment
	cursor      int
}

func NewAttachmentBar() *AttachmentBar {
	b := &AttachmentBar{Base: NewBase("attachments"), cursor: -1}
	b.SetDraw(b.draw)
	return b
}

func (b *AttachmentBar) Page() int {
	if b.cursor < 0 {
		return 0
	}
	return b.cursor / attachmentsPerPage
}

func (b *AttachmentBar) Count() int { return len(b.attachments) }

func (b *AttachmentBar) Selected() *tui.Attachment {
	if b.cursor < 0 || b.cursor >= len(b.attachments) {
		return nil
	}
	return b.attachments[b.cursor]
}

func (b *AttachmentBar) Add(a *tui.Attachment) {
	b.attachments = append(b.attachments, a)
	if b.cursor < 0 {
		b.cursor = 0
	}
	b.RequestRender()
}

func (b *AttachmentBar) Remove(idx int) {
	if idx < 0 || idx >= len(b.attachments) {
		return
	}
	b.attachments = append(b.attachments[:idx], b.attachments[idx+1:]...)
	if len(b.attachments) == 0 {
		b.cursor = -1
	} else if b.cursor >= len(b.attachments) {
		b.cursor = len(b.attachments) - 1
	}
	b.RequestRender()
}

func (b *AttachmentBar) TakeAll() []*tui.Attachment {
	atts := b.attachments
	b.attachments = nil
	b.cursor = -1
	b.RequestRender()
	return atts
}

func (b *AttachmentBar) SetFocused(f bool) {
	if f && b.cursor < 0 && len(b.attachments) > 0 {
		b.cursor = 0
	}
	b.Base.SetFocused(f)
}

func (b *AttachmentBar) DesiredHeight() int {
	if len(b.attachments) == 0 {
		return 0
	}
	return 5
}

func (b *AttachmentBar) Focusable() bool { return len(b.attachments) > 0 }

func (b *AttachmentBar) pageCount() int {
	n := len(b.attachments)
	if n == 0 {
		return 0
	}
	return (n + attachmentsPerPage - 1) / attachmentsPerPage
}

func (b *AttachmentBar) draw(s tcell.Screen, pal *tui.Palette) {
	width := b.Width()
	if width < 12 || len(b.attachments) == 0 {
		return
	}
	x0, y0 := b.bounds.Min.X, b.bounds.Min.Y

	clear := pal.Style(pal.Foreground, pal.Background)
	for y := 0; y < b.Height(); y++ {
		for x := 0; x < width; x++ {
			s.SetContent(x0+x, y0+y, ' ', nil, clear)
		}
	}

	if b.cursor < 0 {
		b.cursor = 0
	}
	if b.cursor >= len(b.attachments) {
		b.cursor = len(b.attachments) - 1
	}
	page := b.cursor / attachmentsPerPage
	pages := b.pageCount()

	borderColor := pal.Border
	arrowColor := pal.Muted
	if b.Focused() {
		arrowColor = pal.Accent
	}

	arrowStyle := pal.Style(arrowColor, pal.Background).Bold(true)
	if pages > 1 {
		s.SetContent(x0, y0, '‹', nil, arrowStyle)
		s.SetContent(x0+width-1, y0, '›', nil, arrowStyle)
	}

	cardW := (width - 8) / 3
	if cardW < 10 {
		cardW = 10
	}
	start := page * attachmentsPerPage
	for i := 0; i < attachmentsPerPage; i++ {
		idx := start + i
		if idx >= len(b.attachments) {
			break
		}
		bc := borderColor
		if b.Focused() && idx == b.cursor {
			bc = pal.Accent
		}
		b.drawCard(s, pal, x0+2+i*(cardW+1), y0, cardW, b.attachments[idx], bc)
	}

	var dots tui.Line
	dim := pal.Style(pal.Muted, pal.Background)
	bg := pal.Style(pal.Foreground, pal.Background)
	for p := 0; p < pages; p++ {
		if p == page {
			dots = append(dots, tui.Cell{R: '●', S: pal.Style(pal.Accent, pal.Background)})
		} else {
			dots = append(dots, tui.Cell{R: '○', S: dim})
		}
		dots = append(dots, tui.Cell{R: ' ', S: bg})
	}
	dotX := x0 + (width-len(dots))/2
	tui.DrawLine(s, dotX, y0+4, dots)
}

func (b *AttachmentBar) drawCard(s tcell.Screen, pal *tui.Palette, x, y, w int, a *tui.Attachment, borderColor tcell.Color) {
	border := pal.Style(borderColor, pal.Surface)
	bg := pal.Style(pal.Foreground, pal.Surface)
	innerW := w - 2

	top := tui.Line{{R: '┌', S: border}}
	for len(top) < w-1 {
		top = append(top, tui.Cell{R: '─', S: border})
	}
	top = append(top, tui.Cell{R: '┐', S: border})
	tui.DrawLine(s, x, y, top)

	var icon rune
	switch a.Kind {
	case "image":
		icon = '🖼'
	case "video":
		icon = '🎬'
	case "pdf":
		icon = '📄'
	default:
		icon = '📎'
	}
	nameStyle := pal.Style(pal.Foreground, pal.Surface)
	row1 := tui.Line{{R: '│', S: border}, {R: ' ', S: bg}, {R: icon, S: bg}, {R: ' ', S: bg}}
	name := a.Name
	for row1.Width()+tui.DisplayWidth(name) > innerW {
		if len(name) <= 1 {
			break
		}
		name = name[:len(name)-1]
	}
	if a.Name != name {
		name += "…"
	}
	for _, r := range name {
		row1 = append(row1, tui.Cell{R: r, S: nameStyle})
	}
	for row1.Width() < w-1 {
		row1 = append(row1, tui.Cell{R: ' ', S: bg})
	}
	row1 = append(row1, tui.Cell{R: '│', S: border})
	tui.DrawLine(s, x, y+1, row1)

	size := pal.Style(pal.Muted, pal.Surface)
	row2 := tui.Line{{R: '│', S: border}, {R: ' ', S: bg}}
	sizeText := tui.FormatSize(a.Size)
	for _, r := range sizeText {
		row2 = append(row2, tui.Cell{R: r, S: size})
	}
	for row2.Width() < w-1 {
		row2 = append(row2, tui.Cell{R: ' ', S: bg})
	}
	row2 = append(row2, tui.Cell{R: '│', S: border})
	tui.DrawLine(s, x, y+2, row2)

	bottom := tui.Line{{R: '└', S: border}}
	for len(bottom) < w-1 {
		bottom = append(bottom, tui.Cell{R: '─', S: border})
	}
	bottom = append(bottom, tui.Cell{R: '┘', S: border})
	tui.DrawLine(s, x, y+3, bottom)
}

func (b *AttachmentBar) HandleEvent(ev tcell.Event) bool {
	switch e := ev.(type) {
	case *tcell.EventKey:
		switch e.Key() {
		case tcell.KeyLeft:
			if b.cursor > 0 {
				b.cursor--
				b.RequestRender()
			}
			return true
		case tcell.KeyRight:
			if b.cursor < len(b.attachments)-1 {
				b.cursor++
				b.RequestRender()
			}
			return true
		case tcell.KeyHome:
			if len(b.attachments) > 0 {
				b.cursor = 0
				b.RequestRender()
			}
			return true
		case tcell.KeyEnd:
			if len(b.attachments) > 0 {
				b.cursor = len(b.attachments) - 1
				b.RequestRender()
			}
			return true
		case tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyDelete:
			b.Remove(b.cursor)
			return true
		}
	case *tcell.EventMouse:
		if e.Buttons()&tcell.Button1 != 0 {
			x, y := e.Position()
			if y == b.bounds.Min.Y {
				switch {
				case x == b.bounds.Min.X:
					p := b.cursor / attachmentsPerPage
					if p > 0 {
						b.cursor = (p - 1) * attachmentsPerPage
						b.RequestRender()
					}
				case x == b.bounds.Max.X-1:
					p := b.cursor / attachmentsPerPage
					if p < b.pageCount()-1 {
						b.cursor = (p + 1) * attachmentsPerPage
						if b.cursor >= len(b.attachments) {
							b.cursor = len(b.attachments) - 1
						}
						b.RequestRender()
					}
				default:
					cardW := (b.Width() - 8) / 3
					if cardW < 10 {
						cardW = 10
					}
					start := (b.cursor / attachmentsPerPage) * attachmentsPerPage
					for i := 0; i < attachmentsPerPage; i++ {
						idx := start + i
						if idx >= len(b.attachments) {
							break
						}
						cx := b.bounds.Min.X + 2 + i*(cardW+1)
						if x >= cx && x < cx+cardW {
							b.cursor = idx
							b.RequestRender()
							break
						}
					}
				}
			}
			return true
		}
	}
	return false
}
