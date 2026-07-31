package components

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type MarkdownRenderer struct {
	md    goldmark.Markdown
	theme string
}

func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{
		md:    goldmark.New(),
		theme: "monokai",
	}
}

type StyledLine struct {
	Segments    []render.StyledSegment
	PreRendered bool
}

func (mr *MarkdownRenderer) Render(markdown string) []StyledLine {
	source := []byte(markdown)
	reader := text.NewReader(source)
	doc := mr.md.Parser().Parse(reader)

	var lines []StyledLine
	mr.renderNode(doc, source, &lines, 0)

	if len(lines) == 0 {
		lines = append(lines, StyledLine{
			Segments: []render.StyledSegment{
				{Text: markdown, Style: theme.MarkdownParagraph},
			},
		})
	}

	return lines
}

func (mr *MarkdownRenderer) renderNode(n ast.Node, source []byte, lines *[]StyledLine, depth int) {
	switch n.Kind() {
	case ast.KindDocument:
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			mr.renderNode(child, source, lines, depth)
		}

	case ast.KindHeading:
		heading := n.(*ast.Heading)
		textContent := mr.extractText(n, source)
		style := mr.headingStyle(heading.Level)
		*lines = append(*lines, StyledLine{
			Segments: []render.StyledSegment{
				{Text: textContent, Style: style},
			},
		})
		*lines = append(*lines, StyledLine{})

	case ast.KindParagraph:
		textContent := mr.extractText(n, source)
		wrapped := render.WrapText(textContent, 80)
		for _, line := range wrapped {
			*lines = append(*lines, StyledLine{
				Segments: []render.StyledSegment{
					{Text: line, Style: theme.MarkdownParagraph},
				},
			})
		}
		*lines = append(*lines, StyledLine{})

	case ast.KindFencedCodeBlock:
		codeBlock := n.(*ast.FencedCodeBlock)
		language := ""
		if codeBlock.Info != nil {
			language = string(codeBlock.Info.Text(source))
		}
		mr.renderCodeBlock(codeBlock, source, language, lines)

	case ast.KindCodeBlock:
		mr.renderIndentedCodeBlock(n.(*ast.CodeBlock), source, lines)

	case ast.KindBlockquote:
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			textContent := mr.extractText(child, source)
			wrapped := render.WrapText(textContent, 76)
			for _, line := range wrapped {
				*lines = append(*lines, StyledLine{
					Segments: []render.StyledSegment{
						{Text: "  " + theme.Pipe + " ", Style: theme.MarkdownBlockquote},
						{Text: line, Style: theme.MarkdownBlockquote},
					},
				})
			}
		}
		*lines = append(*lines, StyledLine{})

	case ast.KindList:
		mr.renderList(n.(*ast.List), source, lines, depth)

	case ast.KindListItem:
		mr.renderListItem(n.(*ast.ListItem), source, lines, depth)

	case ast.KindThematicBreak:
		*lines = append(*lines, StyledLine{
			Segments: []render.StyledSegment{
				{Text: strings.Repeat(theme.Dash, 60), Style: theme.MarkdownHR},
			},
		})
		*lines = append(*lines, StyledLine{})

	case ast.KindHTMLBlock:
		textContent := mr.extractText(n, source)
		*lines = append(*lines, StyledLine{
			Segments: []render.StyledSegment{
				{Text: textContent, Style: theme.NewStyle().WithForeground(theme.TextDim)},
			},
		})

	default:
		if n.HasChildren() {
			for child := n.FirstChild(); child != nil; child = child.NextSibling() {
				mr.renderNode(child, source, lines, depth)
			}
		}
	}
}

func (mr *MarkdownRenderer) renderCodeBlock(codeBlock *ast.FencedCodeBlock, source []byte, language string, lines *[]StyledLine) {
	rawText := string(codeBlock.Text(source))
	codeLines := strings.Split(rawText, "\n")
	if len(codeLines) > 0 && codeLines[len(codeLines)-1] == "" {
		codeLines = codeLines[:len(codeLines)-1]
	}

	code := strings.Join(codeLines, "\n")

	header := "  " + language + " "
	if language == "" {
		header = "  code "
	}
	*lines = append(*lines, StyledLine{
		Segments: []render.StyledSegment{
			{Text: header, Style: theme.ShortcutKeyStyle},
		},
		PreRendered: true,
	})

	highlighted := mr.highlightCode(code, language)
	for _, line := range highlighted {
		paddedSegs := []render.StyledSegment{
			{Text: "  ", Style: theme.MarkdownCodeBlock},
		}
		paddedSegs = append(paddedSegs, line.Segments...)
		*lines = append(*lines, StyledLine{Segments: paddedSegs, PreRendered: true})
	}

	*lines = append(*lines, StyledLine{})
}

func (mr *MarkdownRenderer) renderIndentedCodeBlock(codeBlock *ast.CodeBlock, source []byte, lines *[]StyledLine) {
	rawText := string(codeBlock.Text(source))
	codeLines := strings.Split(rawText, "\n")
	if len(codeLines) > 0 && codeLines[len(codeLines)-1] == "" {
		codeLines = codeLines[:len(codeLines)-1]
	}

	for _, line := range codeLines {
		*lines = append(*lines, StyledLine{
			Segments: []render.StyledSegment{
				{Text: "    " + line, Style: theme.MarkdownCodeBlock},
			},
			PreRendered: true,
		})
	}
	*lines = append(*lines, StyledLine{})
}

func (mr *MarkdownRenderer) highlightCode(code string, language string) []StyledLine {
	if code == "" {
		return []StyledLine{}
	}

	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get(mr.theme)
	if style == nil {
		style = styles.Fallback
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		lines := []StyledLine{}
		for _, l := range strings.Split(code, "\n") {
			lines = append(lines, StyledLine{Segments: []render.StyledSegment{{Text: l, Style: theme.MarkdownCodeBlock}}})
		}
		return lines
	}

	var result []StyledLine
	currentLine := StyledLine{}

	for _, token := range iterator.Tokens() {
		value := token.Value
		lines := strings.Split(value, "\n")

		for i, part := range lines {
			if i > 0 {
				result = append(result, currentLine)
				currentLine = StyledLine{}
			}
			if part == "" {
				continue
			}

			fg := theme.TextPrimary
			if tokenType := token.Type; tokenType != chroma.Text {
				entry := style.Get(tokenType)
				if entry.Colour.IsSet() {
					fg = theme.Color(tcell.NewRGBColor(int32(entry.Colour.Red()), int32(entry.Colour.Green()), int32(entry.Colour.Blue())))
				}
			}

			currentLine.Segments = append(currentLine.Segments, render.StyledSegment{
				Text:  part,
				Style: theme.NewStyle().WithForeground(fg).WithBackground(theme.BgSecondary),
			})
		}
	}

	if len(currentLine.Segments) > 0 {
		result = append(result, currentLine)
	}

	return result
}

func (mr *MarkdownRenderer) renderList(list *ast.List, source []byte, lines *[]StyledLine, depth int) {
	for child := list.FirstChild(); child != nil; child = child.NextSibling() {
		mr.renderNode(child, source, lines, depth+1)
	}
}

func (mr *MarkdownRenderer) renderListItem(item *ast.ListItem, source []byte, lines *[]StyledLine, depth int) {
	indent := strings.Repeat("  ", depth)
	bullet := theme.Bullet

	textContent := mr.extractText(item, source)

	wrapped := render.WrapText(textContent, max(1, 76-len(indent)))
	if len(wrapped) > 0 {
		*lines = append(*lines, StyledLine{
			Segments: []render.StyledSegment{
				{Text: indent, Style: theme.MarkdownParagraph},
				{Text: bullet + " ", Style: theme.MarkdownListBullet},
				{Text: wrapped[0], Style: theme.MarkdownParagraph},
			},
		})
		for _, line := range wrapped[1:] {
			*lines = append(*lines, StyledLine{
				Segments: []render.StyledSegment{
					{Text: indent + "  " + line, Style: theme.MarkdownParagraph},
				},
			})
		}
	}

	for child := item.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() == ast.KindList {
			mr.renderNode(child, source, lines, depth+1)
		}
	}
}

func (mr *MarkdownRenderer) extractText(n ast.Node, source []byte) string {
	var buf bytes.Buffer
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch t := child.(type) {
		case *ast.Text:
			buf.Write(t.Segment.Value(source))
		case *ast.String:
			buf.Write(t.Value)
		case *ast.CodeSpan:
			buf.WriteString("`")
			for c := t.FirstChild(); c != nil; c = c.NextSibling() {
				if textNode, ok := c.(*ast.Text); ok {
					buf.Write(textNode.Segment.Value(source))
				}
			}
			buf.WriteString("`")
		case *ast.Emphasis:
			marker := "*"
			if t.Level == 2 {
				marker = "**"
			}
			buf.WriteString(marker)
			for c := t.FirstChild(); c != nil; c = c.NextSibling() {
				if textNode, ok := c.(*ast.Text); ok {
					buf.Write(textNode.Segment.Value(source))
				}
			}
			buf.WriteString(marker)
		case *ast.Link:
			buf.WriteString("[")
			for c := t.FirstChild(); c != nil; c = c.NextSibling() {
				if textNode, ok := c.(*ast.Text); ok {
					buf.Write(textNode.Segment.Value(source))
				}
			}
			buf.WriteString("](" + string(t.Destination) + ")")
		case *ast.AutoLink:
			buf.WriteString(string(t.URL(source)))
		default:
			if child.HasChildren() {
				buf.WriteString(mr.extractText(child, source))
			}
		}
	}
	return buf.String()
}

func (mr *MarkdownRenderer) headingStyle(level int) theme.Style {
	switch level {
	case 1:
		return theme.MarkdownHeading1
	case 2:
		return theme.MarkdownHeading2
	case 3:
		return theme.MarkdownHeading3
	case 4:
		return theme.MarkdownHeading4
	case 5:
		return theme.MarkdownHeading5
	case 6:
		return theme.MarkdownHeading6
	default:
		return theme.MarkdownHeading1
	}
}

func (mr *MarkdownRenderer) RenderInline(text string) []render.StyledSegment {
	var segments []render.StyledSegment
	remaining := text

	for len(remaining) > 0 {
		if strings.HasPrefix(remaining, "**") {
			end := strings.Index(remaining[2:], "**")
			if end != -1 {
				bold := remaining[2 : end+2]
				segments = append(segments, render.StyledSegment{
					Text:  bold,
					Style: theme.MarkdownBold,
				})
				remaining = remaining[end+4:]
				continue
			}
		}

		if strings.HasPrefix(remaining, "*") && !strings.HasPrefix(remaining, "**") {
			end := strings.Index(remaining[1:], "*")
			if end != -1 {
				italic := remaining[1 : end+1]
				segments = append(segments, render.StyledSegment{
					Text:  italic,
					Style: theme.MarkdownItalic,
				})
				remaining = remaining[end+2:]
				continue
			}
		}

		if strings.HasPrefix(remaining, "~~") {
			end := strings.Index(remaining[2:], "~~")
			if end != -1 {
				strike := remaining[2 : end+2]
				segments = append(segments, render.StyledSegment{
					Text:  strike,
					Style: theme.MarkdownStrikethrough,
				})
				remaining = remaining[end+4:]
				continue
			}
		}

		if strings.HasPrefix(remaining, "`") {
			end := strings.Index(remaining[1:], "`")
			if end != -1 {
				code := remaining[1 : end+1]
				segments = append(segments, render.StyledSegment{
					Text:  "`" + code + "`",
					Style: theme.MarkdownCode,
				})
				remaining = remaining[end+2:]
				continue
			}
		}

		nextSpecial := len(remaining)
		for _, prefix := range []string{"**", "*", "~~", "`"} {
			idx := strings.Index(remaining, prefix)
			if idx >= 0 && idx < nextSpecial {
				nextSpecial = idx
			}
		}

		if nextSpecial > 0 {
			segments = append(segments, render.StyledSegment{
				Text:  remaining[:nextSpecial],
				Style: theme.MarkdownParagraph,
			})
			remaining = remaining[nextSpecial:]
		} else {
			segments = append(segments, render.StyledSegment{
				Text:  remaining,
				Style: theme.MarkdownParagraph,
			})
			break
		}
	}

	return segments
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
