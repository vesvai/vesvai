package components

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

type Mention struct {
	ID    string
	Kind  string
	Label string
}

func DefaultMentions() []Mention {
	return []Mention{
		{ID: "orchestrator", Kind: "agent", Label: "orchestrator"},
		{ID: "planner", Kind: "agent", Label: "planner"},
		{ID: "explorer", Kind: "agent", Label: "explorer"},
	}
}

type blockToken struct {
	start, end int
	kind       rune
}

func isMentionChar(r rune) bool {
	return r == '/' || r == '.' || r == '-' || r == '_' ||
		(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func isSkillChar(r rune) bool {
	return r == '-' || r == '_' ||
		(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func (t *Textarea) blockTokens(line []rune) []blockToken {
	var out []blockToken
	i := 0
	for i < len(line) {
		switch line[i] {
		case '@':
			j := i + 1
			for j < len(line) && isMentionChar(line[j]) {
				j++
			}
			if j > i+1 {
				out = append(out, blockToken{start: i, end: j, kind: '@'})
			}
			i = j
		case '/':
			j := i + 1
			for j < len(line) && isSkillChar(line[j]) {
				j++
			}
			if j > i+1 && t.skillNames[string(line[i+1:j])] {
				out = append(out, blockToken{start: i, end: j, kind: '/'})
			}
			i = j
		default:
			i++
		}
	}
	return out
}

func tokenAt(tokens []blockToken, col int) *blockToken {
	for i := range tokens {
		if col >= tokens[i].start && col < tokens[i].end {
			return &tokens[i]
		}
	}
	return nil
}

func tokenEndingAt(tokens []blockToken, col int) *blockToken {
	for i := range tokens {
		if col == tokens[i].end {
			return &tokens[i]
		}
	}
	return nil
}

const mentionPickerHeight = 8

func filterMentions(catalog []Mention, q string) []Mention {
	if q == "" {
		return catalog
	}
	ql := strings.ToLower(q)
	var out []Mention
	for _, m := range catalog {
		if strings.Contains(strings.ToLower(m.ID+" "+m.Label), ql) {
			out = append(out, m)
		}
	}
	return out
}

func (t *Textarea) drawMentionPicker(s tcell.Screen, pal *tui.Palette) {
	if len(t.pickerResults) == 0 {
		return
	}
	w := t.Width() - 2
	if w > 46 {
		w = 46
	}
	if w < 12 {
		w = 12
	}
	h := len(t.pickerResults) + 2
	if h > mentionPickerHeight {
		h = mentionPickerHeight
	}

	x0 := t.bounds.Min.X + 1
	y0 := t.bounds.Min.Y - h
	if y0 < 0 {
		y0 = 0
	}
	x1 := x0 + w
	if x1 > t.bounds.Max.X {
		x0 = t.bounds.Max.X - w
		x1 = x0 + w
	}
	t.pickerX0, t.pickerY0, t.pickerW, t.pickerH = x0, y0, w, h

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
		s.SetContent(x1-1, y0+y, '│', nil, border)
		for x := x0 + 1; x < x1-1; x++ {
			s.SetContent(x, y0+y, ' ', nil, bg)
		}
	}
	bottom := tui.Line{{R: '└', S: border}}
	for len(bottom) < w-1 {
		bottom = append(bottom, tui.Cell{R: '─', S: border})
	}
	bottom = append(bottom, tui.Cell{R: '┘', S: border})
	tui.DrawLine(s, x0, y0+h-1, bottom)

	query := tui.Line{
		{R: '❯', S: pal.Style(pal.Accent, pal.Surface).Bold(true)},
		{R: ' ', S: bg},
		{R: t.pickerKind, S: pal.Style(pal.Accent, pal.Surface).Bold(true)},
	}
	for _, r := range t.mentionQuery() {
		query = append(query, tui.Cell{R: r, S: bg})
	}
	tui.DrawLine(s, x0+1, y0+1, query)

	if t.pickerCursor < t.pickerOffset {
		t.pickerOffset = t.pickerCursor
	}
	if t.pickerCursor >= t.pickerOffset+h-3 {
		t.pickerOffset = t.pickerCursor - h + 4
	}
	if t.pickerOffset < 0 {
		t.pickerOffset = 0
	}

	selStyle := pal.Style(pal.Foreground, pal.Selection)
	for row := 0; row < h-2; row++ {
		ei := t.pickerOffset + row
		if ei >= len(t.pickerResults) {
			break
		}
		m := t.pickerResults[ei]
		selected := ei == t.pickerCursor
		style := bg
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

		var glyph rune
		var glyphColor tcell.Color
		switch m.Kind {
		case "agent":
			glyph, glyphColor = '✦', pal.Mention
		case "dir":
			glyph, glyphColor = '▸', pal.Accent
		default:
			glyph, glyphColor = '·', pal.TextDim
		}
		line = append(line, tui.Cell{R: glyph, S: style.Foreground(glyphColor)})
		line = append(line, tui.Cell{R: ' ', S: style})
		labelStyle := style
		if selected {
			labelStyle = style.Foreground(pal.Accent)
		}
		for _, r := range m.Label {
			line = append(line, tui.Cell{R: r, S: labelStyle})
		}
		if len(line) > w-2 {
			line = line[:w-2]
		}
		for len(line) < w-2 {
			line = append(line, tui.Cell{R: ' ', S: style})
		}
		tui.DrawLine(s, x0+1, y0+2+row, line)
	}
}
