package components

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/vesvai/vesvai/internal/llm"
	"github.com/vesvai/vesvai/internal/tui/layout"
	"github.com/vesvai/vesvai/internal/tui/styles"
)

type ItemKind int

const (
	ItemUser ItemKind = iota
	ItemAssistant
	ItemThinking
	ItemTool
	ItemSubagent
	ItemFinished
	ItemError
)

type ChatItem struct {
	Kind ItemKind
	ID   string

	Text      string
	Reasoning string

	ToolName   string
	ToolArgs   string
	ToolOutput string
	ToolErr    string
	Diff       []DiffLine

	AgentID        string
	SubagentName   string
	SubagentStatus string

	Attachments []llm.Attachment

	Expanded bool
}

type renderLine struct {
	segs     []MdSeg
	code     bool
	diffKind byte
	heading  int
	quote    bool
	bullet   bool
	hr       bool
	header   bool
	dim      bool
	err      bool
	warning  bool
	spacer   bool
}

type Chat struct {
	items       []*ChatItem
	sel         int
	scroll      int
	autoScroll  bool
	wasAtBottom bool
	back        bool
	hasMore     bool

	onActivate func(*ChatItem)
	onBack     func()
	onLoadMore func()

	lastWidth   int
	lastVisible int
}

func NewChat() *Chat { return &Chat{autoScroll: true, lastWidth: -1} }

func (c *Chat) SetItems(items []*ChatItem) {
	c.items = items
	c.sel = len(items) - 1
	if c.sel < 0 {
		c.sel = 0
	}
	c.autoScroll = true
	c.lastWidth = -1
}

func (c *Chat) AppendItem(it *ChatItem) {
	c.items = append(c.items, it)
	c.sel = len(c.items) - 1
	if c.wasAtBottom {
		c.autoScroll = true
	}
	c.lastWidth = -1
}

func (c *Chat) PrependItems(items []*ChatItem) {
	if len(items) == 0 {
		return
	}
	added := 0
	for _, it := range items {
		added += len(c.itemLines(it, c.lastWidth))
	}
	c.items = append(items, c.items...)
	c.sel += len(items)
	c.scroll += added
	c.autoScroll = false
	c.lastWidth = -1
}

func (c *Chat) Clear() {
	c.items = nil
	c.sel = 0
	c.scroll = 0
	c.autoScroll = true
	c.lastWidth = -1
}

func (c *Chat) HasItems() bool { return len(c.items) > 0 }

func (c *Chat) Items() []*ChatItem { return c.items }

func (c *Chat) SetBack(b bool) { c.back = b }

func (c *Chat) HasBack() bool { return c.back }

func (c *Chat) SetHasMore(b bool) { c.hasMore = b }

func (c *Chat) HasMore() bool { return c.hasMore }

func (c *Chat) SetOnActivate(fn func(*ChatItem)) { c.onActivate = fn }

func (c *Chat) SetOnBack(fn func()) { c.onBack = fn }

func (c *Chat) SetOnLoadMore(fn func()) { c.onLoadMore = fn }

func (c *Chat) Invalidate() { c.lastWidth = -1 }

func (c *Chat) HandleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyUp:
		if c.sel > 0 {
			c.sel--
			c.ensureSelVisible()
		}
		c.maybeLoadMore()
		return true
	case tcell.KeyDown:
		if c.sel < len(c.items)-1 {
			c.sel++
			c.ensureSelVisible()
		}
		return true
	case tcell.KeyPgUp:
		c.autoScroll = false
		c.scroll -= c.pageSize()
		c.clampScroll()
		c.maybeLoadMore()
		return true
	case tcell.KeyPgDn:
		c.autoScroll = false
		c.scroll += c.pageSize()
		c.clampScroll()
		return true
	case tcell.KeyHome:
		c.autoScroll = false
		c.scroll = 0
		c.sel = 0
		c.maybeLoadMore()
		return true
	case tcell.KeyEnd:
		c.sel = len(c.items) - 1
		c.autoScroll = true
		return true
	case tcell.KeyEnter:
		if len(c.items) == 0 {
			return false
		}
		if c.onActivate != nil {
			c.onActivate(c.items[c.sel])
		}
		return true
	case tcell.KeyEsc:
		if c.back && c.onBack != nil {
			c.onBack()
			return true
		}
		return false
	}
	return false
}

func (c *Chat) ScrollUp(page int) {
	c.autoScroll = false
	c.scroll -= page
	c.clampScroll()
	c.maybeLoadMore()
}

func (c *Chat) ScrollDown(page int) {
	c.autoScroll = false
	c.scroll += page
	c.clampScroll()
}

func (c *Chat) ScrollToBottom() {
	c.autoScroll = true
	c.wasAtBottom = true
	c.sel = len(c.items) - 1
}

func (c *Chat) ScrollBy(delta int) {
	c.autoScroll = false
	c.scroll += delta
	if c.scroll < 0 {
		c.scroll = 0
	}
	c.clampScroll()
	if c.scroll <= 0 && c.hasMore && c.onLoadMore != nil {
		c.onLoadMore()
	}
}

func (c *Chat) HandleClick(x, y, top int) bool {
	if len(c.items) == 0 || y < top {
		return false
	}
	rel := y - top
	acc := 0
	idx := 0
	for i, it := range c.items {
		h := len(c.itemLines(it, c.lastWidth))
		if rel < acc+h {
			idx = i
			break
		}
		acc += h
		idx = i
	}
	c.sel = idx
	c.ensureSelVisible()
	if c.onActivate != nil {
		c.onActivate(c.items[idx])
	}
	return true
}

func (c *Chat) pageSize() int {
	if c.lastVisible > 0 {
		return c.lastVisible
	}
	return 20
}

func (c *Chat) maybeLoadMore() {
	if c.scroll <= 0 && c.hasMore && c.onLoadMore != nil {
		c.onLoadMore()
	}
}

func (c *Chat) clampScroll() {
	if c.scroll < 0 {
		c.scroll = 0
	}
}

func (c *Chat) ensureSelVisible() {
	c.autoScroll = false
	if c.sel < 0 {
		c.sel = 0
	}
	if c.sel >= len(c.items) {
		c.sel = len(c.items) - 1
	}
	if c.lastWidth <= 0 || len(c.items) == 0 {
		return
	}
	acc := 0
	for i := 0; i < c.sel; i++ {
		acc += len(c.itemLines(c.items[i], c.lastWidth))
	}
	selHeight := len(c.itemLines(c.items[c.sel], c.lastWidth))
	visible := c.lastVisible
	if visible <= 0 {
		return
	}
	if c.scroll > acc {
		c.scroll = acc
	}
	if c.scroll+visible < acc+selHeight {
		c.scroll = acc + selHeight - visible
	}
	if c.scroll < 0 {
		c.scroll = 0
	}
}

func (c *Chat) Draw(s tcell.Screen, bounds layout.Region, focused bool) {
	th := styles.Current()
	bg := th.Base().Background(th.Background)
	FillRegion(s, bounds, ' ', bg)

	innerW := bounds.Width - 2
	if innerW < 4 {
		return
	}
	c.lastWidth = innerW
	c.lastVisible = bounds.Height

	heights := make([]int, len(c.items))
	total := 0
	for i, it := range c.items {
		h := len(c.itemLines(it, innerW))
		heights[i] = h
		total += h
	}

	visible := bounds.Height
	needsScrollbar := total > visible
	if c.autoScroll {
		c.scroll = total - visible
	}
	if c.scroll < 0 {
		c.scroll = 0
	}
	maxScroll := total - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if c.scroll > maxScroll {
		c.scroll = maxScroll
	}
	c.wasAtBottom = c.scroll >= maxScroll-2

	itemIdx, offset := 0, 0
	acc := 0
	for i, h := range heights {
		if acc+h > c.scroll {
			itemIdx = i
			offset = c.scroll - acc
			break
		}
		acc += h
		if i == len(heights)-1 {
			itemIdx = i
			offset = c.scroll - acc
		}
	}
	if offset < 0 {
		offset = 0
	}

	if c.back && bounds.Height > 2 {
		headerStyle := th.Base().Foreground(th.Accent).Bold(true).Background(th.Background)
		DrawText(s, bounds.Left+1, bounds.Top, "← back to main chat (Esc)", headerStyle)
		bounds.Top++
		bounds.Height--
		visible = bounds.Height
	}

	y := bounds.Top
	remaining := visible
	for i := itemIdx; i < len(c.items) && remaining > 0; i++ {
		it := c.items[i]
		lines := c.itemLines(it, innerW)
		isSel := focused && i == c.sel
		for j := offset; j < len(lines) && remaining > 0; j++ {
			c.drawLine(s, bounds, y, lines[j], isSel && j == 0)
			y++
			remaining--
		}
		offset = 0
	}

	if needsScrollbar {
		c.drawScrollbar(s, bounds, total, visible, c.scroll)
	}
}

func (c *Chat) drawScrollbar(s tcell.Screen, bounds layout.Region, total, visible, scroll int) {
	if total <= visible {
		return
	}
	th := styles.Current()
	style := th.Base().Foreground(th.Border)
	x := bounds.Right() - 1

	thumbH := (visible * visible) / total
	if thumbH < 1 {
		thumbH = 1
	}
	if thumbH > visible {
		thumbH = visible
	}

	maxScroll := total - visible
	if maxScroll <= 0 {
		return
	}
	thumbPos := scroll * (visible - thumbH) / maxScroll

	for y := bounds.Top; y < bounds.Bottom(); y++ {
		rel := y - bounds.Top
		if rel >= thumbPos && rel < thumbPos+thumbH {
			s.SetContent(x, y, '▓', nil, style)
		} else {
			s.SetContent(x, y, '░', nil, style)
		}
	}
}

func (c *Chat) drawLine(s tcell.Screen, bounds layout.Region, y int, l renderLine, selected bool) {
	if y < bounds.Top || y >= bounds.Bottom() {
		return
	}
	th := styles.Current()
	base := th.Base().Background(th.Background)

	var style tcell.Style
	prefix := ""
	switch {
	case l.hr:
		style = base.Foreground(th.Border)
		line := strings.Repeat("─", bounds.Width-2)
		DrawText(s, bounds.Left+1, y, TruncateTo(line, bounds.Width-2), style)
		return
	case l.spacer:
		return
	case l.header:
		style = base.Foreground(th.Accent).Bold(true)
	case l.dim:
		style = base.Foreground(th.Hint)
	case l.err:
		style = base.Foreground(tcell.ColorRed).Bold(true)
	case l.warning:
		style = base.Foreground(th.Warning).Bold(true)
	case l.diffKind == '+':
		style = base.Foreground(th.Accent).Background(th.InputBg)
		prefix = "+ "
	case l.diffKind == '-':
		style = base.Foreground(tcell.ColorRed).Background(th.InputBg)
		prefix = "- "
	case l.code:
		style = base.Foreground(th.Hint).Background(th.InputBg)
	case l.heading > 0:
		style = base.Foreground(th.Accent).Bold(true)
	case l.quote:
		style = base.Foreground(th.Hint)
		prefix = "│ "
	case l.bullet:
		style = base.Foreground(th.Foreground)
		prefix = "• "
	default:
		style = base.Foreground(th.Foreground)
	}

	if selected {
		DrawText(s, bounds.Left, y, "▌", th.Base().Foreground(th.Accent).Background(th.Background))
	}

	if l.code || l.diffKind != 0 {
		text := prefix + l.text()
		DrawText(s, bounds.Left+1, y, TruncateTo(text, bounds.Width-2), style)
		return
	}

	x := bounds.Left + 1
	if prefix != "" {
		DrawText(s, x, y, prefix, style)
		x += len(prefix)
	}
	for _, seg := range l.segs {
		segStyle := style
		if seg.Bold {
			segStyle = segStyle.Bold(true)
		}
		if seg.Italic {
			segStyle = segStyle.Italic(true)
		}
		if seg.Code {
			segStyle = base.Foreground(th.Accent).Background(th.InputBg)
		}
		room := bounds.Right() - x
		if room <= 0 {
			break
		}
		DrawText(s, x, y, TruncateTo(seg.Text, room), segStyle)
		x += len(seg.Text)
	}
}

func (l renderLine) text() string {
	var b strings.Builder
	for _, seg := range l.segs {
		b.WriteString(seg.Text)
	}
	return b.String()
}

func (c *Chat) itemLines(it *ChatItem, width int) []renderLine {
	if width < 1 {
		width = 80
	}
	switch it.Kind {
	case ItemUser:
		lines := mdToRender(RenderMarkdown(it.Text), width, false)
		for i := range lines {
			lines[i].header = false
			lines[i].segs = append([]MdSeg{{Text: "> ", Bold: true}}, lines[i].segs...)
		}
		if len(it.Attachments) > 0 {
			attLines := attachmentBoxLines(it.Attachments, width)
			lines = append(attLines, lines...)
		}
		return lines
	case ItemAssistant:
		return mdToRender(RenderMarkdown(it.Text), width, false)
	case ItemThinking:
		if !it.Expanded {
			return []renderLine{{header: true, segs: []MdSeg{{Text: "💭 Thinking…"}}}}
		}
		out := []renderLine{{header: true, segs: []MdSeg{{Text: "💭 Thinking (collapsed below)"}}}}
		out = append(out, mdToRender(RenderMarkdown(it.Reasoning), width, true)...)
		return out
	case ItemTool:
		status := ""
		if it.ToolErr != "" {
			status = " ✖"
		} else if it.ToolOutput != "" {
			status = " ✓"
		}

		icon, displayName := toolIconAndName(it.ToolName)
		header := icon + " " + displayName + status
		if !it.Expanded {
			return []renderLine{{header: true, segs: []MdSeg{{Text: header}}}}
		}
		out := []renderLine{{header: true, segs: []MdSeg{{Text: icon + " " + displayName + status + " — click to collapse"}}}}

		if isTodoTool(it.ToolName) && it.ToolOutput != "" {
			out = append(out, todoOutputLines(it.ToolOutput, width)...)
		} else {
			if it.ToolArgs != "" {
				out = append(out, codeBlockLines("args", it.ToolArgs, width)...)
			}
			if HasDiff(it.Diff) {
				out = append(out, diffBlockLines(it.Diff, width)...)
			}
			if it.ToolErr != "" {
				out = append(out, renderLine{err: true, segs: []MdSeg{{Text: "error: " + it.ToolErr}}})
			} else if it.ToolOutput != "" {
				lines := strings.Split(it.ToolOutput, "\n")
				if len(lines) > maxOutputLines {
					truncated := lines[:maxOutputLines]
					remainder := len(lines) - maxOutputLines
					out = append(out, truncatedCodeBlockLines("result", strings.Join(truncated, "\n"), width, remainder)...)
				} else {
					out = append(out, codeBlockLines("result", it.ToolOutput, width)...)
				}
			}
		}
		return out
	case ItemSubagent:
		status := it.SubagentStatus
		if status == "" {
			status = "running"
		}
		header := "◉ " + it.SubagentName + " — " + status
		if !it.Expanded {
			return []renderLine{{header: true, segs: []MdSeg{{Text: header}}}}
		}
		return []renderLine{
			{header: true, segs: []MdSeg{{Text: header}}},
			{dim: true, segs: []MdSeg{{Text: "↳ press Enter to open this subagent's chat"}}},
		}
	case ItemFinished:
		return []renderLine{{dim: true, segs: []MdSeg{{Text: "— finished —"}}}}
	case ItemError:
		return []renderLine{{err: true, segs: []MdSeg{{Text: "✖ error: " + it.Text}}}}
	}
	return nil
}

func mdToRender(md []MdLine, width int, dim bool) []renderLine {
	var out []renderLine
	for _, ln := range md {
		if ln.Code {
			out = append(out, renderLine{code: true, segs: []MdSeg{{Text: ln.Text()}}})
			continue
		}
		if ln.Hr {
			out = append(out, renderLine{hr: true})
			continue
		}
		for _, segs := range wrapSegs(ln.Segs, width) {
			out = append(out, renderLine{
				segs:    segs,
				heading: ln.Heading,
				quote:   ln.Quote,
				bullet:  ln.Bullet,
			})
		}
	}
	return out
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

func codeBlockLines(label, text string, width int) []renderLine {
	var out []renderLine
	out = append(out, renderLine{code: true, segs: []MdSeg{{Text: "``` " + label}}})
	lines := strings.Split(text, "\n")
	for _, ln := range lines {
		out = append(out, renderLine{code: true, segs: []MdSeg{{Text: ln}}})
	}
	out = append(out, renderLine{code: true, segs: []MdSeg{{Text: "```"}}})
	return out
}

func diffBlockLines(diff []DiffLine, width int) []renderLine {
	var out []renderLine
	out = append(out, renderLine{code: true, segs: []MdSeg{{Text: "``` diff"}}})
	for _, l := range diff {
		out = append(out, renderLine{diffKind: l.Kind, segs: []MdSeg{{Text: l.Text}}})
	}
	out = append(out, renderLine{code: true, segs: []MdSeg{{Text: "```"}}})
	return out
}

const maxOutputLines = 20

func toolIconAndName(name string) (string, string) {
	switch {
	case strings.HasPrefix(name, "read:"):
		return "📖", name
	case strings.HasPrefix(name, "write:"):
		return "✏️", name
	case strings.HasPrefix(name, "edit:"):
		return "✏️", name
	case strings.HasPrefix(name, "bash:"):
		return "🔧", name
	case strings.HasPrefix(name, "glob:"):
		return "🔍", name
	case strings.HasPrefix(name, "grep:"):
		return "🔍", name
	case strings.HasPrefix(name, "webfetch:"):
		return "🌐", name
	case name == "list-todo":
		return "📋", name
	case name == "update-todo":
		return "📋", name
	default:
		return "🛠", name
	}
}

func isTodoTool(name string) bool {
	return name == "list-todo" || name == "update-todo"
}

func todoOutputLines(output string, width int) []renderLine {
	var out []renderLine
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		switch {
		case strings.Contains(trimmed, "Todos:"):
			out = append(out, renderLine{dim: true, segs: []MdSeg{{Text: line}}})
		case strings.Contains(trimmed, "[x]"):
			out = append(out, renderLine{segs: []MdSeg{{Text: line}}})
		case strings.Contains(trimmed, "[~]"):
			out = append(out, renderLine{warning: true, segs: []MdSeg{{Text: line}}})
		case strings.Contains(trimmed, "[-]"):
			out = append(out, renderLine{dim: true, segs: []MdSeg{{Text: line}}})
		case strings.Contains(trimmed, "priority: high"):
			out = append(out, renderLine{header: true, segs: []MdSeg{{Text: line}}})
		default:
			out = append(out, renderLine{segs: []MdSeg{{Text: line}}})
		}
	}
	if len(out) == 0 {
		out = append(out, renderLine{segs: []MdSeg{{Text: output}}})
	}
	return out
}

func truncatedCodeBlockLines(label, text string, width int, remainder int) []renderLine {
	var out []renderLine
	out = append(out, renderLine{code: true, segs: []MdSeg{{Text: "``` " + label}}})
	for _, ln := range strings.Split(text, "\n") {
		out = append(out, renderLine{code: true, segs: []MdSeg{{Text: ln}}})
	}
	out = append(out, renderLine{code: true, segs: []MdSeg{{Text: "```"}}})
	if remainder > 0 {
		out = append(out, renderLine{dim: true, segs: []MdSeg{{Text: fmt.Sprintf("… and %d more lines", remainder)}}})
	}
	return out
}

func formatAttachmentNames(atts []llm.Attachment) string {
	names := make([]string, len(atts))
	for i, att := range atts {
		names[i] = att.FileName
		if names[i] == "" {
			names[i] = "attachment"
		}
	}
	return strings.Join(names, ", ")
}

func attachmentBoxLines(atts []llm.Attachment, width int) []renderLine {
	var lines []renderLine
	lines = append(lines, renderLine{code: true, segs: []MdSeg{{Text: "Attachments"}}})
	for _, att := range atts {
		icon := "📄"
		switch att.Type {
		case llm.AttachmentTypeImage:
			icon = "🖼"
		case llm.AttachmentTypeAudio:
			icon = "🎵"
		}
		name := att.FileName
		if name == "" {
			name = "attachment"
		}
		lines = append(lines, renderLine{code: true, segs: []MdSeg{{Text: icon + " " + name}}})
	}
	return lines
}
