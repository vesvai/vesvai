package components

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	gast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

type MarkdownRenderer struct {
	md goldmark.Markdown
}

func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{
		md: goldmark.New(goldmark.WithExtensions(extension.GFM)),
	}
}

type mdContext struct {
	width int
	pal   *tui.Palette
	lines []tui.Line
	blank bool
}

func (c *mdContext) pushBlank() { c.blank = true }

func (c *mdContext) flushBlank() {
	if c.blank {
		c.lines = append(c.lines, nil)
		c.blank = false
	}
}

func (r *MarkdownRenderer) Render(src string, width int, pal *tui.Palette) []tui.Line {
	if width <= 0 {
		return nil
	}
	source := []byte(src)
	doc := r.md.Parser().Parse(text.NewReader(source))
	ctx := &mdContext{width: width, pal: pal}
	r.renderBlocks(doc, ctx, source)
	return ctx.lines
}

func (r *MarkdownRenderer) renderBlocks(node ast.Node, ctx *mdContext, src []byte) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		r.renderBlock(child, ctx, src)
	}
}

func (r *MarkdownRenderer) renderBlock(n ast.Node, ctx *mdContext, src []byte) {
	switch b := n.(type) {
	case *ast.Paragraph, *ast.TextBlock:
		ctx.flushBlank()
		ctx.lines = append(ctx.lines, tui.WrapSegments(r.renderInline(n, ctx, src), ctx.width)...)
		ctx.pushBlank()

	case *ast.Heading:
		ctx.flushBlank()
		segs := r.renderInline(n, ctx, src)
		style := ctx.pal.TextStyle().Bold(true)
		switch b.Level {
		case 1:
			style = style.Foreground(ctx.pal.Accent)
		case 2:
			style = style.Foreground(ctx.pal.AssistantLabel)
		default:
			style = style.Foreground(ctx.pal.Foreground)
		}
		for i := range segs {
			segs[i].Style = style
		}
		ctx.pushPrefixed(segs, "▍ ", ctx.pal.Style(ctx.pal.AccentDim, ctx.pal.Background))
		ctx.pushBlank()

	case *ast.List:
		r.renderList(b, ctx, src)

	case *ast.Blockquote:
		ctx.flushBlank()
		sub := &mdContext{width: ctx.width - 2, pal: ctx.pal}
		r.renderBlocks(b, sub, src)
		dim := ctx.pal.Style(ctx.pal.TextDim, ctx.pal.Background)
		for _, ln := range sub.lines {
			prefix := tui.Line{{R: '│', S: dim}, {R: ' ', S: dim}}
			ctx.lines = append(ctx.lines, append(prefix, ln...))
		}
		ctx.pushBlank()

	case *ast.CodeBlock, *ast.FencedCodeBlock:
		ctx.flushBlank()
		var lang string
		if l, ok := n.(interface{ Language(src []byte) []byte }); ok {
			lang = string(l.Language(src))
		}
		var code []string
		if ln, ok := n.(interface{ Lines() *text.Segments }); ok {
			for i := 0; i < ln.Lines().Len(); i++ {
				seg := ln.Lines().At(i)
				code = append(code, strings.TrimSuffix(string(seg.Value(src)), "\n"))
			}
		}
		ctx.pushCodeBlock(lang, code)
		ctx.pushBlank()

	case *ast.ThematicBreak:
		ctx.flushBlank()
		ctx.lines = append(ctx.lines, tui.LineFromSegments([]tui.Segment{
			{Text: strings.Repeat("─", ctx.width), Style: ctx.pal.Style(ctx.pal.Muted, ctx.pal.Background)},
		}, ctx.width))
		ctx.pushBlank()

	case *gast.Table:
		r.renderTable(b, ctx, src)

	case *ast.HTMLBlock:

	default:
		r.renderBlocks(n, ctx, src)
	}
}

func (c *mdContext) pushPrefixed(segs []tui.Segment, prefix string, prefixStyle tcell.Style) {
	pw := 0
	for _, r := range prefix {
		pw += tui.RuneWidth(r)
	}
	lines := tui.WrapSegments(segs, c.width-pw)
	for i, ln := range lines {
		if i == 0 {
			marker := tui.LineFromSegments([]tui.Segment{{Text: prefix, Style: prefixStyle}}, pw)
			ln = append(marker, ln...)
		}
		c.lines = append(c.lines, ln)
	}
	if len(lines) == 0 {
		c.lines = append(c.lines, tui.LineFromSegments([]tui.Segment{{Text: prefix, Style: prefixStyle}}, c.width))
	}
}

func (r *MarkdownRenderer) renderList(b *ast.List, ctx *mdContext, src []byte) {
	ctx.flushBlank()
	markerW := 2
	ordered := b.IsOrdered()
	if ordered {
		markerW = 4
	}
	item := b.Start
	if item == 0 {
		item = 1
	}
	for li := b.FirstChild(); li != nil; li = li.NextSibling() {
		sub := &mdContext{width: ctx.width - markerW, pal: ctx.pal}
		r.renderBlocks(li, sub, src)
		lines := sub.lines
		if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
			lines = lines[:len(lines)-1]
		}
		marker := "• "
		if ordered {
			marker = fmt.Sprintf("%d. ", item)
			item++
		}
		style := ctx.pal.TextStyle()
		for i, ln := range lines {
			if i == 0 {
				markerLine := tui.LineFromSegments([]tui.Segment{{Text: marker, Style: style.Foreground(ctx.pal.Accent)}}, markerW)
				ctx.lines = append(ctx.lines, append(markerLine, ln...))
			} else {
				indent := tui.LineFromSegments([]tui.Segment{{Text: strings.Repeat(" ", markerW), Style: style}}, markerW)
				ctx.lines = append(ctx.lines, append(indent, ln...))
			}
		}
		ctx.lines = append(ctx.lines, nil)
	}
	ctx.pushBlank()
}

type tableRow []string

func (r *MarkdownRenderer) renderTable(b *gast.Table, ctx *mdContext, src []byte) {
	ctx.flushBlank()
	var rows []tableRow
	var header tableRow
	var colCount int

	collect := func(n ast.Node) {
		var cells tableRow
		for cell := n.FirstChild(); cell != nil; cell = cell.NextSibling() {
			tc, ok := cell.(*gast.TableCell)
			if !ok {
				continue
			}
			var sb strings.Builder
			for _, seg := range r.renderInline(tc, ctx, src) {
				sb.WriteString(seg.Text)
			}
			cells = append(cells, sb.String())
		}
		if len(cells) > colCount {
			colCount = len(cells)
		}
		rows = append(rows, cells)
		if _, isHeader := n.(*gast.TableHeader); isHeader {
			header = cells
		}
	}
	for child := b.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.(type) {
		case *gast.TableHeader:
			collect(child)
		case *gast.TableRow:
			collect(child)
		default:
			for c := child.FirstChild(); c != nil; c = c.NextSibling() {
				collect(c)
			}
		}
	}
	if colCount == 0 {
		return
	}

	widths := make([]int, colCount)
	total := 0
	for _, rw := range rows {
		for i, c := range rw {
			w := len(c)
			if w > widths[i] {
				widths[i] = w
			}
		}
	}
	avail := ctx.width - (colCount*3 + 1)
	if avail < colCount {
		avail = colCount
	}
	for i := range widths {
		if widths[i] > avail/colCount {
			widths[i] = avail / colCount
		}
		total += widths[i]
	}

	renderRow := func(cells tableRow, style tcell.Style) {
		segs := []tui.Segment{{Text: "│", Style: style}}
		for i := 0; i < colCount; i++ {
			cell := ""
			if i < len(cells) {
				cell = cells[i]
			}
			cw := widths[i]
			if dw := tui.DisplayWidth(cell); dw > cw {
				cell = truncateCell(cell, cw)
			}
			pad := cw - tui.DisplayWidth(cell)
			if pad < 0 {
				pad = 0
			}
			segs = append(segs,
				tui.Segment{Text: " " + cell, Style: style},
				tui.Segment{Text: strings.Repeat(" ", pad), Style: style},
				tui.Segment{Text: " │", Style: style},
			)
		}
		ctx.lines = append(ctx.lines, tui.LineFromSegments(segs, ctx.width))
	}

	border := ctx.pal.Style(ctx.pal.Muted, ctx.pal.Background)
	renderRow(header, border.Foreground(ctx.pal.Foreground))

	sep := []tui.Segment{{Text: "│", Style: border}}
	for i := 0; i < colCount; i++ {
		sep = append(sep, tui.Segment{Text: "─" + strings.Repeat("─", widths[i]) + "─│", Style: border})
	}
	ctx.lines = append(ctx.lines, tui.LineFromSegments(sep, ctx.width))

	body := border.Foreground(ctx.pal.TextDim)
	for _, rw := range rows {
		if len(rw) == 0 {
			continue
		}
		if header != nil && rowEq(rw, header) {
			continue
		}
		renderRow(rw, body)
	}
	ctx.pushBlank()
}

func rowEq(a, b tableRow) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func truncateCell(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	runes := []rune(s)
	w := 0
	keep := 0
	for _, r := range runes {
		rw := tui.RuneWidth(r)
		if w+rw > maxWidth {
			break
		}
		w += rw
		keep++
	}
	if keep == len(runes) {
		return s
	}
	if keep == 0 {
		return ""
	}
	last := runes[keep-1]
	if w-tui.RuneWidth(last)+tui.RuneWidth('…') <= maxWidth {
		return string(runes[:keep-1]) + "…"
	}
	return string(runes[:keep])
}

func (r *MarkdownRenderer) renderInline(n ast.Node, ctx *mdContext, src []byte) []tui.Segment {
	base := ctx.pal.TextStyle()
	var segs []tui.Segment
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch c := child.(type) {
		case *ast.Text:
			s := string(c.Segment.Value(src))
			if c.HardLineBreak() {
				s += "\n"
			}
			segs = append(segs, tui.Segment{Text: s, Style: base})
		case *ast.String:
			segs = append(segs, tui.Segment{Text: string(c.Value), Style: base})
		case *ast.CodeSpan:
			style := ctx.pal.Style(ctx.pal.TokenString, ctx.pal.Surface)
			segs = append(segs, tui.Segment{Text: string(c.Text(src)), Style: style})
		case *ast.Emphasis:
			inner := r.renderInline(c, ctx, src)
			for i := range inner {
				if c.Level >= 2 {
					inner[i].Style = inner[i].Style.Bold(true)
				} else {
					inner[i].Style = inner[i].Style.Dim(true)
				}
			}
			segs = append(segs, inner...)
		case *gast.Strikethrough:
			inner := r.renderInline(c, ctx, src)
			for i := range inner {
				inner[i].Style = inner[i].Style.StrikeThrough(true)
			}
			segs = append(segs, inner...)
		case *ast.Link:
			inner := r.renderInline(c, ctx, src)
			for i := range inner {
				inner[i].Style = inner[i].Style.Foreground(ctx.pal.Accent).Underline(true)
			}
			segs = append(segs, inner...)
		case *ast.AutoLink:
			style := base.Foreground(ctx.pal.Accent).Underline(true)
			segs = append(segs, tui.Segment{Text: string(c.URL(src)), Style: style})
		case *ast.Image:
			inner := r.renderInline(c, ctx, src)
			for i := range inner {
				inner[i].Style = inner[i].Style.Foreground(ctx.pal.TextDim)
			}
			segs = append(segs, inner...)
		default:
			if c.FirstChild() != nil {
				segs = append(segs, r.renderInline(c, ctx, src)...)
			}
		}
	}
	return segs
}

func (c *mdContext) pushCodeBlock(lang string, code []string) {
	innerW := c.width - 4
	if innerW < 1 {
		innerW = 1
	}
	border := c.pal.Style(c.pal.CodeBorder, c.pal.CodeBg)
	content := c.pal.Style(c.pal.CodeText, c.pal.CodeBg)

	label := lang
	if label == "" {
		label = "code"
	}
	title := tui.Line{
		{R: '┌', S: border},
		{R: '─', S: border},
	}
	for _, r := range label {
		title = append(title, tui.Cell{R: r, S: border.Foreground(c.pal.Accent)})
	}
	for len(title) < innerW+2 {
		title = append(title, tui.Cell{R: '─', S: border})
	}
	title = append(title, tui.Cell{R: '┐', S: border})
	c.lines = append(c.lines, title)

	var body [][]tui.Cell
	plain := func(lines []string) [][]tui.Cell {
		out := make([][]tui.Cell, len(lines))
		for i, ln := range lines {
			runes := []rune(ln)
			cells := make([]tui.Cell, 0, len(runes))
			for _, r := range runes {
				cells = append(cells, tui.Cell{R: r, S: content})
			}
			out[i] = cells
		}
		return out
	}

	if lang != "" && lexers.Get(lang) != nil {
		lexer := chroma.Coalesce(lexers.Get(lang))
		if it, err := lexer.Tokenise(nil, strings.Join(code, "\n")); err == nil {
			body = chromaToCells(it.Tokens(), c.pal, content)
		} else {
			body = plain(code)
		}
	} else {
		body = plain(code)
	}

	prefix := tui.Cell{R: '│', S: border}
	pad := tui.Cell{R: ' ', S: content}
	for i := 0; i < len(body); i++ {
		row := append([]tui.Cell{prefix, pad}, body[i]...)
		if len(row) > innerW+2 {
			row = row[:innerW+2]
		}
		for len(row) < innerW+2 {
			row = append(row, tui.Cell{R: ' ', S: content})
		}
		c.lines = append(c.lines, row)
	}

	footer := tui.Line{{R: '└', S: border}}
	for len(footer) < innerW+1 {
		footer = append(footer, tui.Cell{R: '─', S: border})
	}
	footer = append(footer, tui.Cell{R: '┘', S: border})
	c.lines = append(c.lines, footer)
}

func chromaToCells(tokens []chroma.Token, pal *tui.Palette, base tcell.Style) [][]tui.Cell {
	var lines [][]tui.Cell
	var cur []tui.Cell
	for _, tok := range tokens {
		style := tokenStyle(tok.Type, pal, base)
		parts := strings.Split(tok.Value, "\n")
		for i, part := range parts {
			if i > 0 {
				lines = append(lines, cur)
				cur = nil
			}
			for _, r := range part {
				cur = append(cur, tui.Cell{R: r, S: style})
			}
		}
	}
	if len(cur) > 0 {
		lines = append(lines, cur)
	}
	return lines
}

func tokenStyle(t chroma.TokenType, pal *tui.Palette, base tcell.Style) tcell.Style {
	c := pal.TokenName
	switch t.Category() {
	case chroma.Keyword:
		c = pal.TokenKeyword
	case chroma.Name:
		switch t.SubCategory() {
		case chroma.NameFunction:
			c = pal.TokenFunction
		case chroma.NameClass, chroma.NameTag, chroma.NameBuiltin:
			c = pal.TokenType
		case chroma.NameConstant:
			c = pal.TokenConstant
		default:
			c = pal.TokenName
		}
	case chroma.LiteralString:
		c = pal.TokenString
	case chroma.LiteralNumber:
		c = pal.TokenNumber
	case chroma.Comment:
		return base.Foreground(pal.TokenComment).Dim(true)
	case chroma.Operator:
		c = pal.TokenOperator
	case chroma.Punctuation:
		c = pal.TokenPunctuation
	case chroma.Generic:
		c = pal.TokenGeneric
	}
	return base.Foreground(c)
}
