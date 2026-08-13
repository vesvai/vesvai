package components

import (
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

const (
	minVisibleRows = 3
	maxVisibleRows = 6
)

const burstWindow = 50 * time.Millisecond

type Textarea struct {
	*Base
	OnSubmit func(text string)

	lines     [][]rune
	row       int
	col       int
	rowOffset int

	selRow, selCol int
	hasSel         bool
	clipboard      string

	history []string
	histPos int
	draft   string

	paste   strings.Builder
	pasting bool

	mentionCatalog                       []Mention
	skillCatalog                         []Mention
	skillNames                           map[string]bool
	pickerOpen                           bool
	pickerKind                           rune
	pickerAnchor                         int
	pickerCursor                         int
	pickerOffset                         int
	pickerResults                        []Mention
	pickerX0, pickerY0, pickerW, pickerH int
	pickerDrawn                          bool

	OnAttachment func(a *tui.Attachment)

	lastKeyAt     time.Time
	burstStart    int
	justConverted bool
}

func NewTextarea() *Textarea {
	t := &Textarea{Base: NewBase("textarea"), histPos: -1}
	t.lines = [][]rune{{}}
	t.mentionCatalog = DefaultMentions()
	t.skillNames = make(map[string]bool)
	t.pickerAnchor = -1
	t.SetDraw(t.draw)
	return t
}

func (t *Textarea) SetMentionCatalog(catalog []Mention) {
	t.mentionCatalog = catalog
}

func (t *Textarea) MentionCatalog() []Mention { return t.mentionCatalog }

func (t *Textarea) SkillCatalog() []Mention { return t.skillCatalog }

func (t *Textarea) SetSkillCatalog(catalog []Mention) {
	t.skillCatalog = catalog
	t.skillNames = make(map[string]bool, len(catalog))
	for _, m := range catalog {
		t.skillNames[m.ID] = true
	}
}

func (t *Textarea) Focusable() bool { return true }

func (t *Textarea) DesiredHeight() int {
	rows := len(t.lines)
	if rows < minVisibleRows {
		rows = minVisibleRows
	}
	if rows > maxVisibleRows {
		rows = maxVisibleRows
	}
	return rows + 2
}

func (t *Textarea) draw(s tcell.Screen, pal *tui.Palette) {
	width := t.Width()
	height := t.Height()
	if width < 2 || height < 2 {
		return
	}
	innerW := width - 4
	if innerW < 1 {
		innerW = 1
	}
	innerH := height - 2
	x0, y0 := t.bounds.Min.X, t.bounds.Min.Y

	borderColor := pal.Border
	if t.Focused() {
		borderColor = pal.BorderFocus
	}
	border := pal.Style(borderColor, pal.Surface)
	tui.DrawLine(s, x0, y0, tui.Line{{R: '┌', S: border}})
	for x := 1; x < width-1; x++ {
		s.SetContent(x0+x, y0, '─', nil, border)
	}
	tui.DrawLine(s, x0+width-1, y0, tui.Line{{R: '┐', S: border}})
	for y := 1; y < height-1; y++ {
		s.SetContent(x0, y0+y, '│', nil, border)
		s.SetContent(x0+width-1, y0+y, '│', nil, border)
	}
	footer := tui.Line{{R: '└', S: border}}
	for x := 1; x < width-1; x++ {
		footer = append(footer, tui.Cell{R: '─', S: border})
	}
	footer = append(footer, tui.Cell{R: '┘', S: border})
	tui.DrawLine(s, x0, y0+height-1, footer)

	bg := pal.Style(pal.Foreground, pal.Surface)
	for y := 0; y < innerH; y++ {
		for x := 0; x < innerW+2; x++ {
			s.SetContent(x0+1+x, y0+1+y, ' ', nil, bg)
		}
	}

	s.SetContent(x0+1, y0+1, '❯', nil, pal.Style(pal.Accent, pal.Surface).Bold(true))

	display := t.wrapLines(t.lines, innerW)

	if len(display) == 1 && len(display[0]) == 0 && !t.Focused() {
		tui.DrawLine(s, x0+3, y0+1, tui.LineFromSegments([]tui.Segment{
			{Text: "Type a message…", Style: pal.Style(pal.Muted, pal.Surface)},
		}, innerW))
	}

	cursorRow, cursorPx := -1, 0
	if t.Focused() {
		cursorRow, cursorPx = t.cursorDisplayPos(t.lines, t.row, t.col)
	}

	if cursorRow < t.rowOffset {
		t.rowOffset = cursorRow
	}
	if cursorRow >= t.rowOffset+innerH {
		t.rowOffset = cursorRow - innerH + 1
	}
	if t.rowOffset < 0 {
		t.rowOffset = 0
	}

	selStyle := pal.Style(pal.Foreground, pal.Selection)
	normalStyle := pal.Style(pal.Foreground, pal.Surface)

	displaySrc := t.displaySources(innerW)

	for row := 0; row < innerH; row++ {
		dr := t.rowOffset + row
		if dr < 0 || dr >= len(display) {
			continue
		}
		line := display[dr]
		selStart, selEnd, inSel := t.selectionRangeForRow(dr, innerW)
		src := displaySrc[dr]
		tokens := t.blockTokens(t.lines[src.line])

		px := 0
		var cells tui.Line
		for j, r := range line {
			style := normalStyle
			if inSel && px >= selStart && px < selEnd {
				style = selStyle
			}
			if tok := tokenAt(tokens, src.offset+j); tok != nil {
				if tok.kind == '/' {
					style = style.Foreground(pal.SkillBlock)
				} else {
					style = style.Foreground(pal.Mention)
				}
			}
			cells = append(cells, tui.Cell{R: r, S: style})
			px += tui.RuneWidth(r)
		}
		for px < innerW {
			style := normalStyle
			if inSel && px >= selStart && px < selEnd {
				style = selStyle
			}
			cells = append(cells, tui.Cell{R: ' ', S: style})
			px++
		}
		tui.DrawLine(s, x0+3, y0+1+row, cells)
	}

	if t.Focused() && cursorRow >= 0 {
		cy := y0 + 1 + (cursorRow - t.rowOffset)
		cx := x0 + 3 + cursorPx
		if cy >= y0+1 && cy < y0+1+innerH && cx < x0+width-1 {
			ch := rune(' ')
			row := cursorRow - t.rowOffset
			if row >= 0 && row < len(display) && cursorPx < tui.DisplayWidth(string(display[row])) {
				ch = tui.FirstRuneAtWidth(display[row], cursorPx)
			}
			s.SetContent(cx, cy, ch, nil, pal.Style(pal.Background, pal.Accent))
		}
	}

	if t.pickerOpen && len(t.pickerResults) > 0 {
		if t.pickerDrawn {
			t.clearPickerArea(s, pal)
		}
		t.drawMentionPicker(s, pal)
		t.pickerDrawn = true
	} else if t.pickerDrawn {
		t.clearPickerArea(s, pal)
		t.pickerDrawn = false
	}
}

func (t *Textarea) clearPickerArea(s tcell.Screen, pal *tui.Palette) {
	if t.pickerW <= 0 || t.pickerH <= 0 {
		return
	}
	style := pal.Style(pal.Foreground, pal.Background)
	for y := 0; y < t.pickerH; y++ {
		for x := 0; x < t.pickerW; x++ {
			s.SetContent(t.pickerX0+x, t.pickerY0+y, ' ', nil, style)
		}
	}
}

type displaySource struct{ line, offset int }

func (t *Textarea) displaySources(innerW int) []displaySource {
	var src []displaySource
	li := 0
	off := 0
	for _, ln := range t.lines {
		for _, r := range t.wrapLines([][]rune{ln}, innerW) {
			src = append(src, displaySource{line: li, offset: off})
			off += len(r)
		}
		li++
		off = 0
	}
	return src
}

func (t *Textarea) wrapLines(lines [][]rune, width int) [][]rune {
	if width < 1 {
		width = 1
	}
	var out [][]rune
	for _, ln := range lines {
		if len(ln) == 0 {
			out = append(out, []rune{})
			continue
		}
		var cur []rune
		w := 0
		for _, r := range ln {
			rw := tui.RuneWidth(r)
			if w+rw > width && len(cur) > 0 {
				out = append(out, cur)
				cur = nil
				w = 0
			}
			cur = append(cur, r)
			w += rw
		}
		out = append(out, cur)
	}
	return out
}

func (t *Textarea) cursorDisplayPos(lines [][]rune, row, col int) (int, int) {
	width := t.Width() - 4
	if width < 1 {
		width = 1
	}
	drow := 0
	for i := 0; i < row && i < len(lines); i++ {
		drow += len(t.wrapLines([][]rune{lines[i]}, width))
	}
	line := []rune{}
	if row >= 0 && row < len(lines) {
		line = lines[row]
	}
	w := 0
	within := 0
	for i := 0; i < len(line); i++ {
		rw := tui.RuneWidth(line[i])
		if w+rw > width && w > 0 {
			within++
			w = 0
		}
		if i == col {
			return drow + within, w
		}
		w += rw
	}
	return drow + within, w
}

func (t *Textarea) displayToLogical(lines [][]rune, drow, px int) (int, int) {
	width := t.Width() - 4
	if width < 1 {
		width = 1
	}
	cursor := drow
	for i := 0; i < len(lines); i++ {
		rows := t.wrapLines([][]rune{lines[i]}, width)
		if cursor < len(rows) {
			col := 0
			w := 0
			for j := 0; j < len(rows[cursor]); j++ {
				rw := tui.RuneWidth(rows[cursor][j])
				if w+rw > px && w >= px {
					break
				}
				w += rw
				col++
				if w >= px {
					break
				}
			}
			for k := 0; k < cursor; k++ {
				col += len(rows[k])
			}
			return i, col
		}
		cursor -= len(rows)
	}
	return len(lines) - 1, len(lines[len(lines)-1])
}

func (t *Textarea) selectionRangeForRow(dr, width int) (int, int, bool) {
	if !t.hasSel {
		return 0, 0, false
	}
	aRow, aPx := t.cursorDisplayPos(t.lines, t.selRow, t.selCol)
	hRow, hPx := t.cursorDisplayPos(t.lines, t.row, t.col)
	lo, hi := aRow, hRow
	loPx, hiPx := aPx, hPx
	if hRow < aRow || (hRow == aRow && hPx < aPx) {
		lo, hi = hRow, aRow
		loPx, hiPx = hPx, aPx
	}
	if dr < lo || dr > hi {
		return 0, 0, false
	}
	switch {
	case lo == hi:
		return loPx, hiPx, true
	case dr == lo:
		return loPx, width + 1, true
	case dr == hi:
		return 0, hiPx, true
	default:
		return 0, width + 1, true
	}
}

func (t *Textarea) selStart() (int, int) {
	if !t.hasSel {
		return t.row, t.col
	}
	if t.row < t.selRow || (t.row == t.selRow && t.col < t.selCol) {
		return t.row, t.col
	}
	return t.selRow, t.selCol
}

func (t *Textarea) selEnd() (int, int) {
	if !t.hasSel {
		return t.row, t.col
	}
	if t.row > t.selRow || (t.row == t.selRow && t.col > t.selCol) {
		return t.row, t.col
	}
	return t.selRow, t.selCol
}

func (t *Textarea) selectedText() string {
	if !t.hasSel {
		return ""
	}
	sr, sc := t.selStart()
	er, ec := t.selEnd()
	var sb strings.Builder
	for i := sr; i <= er; i++ {
		ln := t.lines[i]
		switch {
		case i == sr && i == er:
			sb.WriteString(string(ln[sc:ec]))
		case i == sr:
			sb.WriteString(string(ln[sc:]))
		case i == er:
			sb.WriteString(string(ln[:ec]))
		default:
			sb.WriteString(string(ln))
		}
		if i < er {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func (t *Textarea) deleteSelection() {
	if !t.hasSel {
		return
	}
	sr, sc := t.selStart()
	er, ec := t.selEnd()
	tail := t.lines[er][ec:]
	t.lines[sr] = append(t.lines[sr][:sc], tail...)
	if er > sr {
		t.lines = append(t.lines[:sr+1], t.lines[er+1:]...)
	}
	t.row, t.col = sr, sc
	t.hasSel = false
	t.resetHistoryEdit()
	t.RequestRender()
}

func (t *Textarea) clearSelection() {
	if t.hasSel {
		t.hasSel = false
		t.RequestRender()
	}
}

func (t *Textarea) moveTo(row, col int, extend bool) {
	if extend && !t.hasSel {
		t.selRow, t.selCol = t.row, t.col
		t.hasSel = true
	}
	t.row, t.col = row, col
	t.col = t.snapMention(row, t.col)
	if !extend {
		t.hasSel = false
	}
	t.RequestRender()
}

func (t *Textarea) HandleEvent(ev tcell.Event) bool {
	switch e := ev.(type) {
	case *tcell.EventPaste:
		if t.pickerOpen {
			t.closeMention(false)
		}
		if e.Start() {
			t.pasting = true
			t.paste.Reset()
			return true
		}
		if e.End() {
			t.pasting = false
			t.justConverted = false
			t.insertPasted(t.paste.String())
			return true
		}
		return true
	case *tcell.EventKey:
		return t.handleKey(e)
	}
	return false
}

func (t *Textarea) insertPasted(text string) {
	whole := strings.Trim(strings.TrimSpace(text), `"'`)
	whole = strings.TrimPrefix(whole, "file://")
	if a, ok := tui.DetectAttachment(whole); ok {
		if t.OnAttachment != nil {
			t.OnAttachment(a)
		}
		return
	}

	var rest []string
	for _, tok := range strings.Fields(text) {
		path := strings.Trim(tok, `"'`)
		path = strings.TrimPrefix(path, "file://")
		if a, ok := tui.DetectAttachment(path); ok {
			if t.OnAttachment != nil {
				t.OnAttachment(a)
			}
			continue
		}
		rest = append(rest, tok)
	}
	if len(rest) > 0 {
		t.insertText(strings.Join(rest, " "))
	}
}

func (t *Textarea) handleKey(e *tcell.EventKey) bool {
	if t.pasting {
		switch e.Key() {
		case tcell.KeyRune:
			t.paste.WriteRune(e.Rune())
		case tcell.KeyEnter:
			t.paste.WriteRune('\n')
		}
		return true
	}

	if t.pickerOpen {
		if t.handleMentionKey(e) {
			return true
		}
		t.closeMention(false)
	}

	mods := e.Modifiers()
	extend := mods&tcell.ModShift != 0
	word := mods&tcell.ModCtrl != 0

	switch e.Key() {
	case tcell.KeyEnter:
		if mods&(tcell.ModShift|tcell.ModAlt) != 0 {
			t.insertRune('\n')
			return true
		}
		if t.justConverted && time.Since(t.lastKeyAt) < burstWindow {
			t.justConverted = false
			t.insertRune('\n')
			return true
		}
		t.justConverted = false
		text := t.text()
		if strings.TrimSpace(text) == "" {
			return true
		}
		if t.OnSubmit != nil {
			t.OnSubmit(text)
		}
		t.history = append(t.history, text)
		t.histPos = -1
		t.hasSel = false
		t.lines = [][]rune{{}}
		t.row, t.col = 0, 0
		t.rowOffset = 0
		t.RequestRender()
		return true

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if t.hasSel {
			t.deleteSelection()
		} else {
			t.backspace()
		}
		return true
	case tcell.KeyDelete:
		if mods&tcell.ModShift != 0 {
			t.cut()
			return true
		}
		if t.hasSel {
			t.deleteSelection()
		} else {
			t.deleteForward()
		}
		return true
	case tcell.KeyInsert:
		if mods&tcell.ModShift != 0 {
			t.insertText(t.clipboard)
			return true
		}
		if mods&tcell.ModCtrl != 0 {
			t.copy()
			return true
		}
		return true
	case tcell.KeyEscape:
		t.clearSelection()
		return true

	case tcell.KeyLeft:
		if word {
			t.moveTo(t.row, t.wordLeftPos(), extend)
		} else {
			t.moveLeft(extend)
		}
		return true
	case tcell.KeyRight:
		if word {
			t.moveTo(t.row, t.wordRightPos(), extend)
		} else {
			t.moveRight(extend)
		}
		return true
	case tcell.KeyUp:
		if mods&tcell.ModAlt != 0 {
			t.historyPrev()
		} else {
			t.moveUp(extend)
		}
		return true
	case tcell.KeyDown:
		if mods&tcell.ModAlt != 0 {
			t.historyNext()
		} else {
			t.moveDown(extend)
		}
		return true
	case tcell.KeyHome, tcell.KeyCtrlA:
		t.moveTo(t.row, 0, extend)
		return true
	case tcell.KeyEnd, tcell.KeyCtrlE:
		t.moveTo(t.row, len(t.currentLine()), extend)
		return true

	case tcell.KeyCtrlU:
		t.hasSel = false
		t.lines = [][]rune{{}}
		t.row, t.col = 0, 0
		t.rowOffset = 0
		t.RequestRender()
		return true
	case tcell.KeyCtrlK:
		if t.hasSel {
			t.deleteSelection()
		} else {
			t.killToEnd()
		}
		return true
	case tcell.KeyCtrlW:
		if t.hasSel {
			t.deleteSelection()
		} else {
			t.killWordBack()
		}
		return true
	case tcell.KeyRune:
		r := e.Rune()
		if r >= 32 && !unicode.IsControl(r) {
			burst := time.Since(t.lastKeyAt) < burstWindow
			if !burst {
				t.justConverted = false
				t.burstStart = t.col
			}
			t.lastKeyAt = time.Now()
			if r == '@' || r == '/' {
				t.insertRune(r)
				t.openPicker(r, t.col-1)
				return true
			}
			t.insertRune(r)
			t.maybeConvertBurstAttachment()
			return true
		}
	}
	return false
}

func (t *Textarea) maybeConvertBurstAttachment() {
	row := t.row
	if row < 0 || row >= len(t.lines) {
		return
	}
	ln := t.lines[row]
	start := t.burstStart
	col := t.col
	if start >= col || col > len(ln) {
		return
	}
	end := col
	for end > start && (ln[end-1] == ' ' || ln[end-1] == '\t') {
		end--
	}
	if end <= start {
		return
	}
	a, ok := tui.DetectAttachment(string(ln[start:end]))
	if !ok {
		return
	}
	t.lines[row] = append(ln[:start], ln[end:]...)
	t.col = start
	t.burstStart = start
	t.justConverted = true
	if t.OnAttachment != nil {
		t.OnAttachment(a)
	}
	t.resetHistoryEdit()
	t.RequestRender()
}

func (t *Textarea) mentionQuery() string {
	if t.pickerAnchor < 0 || t.row < 0 || t.row >= len(t.lines) {
		return ""
	}
	ln := t.lines[t.row]
	if t.pickerAnchor+1 > t.col || t.col > len(ln) {
		return ""
	}
	return string(ln[t.pickerAnchor+1 : t.col])
}

func (t *Textarea) updateMentionResults() {
	catalog := t.mentionCatalog
	if t.pickerKind == '/' {
		catalog = t.skillCatalog
	}
	t.pickerResults = filterMentions(catalog, t.mentionQuery())
	if t.pickerCursor >= len(t.pickerResults) {
		t.pickerCursor = len(t.pickerResults) - 1
	}
	if t.pickerCursor < 0 {
		t.pickerCursor = 0
	}
}

func (t *Textarea) openPicker(kind rune, anchor int) {
	t.pickerKind = kind
	t.pickerOpen = true
	t.pickerAnchor = anchor
	t.pickerCursor = 0
	t.pickerOffset = 0
	t.updateMentionResults()
	t.RequestRender()
}

func (t *Textarea) closeMention(revert bool) {
	if !t.pickerOpen {
		return
	}
	if revert && t.pickerAnchor >= 0 && t.row >= 0 && t.row < len(t.lines) {
		ln := t.lines[t.row]
		if t.pickerAnchor < t.col && t.col <= len(ln) {
			t.lines[t.row] = append(ln[:t.pickerAnchor], ln[t.col:]...)
			t.col = t.pickerAnchor
		}
	}
	t.pickerOpen = false
	t.pickerAnchor = -1
	t.RequestRender()
}

func (t *Textarea) selectMention() {
	if len(t.pickerResults) == 0 {
		return
	}
	m := t.pickerResults[t.pickerCursor]
	row := t.row
	ln := t.lines[row]
	prefix := append([]rune(nil), ln[:t.pickerAnchor]...)
	block := []rune(string(t.pickerKind) + m.ID)
	t.lines[row] = append(prefix, block...)
	t.lines[row] = append(t.lines[row], ln[t.col:]...)
	t.col = t.pickerAnchor + len(block)
	t.pickerOpen = false
	t.pickerAnchor = -1
	t.resetHistoryEdit()
	t.RequestRender()
}

func (t *Textarea) handleMentionKey(e *tcell.EventKey) bool {
	switch e.Key() {
	case tcell.KeyEsc:
		t.closeMention(true)
		return true
	case tcell.KeyEnter:
		t.selectMention()
		return true
	case tcell.KeyUp:
		if t.pickerCursor > 0 {
			t.pickerCursor--
			t.RequestRender()
		}
		return true
	case tcell.KeyDown:
		if t.pickerCursor < len(t.pickerResults)-1 {
			t.pickerCursor++
			t.RequestRender()
		}
		return true
	case tcell.KeyPgUp:
		t.pickerCursor -= 5
		if t.pickerCursor < 0 {
			t.pickerCursor = 0
		}
		t.RequestRender()
		return true
	case tcell.KeyPgDn:
		t.pickerCursor += 5
		if t.pickerCursor >= len(t.pickerResults) {
			t.pickerCursor = len(t.pickerResults) - 1
		}
		t.RequestRender()
		return true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if t.col > t.pickerAnchor+1 {
			t.backspaceChar()
		} else if t.col > t.pickerAnchor {
			t.closeMention(true)
		}
		t.updateMentionResults()
		return true
	case tcell.KeyRune:
		r := e.Rune()
		if r >= 32 && !unicode.IsControl(r) {
			valid := isMentionChar(r)
			if t.pickerKind == '/' {
				valid = isSkillChar(r)
			}
			if r == t.pickerKind || !valid {
				t.closeMention(false)
				return false
			}
			t.insertRune(r)
			t.updateMentionResults()
			return true
		}
	}
	return false
}

func (t *Textarea) currentLine() []rune {
	if t.row < 0 || t.row >= len(t.lines) {
		return nil
	}
	return t.lines[t.row]
}

func (t *Textarea) insertRune(r rune) {
	if t.hasSel {
		t.deleteSelection()
	}
	for t.row >= len(t.lines) {
		t.lines = append(t.lines, []rune{})
	}
	ln := t.lines[t.row]
	if r == '\n' {
		rest := append([]rune(nil), ln[t.col:]...)
		t.lines[t.row] = ln[:t.col]
		t.row++
		t.lines = append(t.lines, nil)
		copy(t.lines[t.row+1:], t.lines[t.row:])
		t.lines[t.row] = rest
		t.col = 0
	} else {
		ln = append(ln, 0)
		copy(ln[t.col+1:], ln[t.col:])
		ln[t.col] = r
		t.lines[t.row] = ln
		t.col++
	}
	t.resetHistoryEdit()
	t.RequestRender()
}

func (t *Textarea) snapMention(row, col int) int {
	if row < 0 || row >= len(t.lines) {
		return col
	}
	for _, tok := range t.blockTokens(t.lines[row]) {
		if col > tok.start && col < tok.end {
			if col-tok.start < tok.end-col {
				return tok.start
			}
			return tok.end
		}
	}
	return col
}

func (t *Textarea) backspace() {
	row, col := t.row, t.col
	if !t.pickerOpen {
		if tok := tokenEndingAt(t.blockTokens(t.lines[row]), col); tok != nil {
			ln := t.lines[row]
			t.lines[row] = append(ln[:tok.start], ln[tok.end:]...)
			t.col = tok.start
			t.resetHistoryEdit()
			t.RequestRender()
			return
		}
	}
	t.backspaceChar()
}

func (t *Textarea) backspaceChar() {
	row, col := t.row, t.col
	if col > 0 {
		ln := t.lines[row]
		t.lines[row] = append(ln[:col-1], ln[col:]...)
		t.col--
	} else if row > 0 {
		prev := t.lines[row-1]
		t.lines[row-1] = append(prev, t.lines[row]...)
		t.lines = append(t.lines[:row], t.lines[row+1:]...)
		t.row--
		t.col = len(prev)
	}
	t.resetHistoryEdit()
	t.RequestRender()
}

func (t *Textarea) deleteForward() {
	row, col := t.row, t.col
	if !t.pickerOpen {
		if tok := tokenAt(t.blockTokens(t.lines[row]), col); tok != nil {
			ln := t.lines[row]
			t.lines[row] = append(ln[:tok.start], ln[tok.end:]...)
			t.resetHistoryEdit()
			t.RequestRender()
			return
		}
	}
	if col < len(t.lines[row]) {
		ln := t.lines[row]
		t.lines[row] = append(ln[:col], ln[col+1:]...)
	} else if row < len(t.lines)-1 {
		next := t.lines[row+1]
		t.lines[row] = append(t.lines[row], next...)
		t.lines = append(t.lines[:row+1], t.lines[row+2:]...)
	}
	t.resetHistoryEdit()
	t.RequestRender()
}

func (t *Textarea) moveLeft(extend bool) {
	row, col := t.row, t.col
	if tok := tokenEndingAt(t.blockTokens(t.lines[row]), col); tok != nil {
		col = tok.start
		t.moveTo(row, col, extend)
		return
	}
	if col > 0 {
		col--
	} else if row > 0 {
		row--
		col = len(t.lines[row])
	}
	t.moveTo(row, col, extend)
}

func (t *Textarea) moveRight(extend bool) {
	row, col := t.row, t.col
	if tok := tokenAt(t.blockTokens(t.lines[row]), col); tok != nil {
		col = tok.end
		t.moveTo(row, col, extend)
		return
	}
	if col < len(t.lines[row]) {
		col++
	} else if row < len(t.lines)-1 {
		row++
		col = 0
	}
	t.moveTo(row, col, extend)
}

func (t *Textarea) moveUp(extend bool) bool {
	if t.row == 0 && t.col == 0 {
		return false
	}
	drow, px := t.cursorDisplayPos(t.lines, t.row, t.col)
	if drow == 0 {
		return false
	}
	row, col := t.displayToLogical(t.lines, drow-1, px)
	t.moveTo(row, col, extend)
	return true
}

func (t *Textarea) moveDown(extend bool) bool {
	drow, px := t.cursorDisplayPos(t.lines, t.row, t.col)
	total := 0
	for _, ln := range t.lines {
		total += len(t.wrapLines([][]rune{ln}, t.Width()-4))
	}
	if total == 0 {
		total = 1
	}
	if drow >= total-1 {
		return false
	}
	row, col := t.displayToLogical(t.lines, drow+1, px)
	t.moveTo(row, col, extend)
	return true
}

func (t *Textarea) wordLeftPos() int {
	ln := t.lines[t.row]
	i := t.col
	for i > 0 && (ln[i-1] == ' ' || ln[i-1] == '\t') {
		i--
	}
	for i > 0 && ln[i-1] != ' ' && ln[i-1] != '\t' {
		i--
	}
	return i
}

func (t *Textarea) wordRightPos() int {
	ln := t.lines[t.row]
	i := t.col
	for i < len(ln) && (ln[i] == ' ' || ln[i] == '\t') {
		i++
	}
	for i < len(ln) && ln[i] != ' ' && ln[i] != '\t' {
		i++
	}
	return i
}

func (t *Textarea) killToEnd() {
	ln := t.lines[t.row]
	t.lines[t.row] = ln[:t.col]
	t.resetHistoryEdit()
	t.RequestRender()
}

func (t *Textarea) killWordBack() {
	row, col := t.row, t.col
	ln := t.lines[row]
	i := col
	for i > 0 && (ln[i-1] == ' ' || ln[i-1] == '\t') {
		i--
	}
	for i > 0 && ln[i-1] != ' ' && ln[i-1] != '\t' {
		i--
	}
	t.lines[row] = append(ln[:i], ln[col:]...)
	t.col = i
	t.resetHistoryEdit()
	t.RequestRender()
}

func (t *Textarea) copy() {
	if t.hasSel {
		t.clipboard = t.selectedText()
	}
}

func (t *Textarea) SetClipboard(text string) {
	t.clipboard = text
}

func (t *Textarea) cut() {
	if !t.hasSel {
		return
	}
	t.clipboard = t.selectedText()
	t.deleteSelection()
}

func (t *Textarea) historyPrev() {
	if len(t.history) == 0 || t.histPos == 0 {
		return
	}
	if t.histPos < 0 {
		t.draft = t.text()
		t.histPos = len(t.history)
	}
	t.histPos--
	t.loadHistory(t.history[t.histPos])
}

func (t *Textarea) historyNext() {
	if t.histPos < 0 {
		return
	}
	t.histPos++
	if t.histPos >= len(t.history) {
		t.histPos = -1
		t.loadHistory(t.draft)
		return
	}
	t.loadHistory(t.history[t.histPos])
}

func (t *Textarea) loadHistory(text string) {
	t.hasSel = false
	t.lines = t.splitLines(text)
	t.row = len(t.lines) - 1
	t.col = len(t.lines[len(t.lines)-1])
	t.rowOffset = 0
	t.RequestRender()
}

func (t *Textarea) resetHistoryEdit() {
	if t.histPos >= 0 {
		t.histPos = -1
		t.lines = t.splitLines(t.draft)
		t.row = len(t.lines) - 1
		t.col = len(t.lines[len(t.lines)-1])
		t.rowOffset = 0
	}
}

func (t *Textarea) splitLines(s string) [][]rune {
	parts := strings.Split(s, "\n")
	lines := make([][]rune, len(parts))
	for i, p := range parts {
		lines[i] = []rune(p)
	}
	return lines
}

func (t *Textarea) text() string {
	var sb strings.Builder
	for i, ln := range t.lines {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(string(ln))
	}
	return sb.String()
}

func (t *Textarea) insertText(s string) {
	for _, r := range s {
		t.insertRune(r)
	}
}

func (t *Textarea) Tick(elapsed time.Duration) bool {
	t.now = elapsed
	return t.Hooks.Tick(elapsed)
}
