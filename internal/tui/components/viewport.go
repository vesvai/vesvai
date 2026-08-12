package components

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

type FlatItem struct {
	ID    string
	Kind  string
	Start int
	End   int
}

type FlatMsg struct {
	Msg   *tui.Message
	Start int
	End   int
}

type Flat struct {
	Lines    []tui.Line
	Items    []*FlatItem
	Messages []*FlatMsg
}

type Viewport struct {
	*Base
	conv        *tui.Conversation
	mv          *MessageView
	OnToggle    func(itemID string)
	OnUserClick func(x, y int, m *tui.Message)

	flat         *Flat
	flatRev      int64
	itemCursor   int
	follow       bool
	scrollOffset int
	lastOffset   int

	activeSubagent *tui.Subagent
	subLines       []tui.Line
	subToolBlocks  []Block
	subRev         int64

	pendingSubToggle int

	indicatorX0, indicatorX1, indicatorY int
	indicatorVisible                     bool

	backX0, backX1, backY int
	backVisible           bool
}

func NewViewport(conv *tui.Conversation) *Viewport {
	v := &Viewport{
		Base:             NewBase("viewport"),
		conv:             conv,
		mv:               NewMessageView(),
		itemCursor:       -1,
		follow:           true,
		pendingSubToggle: -1,
	}
	v.flatRev = -1
	v.subRev = -1
	v.SetDraw(v.draw)
	return v
}

func (v *Viewport) setFollow(f bool) {
	if v.follow != f {
		v.follow = f
		v.RequestRender()
	}
}

func (v *Viewport) setOffset(off int) {
	off = tui.ClampScroll(off, v.maxOffset())
	if v.scrollOffset != off {
		v.scrollOffset = off
		v.RequestRender()
	}
}

func (v *Viewport) Focusable() bool { return true }

func (v *Viewport) rebuildFlat(pal *tui.Palette) {
	width := v.Width() - 1
	if width < 1 {
		width = 1
	}
	flat := &Flat{}
	for _, m := range v.conv.Messages {
		v.mv.Render(m, width, pal)
		start := len(flat.Lines)
		lines, blocks := v.mv.Lines(m, width, pal, v.now)
		for _, blk := range blocks {
			flat.Items = append(flat.Items, &FlatItem{
				ID:    blk.ID,
				Kind:  blk.Kind,
				Start: start + blk.Start,
				End:   start + blk.End,
			})
		}
		flat.Lines = append(flat.Lines, lines...)
		flat.Lines = append(flat.Lines, nil)
		flat.Messages = append(flat.Messages, &FlatMsg{
			Msg:   m,
			Start: start,
			End:   len(flat.Lines),
		})
	}
	v.flat = flat
	v.flatRev = v.conv.Revision()
	if v.itemCursor >= len(flat.Items) {
		v.itemCursor = -1
	}
}

func (v *Viewport) draw(s tcell.Screen, pal *tui.Palette) {
	height := v.Height()
	if height <= 0 {
		return
	}
	width := v.Width()
	if width < 1 {
		return
	}

	if v.activeSubagent != nil {
		v.drawSubagentView(s, pal)
		return
	}

	if len(v.conv.Messages) == 0 {
		v.drawLogo(s, pal)
		return
	}

	if v.flat == nil || v.flatRev != v.conv.Revision() {
		v.rebuildFlat(pal)
	}

	bg := pal.Style(pal.Foreground, pal.Background)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			s.SetContent(v.bounds.Min.X+x, v.bounds.Min.Y+y, ' ', nil, bg)
		}
	}

	total := len(v.flat.Lines)
	offset := 0
	if v.following() {
		offset = tui.ClampScroll(total-height, total)
	} else {
		offset = tui.ClampScroll(v.offset(), total)
	}
	if total <= height {
		offset = 0
	}
	v.lastOffset = offset

	markerLine := -1
	if v.Focused() && v.itemCursor >= 0 && v.itemCursor < v.itemCount() {
		markerLine = v.flat.Items[v.itemCursor].Start
	}

	for row := 0; row < height; row++ {
		idx := offset + row
		if idx >= total {
			break
		}
		x := v.bounds.Min.X
		y := v.bounds.Min.Y + row
		if idx == markerLine {
			s.SetContent(x, y, '▍', nil, pal.Style(pal.Accent, pal.Background))
		} else {
			s.SetContent(x, y, ' ', nil, bg)
		}
		tui.DrawLine(s, x+1, y, v.flat.Lines[idx])
	}

	v.indicatorVisible = false
	below := total - (offset + height)
	if below > 0 {
		label := fmt.Sprintf(" ↓ %d ", below)
		labelCells := tui.LineFromSegments([]tui.Segment{
			{Text: label, Style: pal.Style(pal.Accent, pal.Surface)},
		}, len(label))
		lx := v.bounds.Min.X + width - len(label) - 1
		ly := v.bounds.Min.Y + height - 1
		tui.DrawLine(s, lx, ly, labelCells)
		v.indicatorX0, v.indicatorX1, v.indicatorY = lx, lx+len(label)-1, ly
		v.indicatorVisible = true
	}
}

func (v *Viewport) drawSubagentView(s tcell.Screen, pal *tui.Palette) {
	height := v.Height()
	width := v.Width()
	x0, y0 := v.bounds.Min.X, v.bounds.Min.Y

	bg := pal.Style(pal.Foreground, pal.Background)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			s.SetContent(x0+x, y0+y, ' ', nil, bg)
		}
	}

	if v.subRev != v.conv.Revision() {
		v.rebuildSubagent(pal)
	}
	if v.pendingSubToggle >= 0 {
		idx := v.pendingSubToggle
		v.pendingSubToggle = -1
		v.applySubagentToggle(idx)
		if v.subRev != v.conv.Revision() {
			v.rebuildSubagent(pal)
		}
	}

	total := len(v.subLines)
	offset := 0
	if v.follow {
		offset = tui.ClampScroll(total-height, total)
	} else {
		offset = tui.ClampScroll(v.scrollOffset, total)
	}
	if total <= height {
		offset = 0
	}
	v.lastOffset = offset

	for row := 0; row < height; row++ {
		idx := offset + row
		if idx >= total {
			break
		}
		tui.DrawLine(s, x0, y0+row, v.subLines[idx])
	}

	ly := y0 + height - 1
	backText := " ← back "
	backCells := tui.LineFromSegments([]tui.Segment{
		{Text: backText, Style: pal.Style(pal.Accent, pal.Surface).Bold(true)},
	}, len(backText))
	bx := x0 + width - len(backText)
	tui.DrawLine(s, bx, ly, backCells)
	v.backX0, v.backX1, v.backY = bx, bx+len(backText)-1, ly
	v.backVisible = true

	v.indicatorVisible = false
	below := total - (offset + height)
	if below > 0 {
		label := fmt.Sprintf(" ↓ %d ", below)
		labelCells := tui.LineFromSegments([]tui.Segment{
			{Text: label, Style: pal.Style(pal.Accent, pal.Surface)},
		}, len(label))
		lx := bx - len(label)
		tui.DrawLine(s, lx, ly, labelCells)
		v.indicatorX0, v.indicatorX1, v.indicatorY = lx, lx+len(label)-1, ly
		v.indicatorVisible = true
	}
}

func (v *Viewport) rebuildSubagent(pal *tui.Palette) {
	width := v.Width()
	if width < 1 {
		width = 1
	}
	v.subLines, v.subToolBlocks = v.mv.SubagentTranscript(v.activeSubagent, width, pal, v.now)
	v.subRev = v.conv.Revision()
}

func (v *Viewport) showSubagent(sa *tui.Subagent) {
	if v.activeSubagent == sa {
		return
	}
	v.activeSubagent = sa
	v.subRev = -1
	v.pendingSubToggle = -1
	v.follow = true
	v.scrollOffset = 0
	v.itemCursor = -1
	v.RequestRender()
}

func (v *Viewport) BackToMain() {
	if v.activeSubagent == nil {
		return
	}
	v.activeSubagent = nil
	v.subRev = -1
	v.pendingSubToggle = -1
	v.backVisible = false
	v.follow = true
	v.scrollOffset = 0
	v.itemCursor = -1
	v.RequestRender()
}

func (v *Viewport) InSubagentView() bool { return v.activeSubagent != nil }

func (v *Viewport) toggleSubagentToolAt(lineIdx int) {
	sa := v.activeSubagent
	if sa == nil {
		return
	}
	if len(v.subToolBlocks) == 0 && len(sa.Tools) > 0 {
		v.pendingSubToggle = lineIdx
		v.subRev = -1
		v.RequestRender()
		return
	}
	v.applySubagentToggle(lineIdx)
}

func (v *Viewport) applySubagentToggle(lineIdx int) {
	sa := v.activeSubagent
	if sa == nil {
		return
	}
	for _, b := range v.subToolBlocks {
		if lineIdx < b.Start || lineIdx >= b.End {
			continue
		}
		var j int
		switch {
		case sscanfOK(b.ID, "subtool:%d", &j) && j >= 0 && j < len(sa.Tools):
			sa.Tools[j].Expanded = !sa.Tools[j].Expanded
		case sscanfOK(b.ID, "subthink:%d", &j):
			th := 0
			for i := range sa.Parts {
				p := &sa.Parts[i]
				if p.Kind != tui.PartThinking {
					continue
				}
				if th == j {
					p.ThinkingExpanded = !p.ThinkingExpanded
					break
				}
				th++
			}
		default:
			return
		}
		v.conv.BumpRevision()
		v.subRev = -1
		v.RequestRender()
		return
	}
}

func (v *Viewport) lineCount() int {
	if v.activeSubagent != nil {
		return len(v.subLines)
	}
	if v.flat == nil {
		return 0
	}
	return len(v.flat.Lines)
}

func (v *Viewport) drawLogo(s tcell.Screen, pal *tui.Palette) {
	width := v.Width()
	height := v.Height()
	x0, y0 := v.bounds.Min.X, v.bounds.Min.Y

	bg := pal.Style(pal.Foreground, pal.Background)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			s.SetContent(x0+x, y0+y, ' ', nil, bg)
		}
	}

	logoW := tui.DisplayWidth(logoVESVAI[0])
	blockH := len(logoVESVAI) + 2
	startY := y0 + (height-blockH)/2
	if startY < y0 {
		startY = y0
	}
	startX := x0 + (width-logoW)/2
	if startX < x0 {
		startX = x0
	}

	logoStyle := pal.Style(pal.Accent, pal.Background)
	for i, line := range logoVESVAI {
		cells := tui.LineFromSegments([]tui.Segment{{Text: line, Style: logoStyle}}, width)
		tui.DrawLine(s, startX, startY+i, cells)
	}

	tagY := startY + len(logoVESVAI) + 1
	tagStyle := pal.Style(pal.Muted, pal.Background)
	tagCells := tui.LineFromSegments([]tui.Segment{{Text: logoTagline, Style: tagStyle}}, width)
	tui.DrawLine(s, x0+(width-tui.DisplayWidth(logoTagline))/2, tagY, tagCells)
}

func (v *Viewport) following() bool {
	return v.follow
}

func (v *Viewport) offset() int {
	return v.scrollOffset
}

func (v *Viewport) maxOffset() int {
	return tui.ClampScroll(v.lineCount()-v.Height(), v.lineCount())
}

func (v *Viewport) scrollBy(delta int) {
	if v.flat == nil {
		return
	}
	off := v.offset() + delta
	if v.follow {
		if off >= v.maxOffset() {
			v.setFollow(true)
			return
		}
		v.setFollow(false)
		v.setOffset(off)
		return
	}
	if off >= v.maxOffset() {
		v.setFollow(true)
		return
	}
	v.setOffset(tui.ClampScroll(off, v.maxOffset()))
}

func (v *Viewport) HandleEvent(ev tcell.Event) bool {
	switch e := ev.(type) {
	case *tcell.EventKey:
		if e.Key() == tcell.KeyEsc && v.activeSubagent != nil {
			v.BackToMain()
			return true
		}
		switch e.Key() {
		case tcell.KeyUp:
			v.scrollBy(-1)
			return true
		case tcell.KeyDown:
			v.scrollBy(1)
			return true
		case tcell.KeyPgUp:
			v.scrollBy(-v.Height())
			return true
		case tcell.KeyPgDn:
			v.scrollBy(v.Height())
			return true
		case tcell.KeyHome:
			v.setFollow(false)
			v.setOffset(0)
			return true
		case tcell.KeyEnd:
			v.setFollow(true)
			return true
		case tcell.KeyEnter:
			if v.itemCursor >= 0 && v.itemCursor < v.itemCount() {
				item := v.flat.Items[v.itemCursor]
				if item.Kind == "subagent" {
					if sa := v.conv.SubagentByBlockID(item.ID); sa != nil {
						v.showSubagent(sa)
					}
				} else if v.OnToggle != nil {
					v.OnToggle(item.ID)
				}
			}
			return true
		case tcell.KeyRune:
			switch e.Rune() {
			case ' ':
				if v.itemCursor >= 0 && v.itemCursor < v.itemCount() {
					item := v.flat.Items[v.itemCursor]
					if item.Kind == "subagent" {
						if sa := v.conv.SubagentByBlockID(item.ID); sa != nil {
							v.showSubagent(sa)
						}
					} else if v.OnToggle != nil {
						v.OnToggle(item.ID)
					}
				}
				return true
			case ']':
				v.nextItem()
				return true
			case '[':
				v.prevItem()
				return true
			}
		}
	case *tcell.EventMouse:
		x, y := e.Position()
		if v.activeSubagent != nil {
			switch {
			case e.Buttons()&tcell.WheelUp != 0:
				v.scrollBy(-3)
			case e.Buttons()&tcell.WheelDown != 0:
				v.scrollBy(3)
			case e.Buttons()&tcell.Button1 != 0:
				if v.indicatorVisible && y == v.indicatorY &&
					x >= v.indicatorX0 && x <= v.indicatorX1 {
					v.setFollow(true)
				} else if v.backVisible && y == v.backY &&
					x >= v.backX0 && x <= v.backX1 {
					v.BackToMain()
				} else {
					v.toggleSubagentToolAt(y - v.bounds.Min.Y + v.lastOffset)
				}
			}
			return true
		}
		switch {
		case e.Buttons()&tcell.WheelUp != 0:
			v.scrollBy(-3)
			return true
		case e.Buttons()&tcell.WheelDown != 0:
			v.scrollBy(3)
			return true
		case e.Buttons()&tcell.Button1 != 0:
			if v.indicatorVisible && y == v.indicatorY &&
				x >= v.indicatorX0 && x <= v.indicatorX1 {
				v.setFollow(true)
				return true
			}
			row := y - v.bounds.Min.Y
			idx := row + v.lastOffset
			if idx >= 0 {
				if fm := v.userMsgAt(idx); fm != nil && fm.Msg.Role == tui.RoleUser {
					if v.OnUserClick != nil {
						v.OnUserClick(x, y, fm.Msg)
					}
					return true
				}
				if item := v.itemAt(idx); item != nil {
					if item.Kind == "subagent" {
						if sa := v.conv.SubagentByBlockID(item.ID); sa != nil {
							v.showSubagent(sa)
						}
					} else if v.OnToggle != nil {
						v.OnToggle(item.ID)
					}
					return true
				}
			}
			return true
		}
	}
	return false
}

func sscanfOK(s, format string, dest ...any) bool {
	_, err := fmt.Sscanf(s, format, dest...)
	return err == nil
}

func (v *Viewport) IndicatorVisible() bool { return v.indicatorVisible }

func (v *Viewport) IndicatorHitbox() (x0, x1, y int) {
	return v.indicatorX0, v.indicatorX1, v.indicatorY
}

func (v *Viewport) BackHitbox() (x0, x1, y int) {
	return v.backX0, v.backX1, v.backY
}

func (v *Viewport) userMsgAt(lineIdx int) *FlatMsg {
	if v.flat == nil {
		return nil
	}
	for _, fm := range v.flat.Messages {
		if lineIdx >= fm.Start && lineIdx < fm.End {
			return fm
		}
	}
	return nil
}

func (v *Viewport) itemAt(lineIdx int) *FlatItem {
	if v.flat == nil {
		return nil
	}
	for _, it := range v.flat.Items {
		if lineIdx >= it.Start && lineIdx < it.End {
			return it
		}
	}
	return nil
}

func (v *Viewport) itemCount() int {
	if v.flat == nil {
		return 0
	}
	return len(v.flat.Items)
}

func (v *Viewport) nextItem() {
	if v.itemCount() == 0 {
		return
	}
	v.itemCursor++
	if v.itemCursor >= v.itemCount() {
		v.itemCursor = 0
	}
	v.revealCursor()
	v.RequestRender()
}

func (v *Viewport) prevItem() {
	if v.itemCount() == 0 {
		return
	}
	v.itemCursor--
	if v.itemCursor < 0 {
		v.itemCursor = v.itemCount() - 1
	}
	v.revealCursor()
	v.RequestRender()
}

func (v *Viewport) revealCursor() {
	if v.flat == nil || v.itemCursor < 0 || v.itemCursor >= v.itemCount() {
		return
	}
	item := v.flat.Items[v.itemCursor]
	if v.follow {
		v.setFollow(false)
	}
	off := v.offset()
	if item.Start < off {
		v.setOffset(item.Start)
	} else if item.Start >= off+v.Height() {
		v.setOffset(item.Start - v.Height() + 1)
	}
}

func (v *Viewport) Tick(elapsed time.Duration) bool {
	v.now = elapsed
	return v.Hooks.Tick(elapsed)
}
