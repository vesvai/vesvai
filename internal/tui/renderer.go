package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
)

type Cell struct {
	R rune
	S tcell.Style
}

type Line []Cell

type Segment struct {
	Text  string
	Style tcell.Style
}

func (s Segment) Width() int {
	gr := uniseg.NewGraphemes(s.Text)
	w := 0
	for gr.Next() {
		w += gr.Width()
	}
	return w
}

func (l Line) Width() int {
	w := 0
	for _, c := range l {
		w += cellWidth(c.R)
	}
	return w
}

func RuneWidth(r rune) int { return cellWidth(r) }

func DisplayWidth(text string) int {
	gr := uniseg.NewGraphemes(text)
	w := 0
	for gr.Next() {
		w += gr.Width()
	}
	return w
}

func FirstRuneAtWidth(line []rune, px int) rune {
	w := 0
	for _, r := range line {
		rw := cellWidth(r)
		if w+rw > px {
			return r
		}
		w += rw
	}
	if len(line) == 0 {
		return ' '
	}
	return line[len(line)-1]
}

func cellWidth(r rune) int {
	switch {
	case r == '\t':
		return 1
	case r >= 0x1F000:
		return 2
	case r >= 0x1100 && (r <= 0x115f || r == 0x2329 || r == 0x232a ||
		(r >= 0x2e80 && r <= 0xa4cf && r != 0x303f) ||
		(r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0xfe10 && r <= 0xfe19) ||
		(r >= 0xfe30 && r <= 0xfe6f) ||
		(r >= 0xff00 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6)):
		return 2
	default:
		return 1
	}
}

func graphemes(text string) []grapheme {
	var out []grapheme
	gr := uniseg.NewGraphemes(text)
	for gr.Next() {
		out = append(out, grapheme{text: gr.Str(), width: gr.Width()})
	}
	return out
}

type grapheme struct {
	text  string
	width int
}

func LineFromSegments(segs []Segment, width int) Line {
	var out Line
	w := 0
	for _, s := range segs {
		for _, g := range graphemes(s.Text) {
			if w+g.width > width {
				return out
			}
			for _, r := range g.text {
				out = append(out, Cell{R: r, S: s.Style})
			}
			w += g.width
		}
	}
	return out
}

func WrapText(text string, style tcell.Style, width int) []Line {
	if width <= 0 {
		return nil
	}
	return WrapSegments([]Segment{{Text: text, Style: style}}, width)
}

func WrapSegments(segs []Segment, width int) []Line {
	if width <= 0 {
		return nil
	}

	var lines []Line
	var cur Line
	curW := 0

	flush := func() {
		if len(cur) > 0 {
			lines = append(lines, cur)
			cur = nil
			curW = 0
		}
	}

	var pendingSpace *Cell
	pendingW := 0

	for _, s := range segs {
		for _, g := range graphemes(s.Text) {
			if g.text == "\n" {
				flush()
				pendingSpace, pendingW = nil, 0
				continue
			}
			cells := make([]Cell, 0, len(g.text))
			for _, r := range g.text {
				cells = append(cells, Cell{R: r, S: s.Style})
			}

			if g.text == " " {
				c := Cell{R: ' ', S: s.Style}
				pendingSpace = &c
				pendingW = g.width
				continue
			}

			needed := g.width
			if pendingSpace != nil {
				needed += pendingW
			}
			if curW+needed > width {
				flush()
				pendingSpace, pendingW = nil, 0
			}
			if pendingSpace != nil && len(cur) > 0 {
				cur = append(cur, *pendingSpace)
				curW += pendingW
				pendingSpace, pendingW = nil, 0
			}
			cur = append(cur, cells...)
			curW += g.width
		}
	}
	flush()
	return lines
}

func Truncate(l Line, width int) Line {
	if width <= 0 {
		return nil
	}
	if l.Width() <= width {
		return l
	}
	if width == 1 {
		return l[:1]
	}
	out := l[:0]
	w := 0
	for _, c := range l {
		cw := cellWidth(c.R)
		if w+cw > width-1 {
			break
		}
		out = append(out, c)
		w += cw
	}
	out = append(out, Cell{R: '…', S: l[len(l)-1].S})
	return out
}

func DrawLine(s tcell.Screen, x, y int, l Line) {
	sW, sH := s.Size()
	if y < 0 || y >= sH || x >= sW {
		return
	}
	px := x
	for _, c := range l {
		if px >= sW {
			return
		}
		s.SetContent(px, y, c.R, nil, c.S)
		px += cellWidth(c.R)
	}
}

func FillRect(s tcell.Screen, r imageRect, style tcell.Style) {
	for y := r.y0; y < r.y1; y++ {
		for x := r.x0; x < r.x1; x++ {
			s.SetContent(x, y, ' ', nil, style)
		}
	}
}

func FillLine(s tcell.Screen, y int, style tcell.Style) {
	sW, _ := s.Size()
	for x := 0; x < sW; x++ {
		s.SetContent(x, y, ' ', nil, style)
	}
}

type imageRect struct{ x0, y0, x1, y1 int }

func (r imageRect) width() int  { return r.x1 - r.x0 }
func (r imageRect) height() int { return r.y1 - r.y0 }
func (r imageRect) empty() bool { return r.width() <= 0 || r.height() <= 0 }

func ClampScroll(offset, max int) int {
	if offset < 0 {
		return 0
	}
	if offset > max {
		return max
	}
	return offset
}
