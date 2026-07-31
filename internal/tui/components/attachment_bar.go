package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

const (
	CardWidth       = 18
	CardContentWidth = CardWidth - 2
	MaxVisibleCards = 4
)

type AttachmentBar struct {
	attachments  []*Attachment
	selectedIdx  int
	scrollOffset int
	focused      bool
	visibleCount int
	boxStartX    int
	boxWidth     int
}

func NewAttachmentBar() *AttachmentBar {
	return &AttachmentBar{
		attachments:  nil,
		selectedIdx:  0,
		scrollOffset: 0,
		focused:      false,
		visibleCount: MaxVisibleCards,
	}
}

func (ab *AttachmentBar) SetCardBox(startX, width int) {
	ab.boxStartX = startX
	ab.boxWidth = width
}

func (ab *AttachmentBar) SetAttachments(attachments []*Attachment) {
	ab.attachments = attachments
	if ab.selectedIdx >= len(ab.attachments) {
		ab.selectedIdx = 0
	}
	ab.ensureVisible()
}

func (ab *AttachmentBar) Attachments() []*Attachment {
	return ab.attachments
}

func (ab *AttachmentBar) Clear() {
	ab.attachments = nil
	ab.selectedIdx = 0
	ab.scrollOffset = 0
	ab.focused = false
}

func (ab *AttachmentBar) Len() int {
	if ab.attachments == nil {
		return 0
	}
	return len(ab.attachments)
}

func (ab *AttachmentBar) Focus() {
	ab.focused = true
}

func (ab *AttachmentBar) Blur() {
	ab.focused = false
}

func (ab *AttachmentBar) IsFocused() bool {
	return ab.focused
}

func (ab *AttachmentBar) FocusFirst() bool {
	if len(ab.attachments) == 0 {
		return false
	}
	ab.selectedIdx = 0
	ab.focused = true
	ab.ensureVisible()
	return true
}

func (ab *AttachmentBar) FocusNext() bool {
	if len(ab.attachments) == 0 {
		return false
	}
	if ab.selectedIdx < len(ab.attachments)-1 {
		ab.selectedIdx++
		ab.ensureVisible()
		return true
	}
	ab.focused = false
	return false
}

func (ab *AttachmentBar) FocusPrev() bool {
	if ab.selectedIdx > 0 {
		ab.selectedIdx--
		ab.ensureVisible()
		return true
	}
	return false
}

func (ab *AttachmentBar) SelectedAttachment() *Attachment {
	if ab.selectedIdx < 0 || ab.selectedIdx >= len(ab.attachments) {
		return nil
	}
	return ab.attachments[ab.selectedIdx]
}

func (ab *AttachmentBar) ensureVisible() {
	if len(ab.attachments) == 0 {
		return
	}
	visible := ab.visibleCount
	if visible <= 0 {
		visible = MaxVisibleCards
	}
	if ab.selectedIdx < ab.scrollOffset {
		ab.scrollOffset = ab.selectedIdx
	}
	if ab.selectedIdx >= ab.scrollOffset+visible {
		ab.scrollOffset = ab.selectedIdx - visible + 1
	}
	if ab.scrollOffset < 0 {
		ab.scrollOffset = 0
	}
}

func (ab *AttachmentBar) Height() int {
	return 5
}

func (ab *AttachmentBar) HandleEvent(ev tcell.Event) bool {
	if !ab.focused {
		return false
	}

	switch e := ev.(type) {
	case *tcell.EventKey:
		return ab.handleKey(e)
	case *tcell.EventMouse:
		return ab.handleMouse(e)
	}
	return false
}

func (ab *AttachmentBar) handleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyTab:
		return ab.FocusNext()

	case tcell.KeyEnter:
		return true

	case tcell.KeyEscape:
		ab.focused = false
		return false
	}

	return false
}

func (ab *AttachmentBar) handleMouse(ev *tcell.EventMouse) bool {
	return false
}

func (ab *AttachmentBar) Draw(s tcell.Screen, y, screenWidth int) {
	if len(ab.attachments) == 0 {
		return
	}

	box := ab.boxWidth
	if box <= 0 {
		box = screenWidth
	}
	startX := ab.boxStartX
	if startX <= 0 && ab.boxWidth <= 0 {
		startX = 0
		box = screenWidth
	}

	maxCards := box / (CardWidth + 1)
	if maxCards > MaxVisibleCards {
		maxCards = MaxVisibleCards
	}
	if maxCards < 1 {
		maxCards = 1
	}
	ab.visibleCount = maxCards
	ab.ensureVisible()

	totalCardWidth := maxCards * (CardWidth + 1)
	leftPad := startX + (box-totalCardWidth)/2

	hasPrev := ab.scrollOffset > 0
	hasNext := ab.scrollOffset+maxCards < len(ab.attachments)

	for i := 0; i < maxCards; i++ {
		cardIdx := ab.scrollOffset + i
		if cardIdx >= len(ab.attachments) {
			break
		}

		att := ab.attachments[cardIdx]
		isSelected := ab.focused && cardIdx == ab.selectedIdx

		cardX := leftPad + i*(CardWidth+1)
		cardTop := y + 1

		var borderCol tcell.Color
		var fillBg tcell.Color
		if isSelected {
			borderCol = theme.AccentCyan
			fillBg = tcell.NewRGBColor(0x0A, 0x1A, 0x30)
		} else {
			borderCol = theme.BorderMuted
			fillBg = theme.BgSecondary
		}

		boxFill := tcell.StyleDefault.Background(fillBg)
		render.FillArea(s, cardX+1, cardTop+1, CardWidth-2, 2, boxFill)

		boxStyle := tcell.StyleDefault.Foreground(borderCol).Background(fillBg)
		render.DrawBox(s, cardX, cardTop, CardWidth, 4, theme.RoundedBorder, boxStyle)

		iconStyle := tcell.StyleDefault.
			Foreground(theme.AccentAmber).
			Background(fillBg)
		if isSelected {
			iconStyle = tcell.StyleDefault.
				Foreground(theme.AccentCyan).
				Background(fillBg)
		}
		render.DrawText(s, cardX+1, cardTop+1, att.Icon()+" ", iconStyle)

		nameStyle := tcell.StyleDefault.
			Foreground(theme.TextPrimary).
			Background(fillBg)
		if isSelected {
			nameStyle = tcell.StyleDefault.
				Foreground(theme.AccentCyan).
				Background(fillBg).
				Bold(true)
		}
		render.DrawTextLimited(s, cardX+3, cardTop+1, CardWidth-5, att.ShortName(CardWidth-5), nameStyle)

		typeStyle := tcell.StyleDefault.
			Foreground(theme.TextMuted).
			Background(fillBg)
		render.DrawText(s, cardX+1, cardTop+2, att.TypeLabel(), typeStyle)

		sizeStyle := tcell.StyleDefault.
			Foreground(theme.TextDim).
			Background(fillBg)
		render.DrawText(s, cardX+1, cardTop+3, att.SizeFormatted(), sizeStyle)

		if att.Previewable {
			eyeStyle := tcell.StyleDefault.
				Foreground(theme.TextMuted).
				Background(fillBg)
			s.SetContent(cardX+CardWidth-3, cardTop+1, '👁', nil, eyeStyle)
		}
	}

	if hasPrev {
		scrollStyle := tcell.StyleDefault.Foreground(theme.AccentCyan).Background(theme.BgPrimary)
		s.SetContent(startX, y+1, '◀', nil, scrollStyle)
	}
	if hasNext {
		scrollStyle := tcell.StyleDefault.Foreground(theme.AccentCyan).Background(theme.BgPrimary)
		s.SetContent(startX+box-1, y+1, '▶', nil, scrollStyle)
	}
}
