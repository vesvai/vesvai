package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type MessageList struct {
	messages         []*MessageBubble
	streamingBubbles []*StreamingMessage
	scrollOffset     int
	maxVisible       int
	stickyBottom     bool
	screenWidth      int
}

func NewMessageList() *MessageList {
	return &MessageList{
		messages:     make([]*MessageBubble, 0),
		scrollOffset: 0,
		stickyBottom: true,
		screenWidth:  80,
	}
}

func (ml *MessageList) AddMessage(msg *Message) {
	bubble := NewMessageBubble(msg)
	ml.messages = append(ml.messages, bubble)
	ml.ScrollToBottom()
}

func (ml *MessageList) AddStreamingMessage(sm *StreamingMessage) {
	ml.streamingBubbles = append(ml.streamingBubbles, sm)
	ml.ScrollToBottom()
}

func (ml *MessageList) UpdateLastMessage(msg *Message) {
	if len(ml.messages) > 0 {
		ml.messages[len(ml.messages)-1] = NewMessageBubble(msg)
		ml.ScrollToBottom()
	}
}

func (ml *MessageList) RemoveLastMessage() {
	if len(ml.messages) > 0 {
		ml.messages = ml.messages[:len(ml.messages)-1]
	}
}

func (ml *MessageList) Clear() {
	ml.messages = make([]*MessageBubble, 0)
	ml.streamingBubbles = make([]*StreamingMessage, 0)
	ml.scrollOffset = 0
	ml.stickyBottom = true
}

func (ml *MessageList) ClearStreaming() {
	ml.streamingBubbles = make([]*StreamingMessage, 0)
}

func (ml *MessageList) Messages() []*MessageBubble {
	return ml.messages
}

func (ml *MessageList) Len() int {
	return len(ml.messages)
}

func (ml *MessageList) ScrollUp() {
	ml.stickyBottom = false
	if ml.scrollOffset > 0 {
		ml.scrollOffset--
	}
}

func (ml *MessageList) ScrollDown() {
	ml.scrollOffset++
	ml.clampScroll()
	if ml.scrollOffset >= ml.totalHeight()-ml.maxVisible {
		ml.stickyBottom = true
	}
}

func (ml *MessageList) ScrollPageUp() {
	ml.stickyBottom = false
	ml.scrollOffset -= ml.maxVisible
	if ml.scrollOffset < 0 {
		ml.scrollOffset = 0
	}
}

func (ml *MessageList) ScrollPageDown() {
	ml.scrollOffset += ml.maxVisible
	ml.clampScroll()
	if ml.scrollOffset >= ml.totalHeight()-ml.maxVisible {
		ml.stickyBottom = true
	}
}

func (ml *MessageList) ScrollToBottom() {
	ml.stickyBottom = true
	if ml.maxVisible <= 0 {
		ml.scrollOffset = 0
		return
	}
	totalHeight := ml.totalHeight()
	if totalHeight > ml.maxVisible {
		ml.scrollOffset = totalHeight - ml.maxVisible
	} else {
		ml.scrollOffset = 0
	}
}

func (ml *MessageList) ScrollToTop() {
	ml.scrollOffset = 0
}

func (ml *MessageList) clampScroll() {
	totalHeight := ml.totalHeight()
	if ml.scrollOffset < 0 {
		ml.scrollOffset = 0
	}
	if totalHeight > ml.maxVisible && ml.scrollOffset > totalHeight-ml.maxVisible {
		ml.scrollOffset = totalHeight - ml.maxVisible
	}
}

func (ml *MessageList) totalHeight() int {
	total := 0
	for _, msg := range ml.messages {
		total += msg.Height(ml.screenWidth)
	}
	for _, sm := range ml.streamingBubbles {
		total += sm.Height(ml.screenWidth)
	}
	return total
}

func (ml *MessageList) SetMaxVisible(height int) {
	ml.maxVisible = height
}

func (ml *MessageList) SetScreenWidth(width int) {
	ml.screenWidth = width
}

func (ml *MessageList) HandleEvent(ev tcell.Event) bool {
	switch e := ev.(type) {
	case *tcell.EventKey:
		return ml.handleKey(e)
	case *tcell.EventMouse:
		return ml.handleMouse(e)
	}
	return false
}

func (ml *MessageList) handleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyUp:
		ml.ScrollUp()
		return true
	case tcell.KeyDown:
		ml.ScrollDown()
		return true
	case tcell.KeyPgUp:
		ml.ScrollPageUp()
		return true
	case tcell.KeyPgDn:
		ml.ScrollPageDown()
		return true
	case tcell.KeyHome:
		ml.ScrollToTop()
		return true
	case tcell.KeyEnd:
		ml.ScrollToBottom()
		return true
	}
	return false
}

func (ml *MessageList) handleMouse(ev *tcell.EventMouse) bool {
	buttons := ev.Buttons()

	if buttons&tcell.WheelUp != 0 {
		ml.ScrollUp()
		ml.ScrollUp()
		return true
	}
	if buttons&tcell.WheelDown != 0 {
		ml.ScrollDown()
		ml.ScrollDown()
		return true
	}

	return false
}

func (ml *MessageList) Draw(s tcell.Screen, y, width, height int) {
	ml.maxVisible = height
	ml.screenWidth = width

	if len(ml.messages) == 0 && len(ml.streamingBubbles) == 0 {
		emptyText := "No messages yet. Start a conversation!"
		emptyStyle := tcell.StyleDefault.Foreground(theme.TextDim).Background(theme.BgPrimary)
		emptyX := render.CenterX(len(emptyText), width)
		for i, r := range emptyText {
			s.SetContent(emptyX+i, y+height/2, r, nil, emptyStyle)
		}
		return
	}

	if ml.stickyBottom {
		ml.ScrollToBottom()
	}

	ml.clampScroll()

	currentY := y - ml.scrollOffset
	visibleBottom := y + height

	for _, msg := range ml.messages {
		msgHeight := msg.Height(width)
		if currentY+msgHeight > y && currentY < visibleBottom {
			drawY := currentY
			if drawY < y {
				drawY = y
			}
			msg.Draw(s, drawY, width)
		}
		currentY += msgHeight
	}

	for _, sm := range ml.streamingBubbles {
		smHeight := sm.Height(width)
		if currentY+smHeight > y && currentY < visibleBottom {
			drawY := currentY
			if drawY < y {
				drawY = y
			}
			sm.Draw(s, drawY, width)
		}
		currentY += smHeight
	}

	totalHeight := ml.totalHeight()
	if totalHeight > height {
		thumbSize := (height * height) / totalHeight
		if thumbSize < 1 {
			thumbSize = 1
		}
		thumbPos := 0
		if totalHeight-height > 0 {
			thumbPos = (ml.scrollOffset * (height - thumbSize)) / (totalHeight - height)
		}

		scrollX := width - 1
		for i := 0; i < height; i++ {
			if i >= thumbPos && i < thumbPos+thumbSize {
				s.SetContent(scrollX, y+i, '█', nil, tcell.StyleDefault.Foreground(theme.BorderDefault).Background(theme.BgPrimary))
			} else {
				s.SetContent(scrollX, y+i, '░', nil, tcell.StyleDefault.Foreground(theme.BgTertiary).Background(theme.BgPrimary))
			}
		}
	}
}
