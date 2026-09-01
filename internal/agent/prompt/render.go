package prompt

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type renderCtx struct {
	format Format
	vars   Vars
	indent int
}

func (rc *renderCtx) withFormat(f Format) *renderCtx {
	c := *rc
	c.format = f
	return &c
}

func (rc *renderCtx) withIndent(n int) *renderCtx {
	c := *rc
	c.indent = n
	return &c
}

func renderPart(rc *renderCtx, part Part) (string, error) {
	switch rc.format {
	case FormatMarkdown:
		return part.renderMarkdown(rc)
	case FormatXML:
		return part.renderXML(rc)
	case FormatJSON:
		v, err := part.renderJSON(rc)
		if err != nil {
			return "", err
		}
		b, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("prompt: encode json part: %w", err)
		}
		return string(b), nil
	}
	return "", fmt.Errorf("prompt: unsupported format %d", rc.format)
}

func joinText(rc *renderCtx, parts []Part, sep string) (string, error) {
	var sb strings.Builder
	for _, part := range parts {
		s, err := renderPart(rc, part)
		if err != nil {
			return "", err
		}
		if s == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(s)
	}
	return sb.String(), nil
}

func jsonNodes(rc *renderCtx, parts []Part) ([]any, error) {
	nodes := make([]any, 0, len(parts))
	for _, part := range parts {
		v, err := part.renderJSON(rc)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, v)
	}
	return nodes, nil
}

var interpRe = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_.]*)\s*\}\}`)

func interpolate(rc *renderCtx, s string) (string, error) {
	if !strings.Contains(s, "{{") {
		return s, nil
	}
	var err error
	out := interpRe.ReplaceAllStringFunc(s, func(m string) string {
		if err != nil {
			return m
		}
		key := interpRe.FindStringSubmatch(m)[1]
		val, ok := rc.vars.Get(key)
		if !ok {
			err = fmt.Errorf("%w: %q", ErrMissingVar, key)
			return m
		}
		return fmt.Sprint(val)
	})
	if err != nil {
		return "", err
	}
	return out, nil
}

func escapeXML(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&apos;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func indent(s string, n int) string {
	if s == "" {
		return ""
	}
	pad := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = pad + line
	}
	return strings.Join(lines, "\n")
}

func mdCell(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "|", `\|`)
	s = strings.ReplaceAll(s, "\n", "<br>")
	return s
}

func fenceFor(code string) string {
	n := 3
	for strings.Contains(code, strings.Repeat("`", n)) {
		n++
	}
	return strings.Repeat("`", n)
}
