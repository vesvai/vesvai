package components

import (
	"strings"
	"unicode/utf8"

	"github.com/vesvai/vesvai/internal/tui/styles"
)

type MdSeg struct {
	Text   string
	Bold   bool
	Italic bool
	Code   bool
}

type MdLine struct {
	Segs    []MdSeg
	Heading int
	Quote   bool
	Bullet  bool
	Hr      bool
	Code    bool
	Lang    string
}

func RenderMarkdown(src string) []MdLine {
	var out []MdLine
	lines := strings.Split(src, "\n")

	inCode := false
	var codeLang string
	for _, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if strings.HasPrefix(trimmed, "```") {
			if !inCode {
				inCode = true
				codeLang = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
				out = append(out, MdLine{Code: true, Lang: codeLang, Segs: []MdSeg{{Text: "``` " + codeLang}}})
			} else {
				inCode = false
				codeLang = ""
				out = append(out, MdLine{Code: true, Segs: []MdSeg{{Text: "```"}}})
			}
			continue
		}
		if inCode {
			out = append(out, MdLine{Code: true, Segs: []MdSeg{{Text: raw}}})
			continue
		}
		out = append(out, renderParagraphLine(raw))
	}
	return out
}

func renderParagraphLine(raw string) MdLine {
	line := MdLine{}
	rest := raw

	if n := headingLevel(rest); n > 0 {
		line.Heading = n
		rest = strings.TrimSpace(rest[n:])
	} else if strings.TrimSpace(rest) == "---" || strings.TrimSpace(rest) == "***" || strings.TrimSpace(rest) == "___" {
		line.Hr = true
		return line
	} else if strings.HasPrefix(rest, ">") {
		line.Quote = true
		rest = strings.TrimSpace(rest[1:])
	} else if isBullet(rest) {
		line.Bullet = true
		rest = strings.TrimSpace(rest[1:])
	}

	line.Segs = parseInline(rest)
	if len(line.Segs) == 0 {
		line.Segs = []MdSeg{{Text: rest}}
	}
	return line
}

func headingLevel(s string) int {
	if len(s) == 0 || s[0] != '#' {
		return 0
	}
	n := 0
	for n < len(s) && n < 6 && s[n] == '#' {
		n++
	}
	if n < len(s) && s[n] != ' ' {
		return 0
	}
	return n
}

func isBullet(s string) bool {
	if len(s) == 0 {
		return false
	}
	switch s[0] {
	case '-', '*', '+':
		return true
	}
	for i := 0; i < len(s) && i < 3; i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
		if i+1 < len(s) && s[i+1] == '.' {
			return true
		}
	}
	return false
}

func parseInline(s string) []MdSeg {
	var segs []MdSeg
	var cur MdSeg
	flush := func() {
		if cur.Text != "" {
			segs = append(segs, cur)
			cur = MdSeg{}
		}
	}
	i := 0
	for i < len(s) {
		switch {
		case strings.HasPrefix(s[i:], "**"):
			flush()
			cur.Bold = !cur.Bold
			i += 2
		case strings.HasPrefix(s[i:], "*"):
			flush()
			cur.Italic = !cur.Italic
			i++
		case strings.HasPrefix(s[i:], "`"):
			flush()
			cur.Code = !cur.Code
			i++
		case strings.HasPrefix(s[i:], "~~"):
			flush()
			cur.Italic = !cur.Italic
			i += 2
		default:
			r, size := utf8.DecodeRuneInString(s[i:])
			cur.Text += string(r)
			i += size
		}
	}
	flush()
	if len(segs) == 0 {
		segs = append(segs, MdSeg{Text: s})
	}
	return segs
}

func wrapSegs(segs []MdSeg, width int) [][]MdSeg {
	if width < 1 {
		width = 1
	}
	var out [][]MdSeg
	var cur []MdSeg
	curLen := 0
	flush := func() {
		if len(cur) > 0 {
			out = append(out, cur)
			cur = nil
			curLen = 0
		}
	}
	for _, seg := range segs {
		words := strings.Fields(seg.Text)
		if len(words) == 0 {
			continue
		}
		for _, w := range words {
			wl := len(w)
			if curLen > 0 && curLen+1+wl > width {
				flush()
			}
			if wl > width {
				for len(w) > width {
					cur = append(cur, MdSeg{Text: w[:width], Bold: seg.Bold, Italic: seg.Italic, Code: seg.Code})
					curLen = width
					flush()
					w = w[width:]
				}
				wl = len(w)
			}
			sp := ""
			if curLen > 0 {
				sp = " "
			}
			cur = append(cur, MdSeg{Text: sp + w, Bold: seg.Bold, Italic: seg.Italic, Code: seg.Code})
			curLen += len(sp) + wl
		}
	}
	flush()
	if len(out) == 0 {
		out = append(out, []MdSeg{{Text: ""}})
	}
	return out
}

func (l MdLine) Text() string {
	var b strings.Builder
	for _, s := range l.Segs {
		b.WriteString(s.Text)
	}
	return b.String()
}

func MdToLines(src string, width int, th styles.Theme) []Line {
	md := RenderMarkdown(src)
	var out []Line
	for _, ln := range md {
		if ln.Code {
			style := th.Base().Foreground(th.Hint).Background(th.CodeBg)
			row := LineFromSegments([]Segment{{Text: ln.Text(), Style: style}}, width)
			out = append(out, row)
			continue
		}
		if ln.Hr {
			style := th.Base().Foreground(th.Border).Background(th.Background)
			row := LineFromSegments([]Segment{{Text: strings.Repeat("─", width), Style: style}}, width)
			out = append(out, row)
			continue
		}
		for _, segs := range wrapSegs(ln.Segs, width) {
			var segList []Segment
			baseStyle := th.Base().Background(th.Background)
			if ln.Heading > 0 {
				baseStyle = baseStyle.Foreground(th.Accent).Bold(true)
			} else if ln.Quote {
				baseStyle = baseStyle.Foreground(th.Hint)
			} else if ln.Bullet {
				baseStyle = baseStyle.Foreground(th.Foreground)
			}
			prefix := ""
			if ln.Quote {
				prefix = "│ "
			} else if ln.Bullet {
				prefix = "• "
			}
			if prefix != "" {
				segList = append(segList, Segment{Text: prefix, Style: baseStyle})
			}
			if ln.Heading > 0 {
				segList = append(segList, Segment{Text: "▍ ", Style: th.Base().Foreground(th.AccentDim).Background(th.Background)})
			}
			for _, s := range segs {
				segStyle := baseStyle
				if s.Bold {
					segStyle = segStyle.Bold(true)
				}
				if s.Italic {
					segStyle = segStyle.Italic(true)
				}
				if s.Code {
					segStyle = th.Base().Foreground(th.Accent).Background(th.InputBg)
				}
				segList = append(segList, Segment{Text: s.Text, Style: segStyle})
			}
			out = append(out, LineFromSegments(segList, width))
		}
	}
	return out
}
