package components

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type Textarea struct {
	text                 []rune
	cursorPos            int
	selectionStart       int
	selectionEnd         int
	hasSelection         bool
	placeholder          string
	focused              bool
	width                int
	minHeight            int
	maxHeight            int
	scrollOffset         int
	cursorVisible        bool
	lastActivity         time.Time
	attachments          []*Attachment
	pendingFilePathCheck bool
	committedBlock       bool
	OnSubmit             func(string)
	OnAttachmentsChange  func([]*Attachment)
	OnTextChange         func()
}

func (t *Textarea) triggerTextChange() {
	if t.OnTextChange != nil {
		t.OnTextChange()
	}
}

func (t *Textarea) AttachmentsHeight() int {
	if len(t.attachments) > 0 {
		return 2
	}
	return 0
}

const (
	MinTextareaHeight = 3
	MaxTextareaHeight = 6
	CursorBlinkInterval = 500 * time.Millisecond
	MaxInputWidth      = 80
)

func NewTextarea(placeholder string, width int, onSubmit func(string)) *Textarea {
	return &Textarea{
		text:           []rune{},
		cursorPos:      0,
		selectionStart: 0,
		selectionEnd:   0,
		hasSelection:   false,
		placeholder:    placeholder,
		focused:        true,
		width:          width,
		minHeight:      MinTextareaHeight,
		maxHeight:      MaxTextareaHeight,
		scrollOffset:   0,
		cursorVisible:  true,
		lastActivity:   time.Now(),
		OnSubmit:       onSubmit,
	}
}

func (t *Textarea) Focus() {
	t.focused = true
	t.cursorVisible = true
	t.lastActivity = time.Now()
}

func (t *Textarea) Blur() {
	t.focused = false
}

func (t *Textarea) IsFocused() bool {
	return t.focused
}

func (t *Textarea) Text() string {
	return string(t.text)
}

func (t *Textarea) Clear() {
	t.text = []rune{}
	t.cursorPos = 0
	t.scrollOffset = 0
	t.hasSelection = false
	t.attachments = nil
	t.pendingFilePathCheck = false
}

func (t *Textarea) SetText(s string) {
	t.text = []rune(s)
	t.cursorPos = len(t.text)
	t.scrollOffset = 0
	t.hasSelection = false
	t.committedBlock = false
	t.triggerTextChange()
}

func (t *Textarea) SetCommittedText(s string) {
	t.text = []rune(s)
	t.cursorPos = len(t.text)
	t.scrollOffset = 0
	t.hasSelection = false
	t.committedBlock = true
	t.triggerTextChange()
}

func (t *Textarea) SetAttachments(atts []*Attachment) {
	t.attachments = atts
}

func (t *Textarea) Attachments() []*Attachment {
	return t.attachments
}

func (t *Textarea) SetWidth(w int) {
	t.width = w
}

func (t *Textarea) Height() int {
	visibleWidth := t.getVisibleWidth()
	if visibleWidth <= 0 {
		visibleWidth = 20
	}

	lineCount := len(t.text) / visibleWidth
	if len(t.text)%visibleWidth > 0 {
		lineCount++
	}
	if lineCount == 0 {
		lineCount = 1
	}

	h := lineCount + 2
	if h < t.minHeight {
		h = t.minHeight
	}
	if h > t.maxHeight {
		h = t.maxHeight
	}
	return h
}

func (t *Textarea) BoxWidth(screenWidth int) int {
	bw := t.width
	if bw > MaxInputWidth {
		bw = MaxInputWidth
	}
	if bw > screenWidth-4 {
		bw = screenWidth - 4
	}
	if bw < 20 {
		bw = 20
	}
	return bw
}

func (t *Textarea) getVisibleWidth() int {
	w := t.width - 4
	if w <= 0 {
		w = 20
	}
	return w
}

func (t *Textarea) getVisualRow(pos int) int {
	visibleWidth := t.getVisibleWidth()
	return pos / visibleWidth
}

func (t *Textarea) getVisualCol(pos int) int {
	visibleWidth := t.getVisibleWidth()
	return pos % visibleWidth
}

func (t *Textarea) UpdateCursorBlink() {
	if !t.focused {
		return
	}
	if time.Since(t.lastActivity) > CursorBlinkInterval {
		t.cursorVisible = !t.cursorVisible
		t.lastActivity = time.Now()
	}
}

func (t *Textarea) HandleEvent(ev tcell.Event) bool {
	switch e := ev.(type) {
	case *tcell.EventKey:
		return t.handleKey(e)
	case *tcell.EventMouse:
		return t.handleMouse(e)
	}
	return false
}

func (t *Textarea) handleKey(ev *tcell.EventKey) bool {
	if !t.focused {
		return false
	}

	t.cursorVisible = true
	t.lastActivity = time.Now()
	t.pendingFilePathCheck = true
	t.committedBlock = false

	mod := ev.Modifiers()
	shiftHeld := mod&tcell.ModShift != 0
	ctrlHeld := mod&tcell.ModCtrl != 0

	switch ev.Key() {
	case tcell.KeyTab:
		if len(t.attachments) > 0 {
			return false
		}
		return true

	case tcell.KeyEnter:
		if (len(t.text) > 0 || len(t.attachments) > 0) && t.OnSubmit != nil {
			t.OnSubmit(string(t.text))
			t.text = []rune{}
			t.cursorPos = 0
			t.scrollOffset = 0
			t.clearSelection()
		}
		t.triggerTextChange()
		return true

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if shiftHeld {
			t.text = []rune{}
			t.cursorPos = 0
			t.scrollOffset = 0
			t.clearSelection()
			t.triggerTextChange()
		} else if t.hasSelection {
			t.deleteSelection()
			t.triggerTextChange()
		} else if t.cursorPos > 0 {
			if t.committedBlock {
				blockEnd := t.prefixBlockEnd()
				if blockEnd > 0 && t.cursorPos <= blockEnd {
					rest := make([]rune, len(t.text)-blockEnd)
					copy(rest, t.text[blockEnd:])
					t.text = rest
					t.cursorPos = 0
					t.adjustScroll()
					t.triggerTextChange()
					return true
				}
				atStart := t.mentionAtPos(t.cursorPos - 1)
				if atStart >= 0 {
					atEnd := t.mentionEnd(atStart)
					if t.cursorPos <= atEnd {
						newText := append([]rune{}, t.text[:atStart]...)
						newText = append(newText, t.text[atEnd:]...)
						t.text = newText
						t.cursorPos = atStart
						t.adjustScroll()
						t.triggerTextChange()
						return true
					}
				}
			}
			t.text = append(t.text[:t.cursorPos-1], t.text[t.cursorPos:]...)
			t.cursorPos--
			t.adjustScroll()
			t.triggerTextChange()
		}
		return true

	case tcell.KeyDelete:
		if t.hasSelection {
			t.deleteSelection()
			t.triggerTextChange()
		} else if t.cursorPos < len(t.text) {
			t.text = append(t.text[:t.cursorPos], t.text[t.cursorPos+1:]...)
			t.adjustScroll()
			t.triggerTextChange()
		}
		return true

	case tcell.KeyUp:
		if shiftHeld {
			t.moveUpWithSelection()
		} else {
			t.moveUp()
		}
		return true

	case tcell.KeyDown:
		if shiftHeld {
			t.moveDownWithSelection()
		} else {
			t.moveDown()
		}
		return true

	case tcell.KeyLeft:
		if shiftHeld && ctrlHeld {
			t.moveLeftWordWithSelection()
		} else if shiftHeld {
			t.moveLeftWithSelection()
		} else if ctrlHeld {
			if t.cursorPos > 0 {
				start := t.cursorPos - 1
				for start > 0 && t.text[start-1] != ' ' {
					start--
				}
				t.cursorPos = start
				t.adjustScroll()
			}
		} else {
			if t.cursorPos > 0 {
				t.cursorPos--
				t.adjustScroll()
			}
		}
		return true

	case tcell.KeyRight:
		if shiftHeld && ctrlHeld {
			t.moveRightWordWithSelection()
		} else if shiftHeld {
			t.moveRightWithSelection()
		} else if ctrlHeld {
			if t.cursorPos < len(t.text) {
				end := t.cursorPos + 1
				for end < len(t.text) && t.text[end] != ' ' {
					end++
				}
				t.cursorPos = end
				t.adjustScroll()
			}
		} else {
			if t.cursorPos < len(t.text) {
				t.cursorPos++
				t.adjustScroll()
			}
		}
		return true

	case tcell.KeyHome:
		t.cursorPos = 0
		t.scrollOffset = 0
		return true

	case tcell.KeyEnd:
		t.cursorPos = len(t.text)
		t.adjustScroll()
		return true

	case tcell.KeyCtrlA:
		t.cursorPos = 0
		t.scrollOffset = 0
		return true

	case tcell.KeyCtrlE:
		t.cursorPos = len(t.text)
		t.adjustScroll()
		return true

	case tcell.KeyCtrlU:
		t.text = []rune{}
		t.cursorPos = 0
		t.scrollOffset = 0
		t.clearSelection()
		t.triggerTextChange()
		return true

	case tcell.KeyCtrlW:
		if t.hasSelection {
			t.deleteSelection()
			t.triggerTextChange()
		} else if t.cursorPos > 0 {
			start := t.cursorPos - 1
			for start > 0 && t.text[start-1] != ' ' {
				start--
			}
			t.text = append(t.text[:start], t.text[t.cursorPos:]...)
			t.cursorPos = start
			t.adjustScroll()
			t.triggerTextChange()
		}
		return true

	case tcell.KeyCtrlK:
		if t.hasSelection {
			t.deleteSelection()
			t.triggerTextChange()
		} else if t.cursorPos < len(t.text) {
			t.text = t.text[:t.cursorPos]
			t.triggerTextChange()
		}
		return true

	case tcell.KeyCtrlD:
		if t.hasSelection {
			t.deleteSelection()
			t.triggerTextChange()
		} else if t.cursorPos < len(t.text) {
			t.text = append(t.text[:t.cursorPos], t.text[t.cursorPos+1:]...)
			t.adjustScroll()
			t.triggerTextChange()
		}
		return true

	case tcell.KeyRune:
		if t.hasSelection {
			t.deleteSelection()
		}
		ch := ev.Rune()
		if ch >= 32 && ch < 127 {
			t.insertRune(ch)
			t.adjustScroll()
		}
		return true
	}

	return false
}

func (t *Textarea) handleMouse(ev *tcell.EventMouse) bool {
	return false
}

func (t *Textarea) insertRune(ch rune) {
	t.text = append(t.text[:t.cursorPos], append([]rune{ch}, t.text[t.cursorPos:]...)...)
	t.cursorPos++
	t.pendingFilePathCheck = true
	t.triggerTextChange()
}

func (t *Textarea) moveUp() {
	visibleWidth := t.getVisibleWidth()
	if t.cursorPos >= visibleWidth {
		t.cursorPos -= visibleWidth
		t.adjustScroll()
	}
}

func (t *Textarea) moveDown() {
	visibleWidth := t.getVisibleWidth()
	newPos := t.cursorPos + visibleWidth
	if newPos <= len(t.text) {
		t.cursorPos = newPos
	} else {
		t.cursorPos = len(t.text)
	}
	t.adjustScroll()
}

func (t *Textarea) moveUpWithSelection() {
	visibleWidth := t.getVisibleWidth()
	if !t.hasSelection {
		t.selectionStart = t.cursorPos
		t.hasSelection = true
	}
	if t.cursorPos >= visibleWidth {
		t.cursorPos -= visibleWidth
	} else {
		t.cursorPos = 0
	}
	t.selectionEnd = t.cursorPos
	t.adjustScroll()
}

func (t *Textarea) moveDownWithSelection() {
	visibleWidth := t.getVisibleWidth()
	if !t.hasSelection {
		t.selectionStart = t.cursorPos
		t.hasSelection = true
	}
	newPos := t.cursorPos + visibleWidth
	if newPos <= len(t.text) {
		t.cursorPos = newPos
	} else {
		t.cursorPos = len(t.text)
	}
	t.selectionEnd = t.cursorPos
	t.adjustScroll()
}

func (t *Textarea) moveLeftWithSelection() {
	if !t.hasSelection {
		t.selectionStart = t.cursorPos
		t.hasSelection = true
	}
	if t.cursorPos > 0 {
		t.cursorPos--
	}
	t.selectionEnd = t.cursorPos
	t.adjustScroll()
}

func (t *Textarea) moveRightWithSelection() {
	if !t.hasSelection {
		t.selectionStart = t.cursorPos
		t.hasSelection = true
	}
	if t.cursorPos < len(t.text) {
		t.cursorPos++
	}
	t.selectionEnd = t.cursorPos
	t.adjustScroll()
}

func (t *Textarea) moveLeftWordWithSelection() {
	if !t.hasSelection {
		t.selectionStart = t.cursorPos
		t.hasSelection = true
	}
	if t.cursorPos > 0 {
		start := t.cursorPos - 1
		for start > 0 && t.text[start-1] != ' ' {
			start--
		}
		t.cursorPos = start
	}
	t.selectionEnd = t.cursorPos
	t.adjustScroll()
}

func (t *Textarea) moveRightWordWithSelection() {
	if !t.hasSelection {
		t.selectionStart = t.cursorPos
		t.hasSelection = true
	}
	if t.cursorPos < len(t.text) {
		end := t.cursorPos + 1
		for end < len(t.text) && t.text[end] != ' ' {
			end++
		}
		t.cursorPos = end
	}
	t.selectionEnd = t.cursorPos
	t.adjustScroll()
}

func (t *Textarea) clearSelection() {
	t.hasSelection = false
	t.selectionStart = 0
	t.selectionEnd = 0
}

func (t *Textarea) deleteSelection() {
	if !t.hasSelection {
		return
	}
	start := t.selectionStart
	end := t.selectionEnd
	if start > end {
		start, end = end, start
	}
	t.text = append(t.text[:start], t.text[end:]...)
	t.cursorPos = start
	t.clearSelection()
	t.adjustScroll()
}

func (t *Textarea) checkForFilePath() {
	if len(t.text) < 3 {
		return
	}
	text := string(t.text)
	if filePath := LookupFileFromText(text); filePath != "" {
		if att, err := NewFileAttachment(filePath); err == nil {
			t.text = []rune{}
			t.cursorPos = 0
			t.scrollOffset = 0
			t.clearSelection()
			t.attachments = append(t.attachments, att)
			t.pendingFilePathCheck = false
			if t.OnAttachmentsChange != nil {
				t.OnAttachmentsChange(t.attachments)
			}
		}
	}
}

func hasFileExt(path string) bool {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return i > 0 && i < len(path)-1
		}
		if path[i] == '/' || path[i] == '\\' {
			break
		}
	}
	return false
}

func (t *Textarea) ConvertTextToAttachment() bool {
	if len(t.text) == 0 || len(t.attachments) > 0 {
		return false
	}
	text := string(t.text)
	if filePath := LookupFileFromText(text); filePath != "" && hasFileExt(filePath) {
		if att, err := NewFileAttachment(filePath); err == nil {
			t.text = []rune{}
			t.cursorPos = 0
			t.scrollOffset = 0
			t.clearSelection()
			t.attachments = append(t.attachments, att)
			t.pendingFilePathCheck = false
			if t.OnAttachmentsChange != nil {
				t.OnAttachmentsChange(t.attachments)
			}
			return true
		}
	}
	if len(text) > 50 {
		att := NewTextAttachment(text)
		t.text = []rune{}
		t.cursorPos = 0
		t.scrollOffset = 0
		t.clearSelection()
		t.attachments = append(t.attachments, att)
		t.pendingFilePathCheck = false
		if t.OnAttachmentsChange != nil {
			t.OnAttachmentsChange(t.attachments)
		}
		return true
	}
	return false
}

func (t *Textarea) prefixBlockEnd() int {
	if len(t.text) == 0 || t.text[0] != '/' {
		return 0
	}
	for i := 1; i < len(t.text); i++ {
		if t.text[i] == ' ' {
			return i
		}
	}
	return len(t.text)
}

func (t *Textarea) mentionAtPos(pos int) int {
	for i := pos; i >= 0; i-- {
		if t.text[i] == ' ' {
			return -1
		}
		if t.text[i] == '@' {
			return i
		}
		if i == 0 {
			break
		}
	}
	if pos < len(t.text) && t.text[0] == '@' {
		return 0
	}
	return -1
}

func (t *Textarea) mentionEnd(start int) int {
	for i := start + 1; i < len(t.text); i++ {
		if t.text[i] == ' ' {
			return i
		}
	}
	return len(t.text)
}

func (t *Textarea) prefixStyle(pos int) tcell.Style {
	if len(t.text) == 0 {
		return theme.InputTextStyle.ToTcell()
	}
	if pos == 0 && t.text[0] == '/' {
		blockEnd := t.prefixBlockEnd()
		if pos < blockEnd {
			return theme.ActionBlockStyle.ToTcell()
		}
	}
	atStart := t.mentionAtPos(pos)
	if atStart >= 0 {
		atEnd := t.mentionEnd(atStart)
		if pos >= atStart && pos < atEnd {
			return theme.MentionBlockStyle.ToTcell()
		}
	}
	return theme.InputTextStyle.ToTcell()
}

func (t *Textarea) adjustScroll() {
	visualRow := t.getVisualRow(t.cursorPos)
	maxVisible := t.maxHeight - 2

	if visualRow < t.scrollOffset {
		t.scrollOffset = visualRow
	} else if visualRow >= t.scrollOffset+maxVisible {
		t.scrollOffset = visualRow - maxVisible + 1
	}

	if t.scrollOffset < 0 {
		t.scrollOffset = 0
	}
}

func (t *Textarea) Draw(s tcell.Screen, y, screenWidth int) {
	t.UpdateCursorBlink()

	boxWidth := t.width
	if boxWidth > MaxInputWidth {
		boxWidth = MaxInputWidth
	}
	if boxWidth > screenWidth-4 {
		boxWidth = screenWidth - 4
	}
	if boxWidth < 20 {
		boxWidth = 20
	}

	startX := render.CenterX(boxWidth, screenWidth)
	height := t.Height()
	visibleWidth := t.getVisibleWidth()

	var borderStyle theme.Style
	if t.focused {
		borderStyle = theme.InputBorderFocusStyle
	} else {
		borderStyle = theme.InputBorderStyle
	}

	FillArea(s, startX, y, boxWidth, height, tcell.StyleDefault.Background(theme.BgSecondary))
	DrawBox(s, startX, y, boxWidth, height, theme.RoundedBorder, borderStyle.ToTcell())

	contentY := y + 1
	contentHeight := height - 2

	if len(t.text) == 0 {
		for i, r := range t.placeholder {
			if i >= visibleWidth {
				break
			}
			s.SetContent(startX+2+i, contentY, r, nil, theme.PlaceholderStyle.ToTcell())
		}
		if t.focused && t.cursorVisible {
			cursorStyle := tcell.StyleDefault.
				Foreground(theme.BgPrimary).
				Background(theme.AccentCyan).
				Bold(true)
			s.SetContent(startX+2, contentY, ' ', nil, cursorStyle)
		}
		return
	}

	for i := 0; i < contentHeight; i++ {
		lineIdx := t.scrollOffset + i
		startChar := lineIdx * visibleWidth
		endChar := startChar + visibleWidth
		if endChar > len(t.text) {
			endChar = len(t.text)
		}
		if startChar >= len(t.text) {
			break
		}

		for j, r := range t.text[startChar:endChar] {
			pos := startChar + j
			style := t.prefixStyle(pos)
			
			if t.hasSelection {
				selStart := t.selectionStart
				selEnd := t.selectionEnd
				if selStart > selEnd {
					selStart, selEnd = selEnd, selStart
				}
				if pos >= selStart && pos < selEnd {
					style = tcell.StyleDefault.
						Foreground(theme.AccentCyan).
						Background(theme.BgTertiary)
				}
			}
			
			s.SetContent(startX+2+j, contentY+i, r, nil, style)
		}
	}

	if t.focused {
		visualRow := t.getVisualRow(t.cursorPos) - t.scrollOffset
		visualCol := t.getVisualCol(t.cursorPos)

		if visualRow >= 0 && visualRow < contentHeight {
			cursorChar := ' '
			if t.cursorPos < len(t.text) {
				cursorChar = t.text[t.cursorPos]
			}

			var cs tcell.Style
			if t.cursorVisible {
				cs = tcell.StyleDefault.
					Foreground(theme.BgPrimary).
					Background(theme.AccentCyan).
					Bold(true)
			} else {
				cs = tcell.StyleDefault.
					Foreground(theme.AccentCyan).
					Background(theme.BgSecondary)
			}

			cursorX := startX + 2 + visualCol
			if cursorX < startX+boxWidth-2 {
				s.SetContent(cursorX, contentY+visualRow, cursorChar, nil, cs)
			}
		}
	}

	lineCount := len(t.text) / visibleWidth
	if len(t.text)%visibleWidth > 0 {
		lineCount++
	}
	if lineCount > contentHeight {
		scrollbarHeight := contentHeight
		thumbSize := (scrollbarHeight * scrollbarHeight) / lineCount
		if thumbSize < 1 {
			thumbSize = 1
		}
		thumbPos := 0
		if lineCount-scrollbarHeight > 0 {
			thumbPos = (t.scrollOffset * (scrollbarHeight - thumbSize)) / (lineCount - scrollbarHeight)
		}

		scrollX := startX + boxWidth - 2
		for i := 0; i < scrollbarHeight; i++ {
			if i >= thumbPos && i < thumbPos+thumbSize {
				s.SetContent(scrollX, contentY+i, '█', nil, tcell.StyleDefault.Foreground(theme.BorderDefault).Background(theme.BgSecondary))
			} else {
				s.SetContent(scrollX, contentY+i, '░', nil, tcell.StyleDefault.Foreground(theme.BgTertiary).Background(theme.BgSecondary))
			}
		}
	}
}

func FillArea(s tcell.Screen, x, y, width, height int, style tcell.Style) {
	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			s.SetContent(x+dx, y+dy, ' ', nil, style)
		}
	}
}

func DrawBox(s tcell.Screen, x, y, width, height int, border theme.BorderStyle, style tcell.Style) {
	if width < 2 || height < 2 {
		return
	}

	s.SetContent(x, y, []rune(border.TopLeft)[0], nil, style)
	s.SetContent(x+width-1, y, []rune(border.TopRight)[0], nil, style)
	s.SetContent(x, y+height-1, []rune(border.BottomLeft)[0], nil, style)
	s.SetContent(x+width-1, y+height-1, []rune(border.BottomRight)[0], nil, style)

	hChar := []rune(border.Horizontal)[0]
	vChar := []rune(border.Vertical)[0]

	for i := 1; i < width-1; i++ {
		s.SetContent(x+i, y, hChar, nil, style)
		s.SetContent(x+i, y+height-1, hChar, nil, style)
	}

	for i := 1; i < height-1; i++ {
		s.SetContent(x, y+i, vChar, nil, style)
		s.SetContent(x+width-1, y+i, vChar, nil, style)
	}
}
