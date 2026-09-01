package components

import (
	"strings"
	"unicode/utf8"
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

func (l MdLine) Text() string {
	var b strings.Builder
	for _, s := range l.Segs {
		b.WriteString(s.Text)
	}
	return b.String()
}
