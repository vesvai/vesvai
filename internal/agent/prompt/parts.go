package prompt

import (
	"fmt"
	"strings"
)

type base struct {
	partID string
}

func (b base) id() string {
	return b.partID
}

type title struct {
	base
	text string
}

func Title(text string) Part {
	return &title{text: text}
}

func (t *title) renderMarkdown(rc *renderCtx) (string, error) {
	text, err := interpolate(rc, t.text)
	if err != nil {
		return "", err
	}
	return "# " + text, nil
}

func (t *title) renderXML(rc *renderCtx) (string, error) {
	text, err := interpolate(rc, t.text)
	if err != nil {
		return "", err
	}
	return "<title>" + escapeXML(text) + "</title>", nil
}

func (t *title) renderJSON(rc *renderCtx) (any, error) {
	text, err := interpolate(rc, t.text)
	if err != nil {
		return nil, err
	}
	return map[string]any{"type": "title", "text": text}, nil
}

type heading struct {
	base
	level int
	text  string
}

func Heading(level int, text string) Part {
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	return &heading{level: level, text: text}
}

func (h *heading) renderMarkdown(rc *renderCtx) (string, error) {
	text, err := interpolate(rc, h.text)
	if err != nil {
		return "", err
	}
	return strings.Repeat("#", h.level) + " " + text, nil
}

func (h *heading) renderXML(rc *renderCtx) (string, error) {
	text, err := interpolate(rc, h.text)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`<heading level="%d">%s</heading>`, h.level, escapeXML(text)), nil
}

func (h *heading) renderJSON(rc *renderCtx) (any, error) {
	text, err := interpolate(rc, h.text)
	if err != nil {
		return nil, err
	}
	return map[string]any{"type": "heading", "level": h.level, "text": text}, nil
}

type paragraph struct {
	base
	text string
}

func Paragraph(text string) Part {
	return &paragraph{text: text}
}

func (p *paragraph) renderMarkdown(rc *renderCtx) (string, error) {
	return interpolate(rc, p.text)
}

func (p *paragraph) renderXML(rc *renderCtx) (string, error) {
	text, err := interpolate(rc, p.text)
	if err != nil {
		return "", err
	}
	return "<paragraph>" + escapeXML(text) + "</paragraph>", nil
}

func (p *paragraph) renderJSON(rc *renderCtx) (any, error) {
	text, err := interpolate(rc, p.text)
	if err != nil {
		return nil, err
	}
	return map[string]any{"type": "paragraph", "text": text}, nil
}

type listEntry struct {
	text     string
	children []Part
}

type listItem struct {
	base
	text     string
	children []Part
}

func ListItem(text string, children ...Part) Part {
	return &listItem{text: text, children: children}
}

func (li *listItem) renderMarkdown(rc *renderCtx) (string, error) {
	text, err := interpolate(rc, li.text)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	writeListItem(&sb, strings.Repeat(" ", rc.indent), "", text)
	if len(li.children) > 0 {
		children, err := joinText(rc.withIndent(rc.indent+2), li.children, "\n")
		if err != nil {
			return "", err
		}
		sb.WriteString("\n" + children)
	}
	return sb.String(), nil
}

func (li *listItem) renderXML(rc *renderCtx) (string, error) {
	text, err := interpolate(rc, li.text)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("<item>" + escapeXML(text))
	if len(li.children) > 0 {
		inner, err := joinText(rc, li.children, "\n")
		if err != nil {
			return "", err
		}
		b.WriteString("\n" + indent(inner, 2) + "\n")
	}
	b.WriteString("</item>")
	return b.String(), nil
}

func (li *listItem) renderJSON(rc *renderCtx) (any, error) {
	text, err := interpolate(rc, li.text)
	if err != nil {
		return nil, err
	}
	nodes, err := jsonNodes(rc, li.children)
	if err != nil {
		return nil, err
	}
	return map[string]any{"text": text, "children": nodes}, nil
}

func normalizeEntries(items []any) []listEntry {
	entries := make([]listEntry, 0, len(items))
	for _, item := range items {
		switch v := item.(type) {
		case string:
			entries = append(entries, listEntry{text: v})
		case *listItem:
			entries = append(entries, listEntry{text: v.text, children: v.children})
		default:
			panic(fmt.Sprintf("prompt: List/OrderedList: item must be a string or ListItem, got %T", item))
		}
	}
	return entries
}

type list struct {
	base
	ordered bool
	items   []listEntry
}

func List(items ...any) Part {
	return &list{ordered: false, items: normalizeEntries(items)}
}

func OrderedList(items ...any) Part {
	return &list{ordered: true, items: normalizeEntries(items)}
}

func (l *list) renderMarkdown(rc *renderCtx) (string, error) {
	var sb strings.Builder
	pad := strings.Repeat(" ", rc.indent)
	for i, entry := range l.items {
		marker := "- "
		if l.ordered {
			marker = fmt.Sprintf("%d. ", i+1)
		}
		text, err := interpolate(rc, entry.text)
		if err != nil {
			return "", err
		}
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		writeListItem(&sb, pad, marker, text)
		if len(entry.children) > 0 {
			children, err := joinText(rc.withIndent(rc.indent+2), entry.children, "\n")
			if err != nil {
				return "", err
			}
			sb.WriteString("\n" + children)
		}
	}
	return sb.String(), nil
}

func writeListItem(sb *strings.Builder, pad, marker, text string) {
	contPad := pad + strings.Repeat(" ", len(marker))
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if i == 0 {
			sb.WriteString(pad + marker + line)
		} else {
			sb.WriteString("\n" + contPad + line)
		}
	}
}

func (l *list) renderXML(rc *renderCtx) (string, error) {
	var items []string
	for _, entry := range l.items {
		text, err := interpolate(rc, entry.text)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		b.WriteString("<item>" + escapeXML(text))
		if len(entry.children) > 0 {
			inner, err := joinText(rc, entry.children, "\n")
			if err != nil {
				return "", err
			}
			b.WriteString("\n" + indent(inner, 2) + "\n")
		}
		b.WriteString("</item>")
		items = append(items, b.String())
	}
	inner := strings.Join(items, "\n")
	ordered := "false"
	if l.ordered {
		ordered = "true"
	}
	return fmt.Sprintf(`<list ordered="%s">%s%s%s</list>`, ordered, "\n", indent(inner, 2), "\n"), nil
}

func (l *list) renderJSON(rc *renderCtx) (any, error) {
	var items []any
	for _, entry := range l.items {
		text, err := interpolate(rc, entry.text)
		if err != nil {
			return nil, err
		}
		if len(entry.children) == 0 {
			items = append(items, text)
			continue
		}
		nodes, err := jsonNodes(rc, entry.children)
		if err != nil {
			return nil, err
		}
		items = append(items, map[string]any{"text": text, "children": nodes})
	}
	return map[string]any{"type": "list", "ordered": l.ordered, "items": items}, nil
}

type codeBlock struct {
	base
	lang string
	code string
}

func Code(lang, code string) Part {
	return &codeBlock{lang: lang, code: code}
}

func (c *codeBlock) renderMarkdown(rc *renderCtx) (string, error) {
	code, err := interpolate(rc, c.code)
	if err != nil {
		return "", err
	}
	fence := fenceFor(code)
	lang := c.lang
	if lang == "" {
		lang = ""
	}
	return fence + lang + "\n" + code + "\n" + fence, nil
}

func (c *codeBlock) renderXML(rc *renderCtx) (string, error) {
	code, err := interpolate(rc, c.code)
	if err != nil {
		return "", err
	}
	attr := ""
	if c.lang != "" {
		attr = fmt.Sprintf(` lang="%s"`, escapeXML(c.lang))
	}
	return fmt.Sprintf("<code%s>%s</code>", attr, escapeXML(code)), nil
}

func (c *codeBlock) renderJSON(rc *renderCtx) (any, error) {
	code, err := interpolate(rc, c.code)
	if err != nil {
		return nil, err
	}
	return map[string]any{"type": "code", "lang": c.lang, "code": code}, nil
}

type kv struct {
	base
	key   string
	value any
}

func KV(key string, value any) Part {
	return &kv{key: key, value: value}
}

func (k *kv) renderMarkdown(rc *renderCtx) (string, error) {
	val, err := interpolate(rc, fmt.Sprint(k.value))
	if err != nil {
		return "", err
	}
	return "**" + k.key + "**: " + val, nil
}

func (k *kv) renderXML(rc *renderCtx) (string, error) {
	val, err := interpolate(rc, fmt.Sprint(k.value))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`<kv key="%s">%s</kv>`, escapeXML(k.key), escapeXML(val)), nil
}

func (k *kv) renderJSON(rc *renderCtx) (any, error) {
	val, err := interpolate(rc, fmt.Sprint(k.value))
	if err != nil {
		return nil, err
	}
	return map[string]any{"type": "kv", "key": k.key, "value": val}, nil
}

type table struct {
	base
	headers []string
	rows    [][]string
}

func Table(headers []string, rows [][]string) Part {
	return &table{headers: headers, rows: rows}
}

func (t *table) cols() int {
	n := len(t.headers)
	for _, row := range t.rows {
		if len(row) > n {
			n = len(row)
		}
	}
	return n
}

func (t *table) cell(row []string, i int) string {
	if i < len(row) {
		return row[i]
	}
	return ""
}

func (t *table) renderMarkdown(rc *renderCtx) (string, error) {
	headers := make([]string, 0, len(t.headers))
	for _, h := range t.headers {
		text, err := interpolate(rc, h)
		if err != nil {
			return "", err
		}
		headers = append(headers, mdCell(text))
	}
	cols := t.cols()
	for len(headers) < cols {
		headers = append(headers, "")
	}
	var sb strings.Builder
	sb.WriteString("| " + strings.Join(headers, " | ") + " |\n")
	seps := make([]string, cols)
	for i := range seps {
		seps[i] = "---"
	}
	sb.WriteString("| " + strings.Join(seps, " | ") + " |")
	for _, row := range t.rows {
		cells := make([]string, cols)
		for i := 0; i < cols; i++ {
			text, err := interpolate(rc, t.cell(row, i))
			if err != nil {
				return "", err
			}
			cells[i] = mdCell(text)
		}
		sb.WriteString("\n| " + strings.Join(cells, " | ") + " |")
	}
	return sb.String(), nil
}

func (t *table) renderXML(rc *renderCtx) (string, error) {
	var sb strings.Builder
	cols := t.cols()
	if len(t.headers) > 0 {
		var cells []string
		for _, h := range t.headers {
			text, err := interpolate(rc, h)
			if err != nil {
				return "", err
			}
			cells = append(cells, "<cell>"+escapeXML(text)+"</cell>")
		}
		sb.WriteString("<header>" + strings.Join(cells, "") + "</header>")
	}
	for _, row := range t.rows {
		cells := make([]string, cols)
		for i := 0; i < cols; i++ {
			text, err := interpolate(rc, t.cell(row, i))
			if err != nil {
				return "", err
			}
			cells[i] = "<cell>" + escapeXML(text) + "</cell>"
		}
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("<row>" + strings.Join(cells, "") + "</row>")
	}
	return fmt.Sprintf("<table>%s%s%s</table>", "\n", indent(sb.String(), 2), "\n"), nil
}

func (t *table) renderJSON(rc *renderCtx) (any, error) {
	headers := make([]string, 0, len(t.headers))
	for _, h := range t.headers {
		text, err := interpolate(rc, h)
		if err != nil {
			return nil, err
		}
		headers = append(headers, text)
	}
	cols := t.cols()
	rows := make([][]string, 0, len(t.rows))
	for _, row := range t.rows {
		cells := make([]string, cols)
		for i := 0; i < cols; i++ {
			text, err := interpolate(rc, t.cell(row, i))
			if err != nil {
				return nil, err
			}
			cells[i] = text
		}
		rows = append(rows, cells)
	}
	return map[string]any{"type": "table", "headers": headers, "rows": rows}, nil
}

type raw struct {
	base
	text string
}

func Raw(text string) Part {
	return &raw{text: text}
}

func (r *raw) renderMarkdown(_ *renderCtx) (string, error) { return r.text, nil }
func (r *raw) renderXML(_ *renderCtx) (string, error)      { return r.text, nil }
func (r *raw) renderJSON(_ *renderCtx) (any, error) {
	return map[string]any{"type": "raw", "text": r.text}, nil
}

type comment struct {
	base
	text string
}

func Comment(text string) Part {
	return &comment{text: text}
}

func (c *comment) renderMarkdown(rc *renderCtx) (string, error) {
	text, err := interpolate(rc, c.text)
	if err != nil {
		return "", err
	}
	return "<!-- " + text + " -->", nil
}

func (c *comment) renderXML(rc *renderCtx) (string, error) {
	text, err := interpolate(rc, c.text)
	if err != nil {
		return "", err
	}
	text = strings.ReplaceAll(text, "--", "- -")
	return "<!-- " + text + " -->", nil
}

func (c *comment) renderJSON(rc *renderCtx) (any, error) {
	text, err := interpolate(rc, c.text)
	if err != nil {
		return nil, err
	}
	return map[string]any{"type": "comment", "text": text}, nil
}

type xmlTag struct {
	base
	tag   string
	parts []Part
}

func XMLTag(tag string, parts ...Part) Part {
	return &xmlTag{tag: tag, parts: parts}
}

func (t *xmlTag) renderXMLDoc(rc *renderCtx) (string, error) {
	inner, err := joinText(rc.withFormat(FormatXML), t.parts, "\n")
	if err != nil {
		return "", err
	}
	return "<" + t.tag + ">\n" + indent(inner, 2) + "\n</" + t.tag + ">", nil
}

func (t *xmlTag) renderMarkdown(rc *renderCtx) (string, error) {
	return t.renderXMLDoc(rc)
}

func (t *xmlTag) renderXML(rc *renderCtx) (string, error) {
	return t.renderXMLDoc(rc)
}

func (t *xmlTag) renderJSON(rc *renderCtx) (any, error) {
	nodes, err := jsonNodes(rc, t.parts)
	if err != nil {
		return nil, err
	}
	return map[string]any{"type": "tag", "tag": t.tag, "parts": nodes}, nil
}

type group struct {
	base
	parts []Part
}

func Group(parts ...Part) Part {
	return &group{parts: parts}
}

func (g *group) renderMarkdown(rc *renderCtx) (string, error) {
	return joinText(rc, g.parts, "\n\n")
}

func (g *group) renderXML(rc *renderCtx) (string, error) {
	return joinText(rc, g.parts, "\n")
}

func (g *group) renderJSON(rc *renderCtx) (any, error) {
	nodes, err := jsonNodes(rc, g.parts)
	if err != nil {
		return nil, err
	}
	return map[string]any{"type": "group", "parts": nodes}, nil
}

type formatOverride struct {
	inner Part
	f     Format
}

func WithFormat(f Format, p Part) Part {
	if !f.valid() {
		panic(fmt.Sprintf("prompt: WithFormat: invalid format %v", f))
	}
	return &formatOverride{inner: p, f: f}
}

func (o *formatOverride) id() string { return o.inner.id() }

func (o *formatOverride) renderMarkdown(rc *renderCtx) (string, error) {
	return renderPart(rc.withFormat(o.f), o.inner)
}

func (o *formatOverride) renderXML(rc *renderCtx) (string, error) {
	return renderPart(rc.withFormat(o.f), o.inner)
}

func (o *formatOverride) renderJSON(rc *renderCtx) (any, error) {
	s, err := renderPart(rc.withFormat(o.f), o.inner)
	if err != nil {
		return nil, err
	}
	return map[string]any{"format": o.f.String(), "rendered": s}, nil
}

type idOverride struct {
	inner  Part
	partID string
}

func WithID(id string, p Part) Part {
	if id == "" {
		return p
	}
	return &idOverride{inner: p, partID: id}
}

func (o *idOverride) id() string { return o.partID }

func (o *idOverride) renderMarkdown(rc *renderCtx) (string, error) { return o.inner.renderMarkdown(rc) }
func (o *idOverride) renderXML(rc *renderCtx) (string, error)      { return o.inner.renderXML(rc) }
func (o *idOverride) renderJSON(rc *renderCtx) (any, error)        { return o.inner.renderJSON(rc) }

func (p *Prompt) Title(text string) *Prompt { return p.Add(Title(text)) }

func (p *Prompt) Heading(level int, text string) *Prompt {
	return p.Add(Heading(level, text))
}

func (p *Prompt) Paragraph(text string) *Prompt { return p.Add(Paragraph(text)) }

func (p *Prompt) List(items ...any) *Prompt { return p.Add(List(items...)) }

func (p *Prompt) OrderedList(items ...any) *Prompt { return p.Add(OrderedList(items...)) }

func (p *Prompt) Code(lang, code string) *Prompt { return p.Add(Code(lang, code)) }

func (p *Prompt) KV(key string, value any) *Prompt { return p.Add(KV(key, value)) }

func (p *Prompt) Table(headers []string, rows [][]string) *Prompt {
	return p.Add(Table(headers, rows))
}

func (p *Prompt) Skills(items []SkillInfo) *Prompt {
	return p.Add(SkillsList(items))
}

func (p *Prompt) Raw(text string) *Prompt { return p.Add(Raw(text)) }

func (p *Prompt) Comment(text string) *Prompt { return p.Add(Comment(text)) }

func (p *Prompt) XMLTag(tag string, parts ...Part) *Prompt {
	return p.Add(XMLTag(tag, parts...))
}

func (p *Prompt) Group(parts ...Part) *Prompt {
	return p.Add(Group(parts...))
}
