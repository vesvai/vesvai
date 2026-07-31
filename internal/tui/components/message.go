package components

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type MessageRole int

const (
	RoleUser MessageRole = iota
	RoleAssistant
	RoleSystem
	RoleError
)

var appBackground = theme.BgPrimary

type Message struct {
	Role        MessageRole
	Content     string
	Tools       []*ToolDisplay
	Timestamp   string
	Attachments []Attachment
}

func NewUserMessage(content string, attachments []Attachment) *Message {
	ts := "now"
	if len(attachments) > 0 {
		ts = fmt.Sprintf("now · %d att.", len(attachments))
	}
	return &Message{
		Role:        RoleUser,
		Content:     content,
		Timestamp:   ts,
		Attachments: attachments,
	}
}

func NewAssistantMessage(content string) *Message {
	return &Message{
		Role:      RoleAssistant,
		Content:   content,
		Timestamp: "now",
	}
}

func NewSystemMessage(content string) *Message {
	return &Message{
		Role:      RoleSystem,
		Content:   content,
		Timestamp: "now",
	}
}

func NewErrorMessage(content string) *Message {
	return &Message{
		Role:      RoleError,
		Content:   content,
		Timestamp: "now",
	}
}

func (m *Message) RoleName() string {
	switch m.Role {
	case RoleUser:
		return "You"
	case RoleAssistant:
		return "Vesvai"
	case RoleSystem:
		return "System"
	case RoleError:
		return "Error"
	default:
		return "Unknown"
	}
}

type MessageBubble struct {
	message  *Message
	renderer *MarkdownRenderer
}

func NewMessageBubble(msg *Message) *MessageBubble {
	return &MessageBubble{
		message:  msg,
		renderer: NewMarkdownRenderer(),
	}
}

func (mb *MessageBubble) boxWidth(screenWidth int) int {
	return screenWidth - 2
}

const userCardIndent = 4

func contentLayout(role MessageRole, screenWidth int) (contentX, contentWidth int) {
	boxWidth := screenWidth - 2
	if role == RoleUser {
		contentX = 1 + userCardIndent
		contentWidth = boxWidth - 8
	} else {
		contentX = 3
		contentWidth = boxWidth - 4
	}
	if contentWidth <= 0 {
		contentWidth = 20
	}
	return contentX, contentWidth
}

func (mb *MessageBubble) Height(screenWidth int) int {
	_, contentWidth := contentLayout(mb.message.Role, screenWidth)

	totalHeight := 0
	if mb.message.Role == RoleUser {
		totalHeight = 2
	}

	for _, td := range mb.message.Tools {
		totalHeight += td.Height(screenWidth)
	}

	lines := mb.renderer.Render(mb.message.Content)
	for _, line := range lines {
		wrapped := render.WrapText(mb.plainText(line), contentWidth)
		totalHeight += len(wrapped)
		if len(wrapped) == 0 {
			totalHeight++
		}
	}

	if len(mb.message.Attachments) > 0 {
		totalHeight += 1
		for range mb.message.Attachments {
			totalHeight++
		}
	}

	return totalHeight
}

func (mb *MessageBubble) plainText(line StyledLine) string {
	var sb strings.Builder
	for _, seg := range line.Segments {
		sb.WriteString(seg.Text)
	}
	return sb.String()
}

func messageBackground(role MessageRole) tcell.Color {
	if role == RoleUser {
		return theme.BgSecondary
	}
	return appBackground
}

func (mb *MessageBubble) Draw(s tcell.Screen, y, screenWidth int) {
	boxWidth := mb.boxWidth(screenWidth)
	startContentX, contentWidth := contentLayout(mb.message.Role, screenWidth)
	bg := messageBackground(mb.message.Role)

	contentY := y
	if mb.message.Role == RoleUser {
		cardHeight := mb.Height(screenWidth)
		cardStyle := tcell.StyleDefault.Background(bg)
		render.FillArea(s, startContentX, y, contentWidth+2, cardHeight, cardStyle)
		contentY = y + 1
	}

	for _, td := range mb.message.Tools {
		td.Draw(s, contentY, screenWidth)
		contentY += td.Height(screenWidth)
	}

	lines := mb.renderer.Render(mb.message.Content)
	for _, line := range lines {
		var segments []render.StyledSegment
		if line.PreRendered {
			segments = line.Segments
		} else {
			segments = mb.renderer.RenderInline(mb.plainText(line))
		}
		curX := startContentX
		for _, seg := range segments {
			segStyle := seg.Style.ToTcell()
			_, segBg, _ := segStyle.Decompose()
			if segBg == tcell.ColorDefault {
				segStyle = segStyle.Background(bg)
			}
			if line.PreRendered {
				for _, r := range seg.Text {
					if curX-startContentX >= contentWidth {
						break
					}
					s.SetContent(curX, contentY, r, nil, segStyle)
					curX++
				}
			} else {
				wrapped := render.WrapText(seg.Text, contentWidth-(curX-startContentX))
				for wi, wLine := range wrapped {
					if wi > 0 {
						contentY++
						curX = startContentX
					}
					for _, r := range wLine {
						if curX-startContentX >= contentWidth {
							break
						}
						s.SetContent(curX, contentY, r, nil, segStyle)
						curX++
					}
				}
			}
		}
		contentY++
	}

	if len(mb.message.Attachments) > 0 {
		contentY++
		sepStyle := tcell.StyleDefault.
			Foreground(theme.BorderMuted).
			Background(bg)
		sepWidth := contentWidth
		if mb.message.Role != RoleUser {
			sepWidth = boxWidth - 2
		}
		for i := 0; i < sepWidth; i++ {
			s.SetContent(startContentX, contentY, '─', nil, sepStyle)
		}
		contentY++

		for _, att := range mb.message.Attachments {
			attStyle := tcell.StyleDefault.
				Foreground(theme.TextSecondary).
				Background(bg)
			render.DrawText(s, startContentX, contentY, att.Icon()+" "+att.ShortName(25), attStyle)
			render.DrawText(s, startContentX+len(att.ShortName(25))+2, contentY, " · "+att.SizeFormatted(), attStyle)
			contentY++
		}
	}
}

type StreamingMessage struct {
	*Message
	renderer     *MarkdownRenderer
	accumulated  string
	toolDisplays []*ToolDisplay
	streaming    bool
	spinnerFrame int
}

func NewStreamingMessage() *StreamingMessage {
	return &StreamingMessage{
		Message: &Message{
			Role:    RoleAssistant,
			Content: "",
		},
		renderer:    NewMarkdownRenderer(),
		accumulated: "",
		streaming:   true,
	}
}

func (sm *StreamingMessage) AppendContent(chunk string) {
	sm.accumulated += chunk
	sm.Message.Content = sm.accumulated
}

func (sm *StreamingMessage) AddToolCall(name string, args map[string]any) {
	td := NewToolDisplay(name, args)
	td.SetRunning()
	sm.toolDisplays = append(sm.toolDisplays, td)
}

func (sm *StreamingMessage) CompleteToolCall(name string, result string, ok bool, duration int64) {
	for _, td := range sm.toolDisplays {
		if td.Name == name && td.Status == ToolRunning {
			if ok {
				td.SetComplete(result, duration)
			} else {
				td.SetFailed(fmt.Errorf("tool failed"), duration)
			}
			break
		}
	}
}

func (sm *StreamingMessage) ToolCount() int {
	return len(sm.toolDisplays)
}

func (sm *StreamingMessage) ActiveToolCount() int {
	count := 0
	for _, td := range sm.toolDisplays {
		if td.Status == ToolRunning || td.Status == ToolPending {
			count++
		}
	}
	return count
}

func (sm *StreamingMessage) Finish() {
	sm.streaming = false
}

func (sm *StreamingMessage) IsStreaming() bool {
	return sm.streaming
}

func (sm *StreamingMessage) HasContent() bool {
	return len(sm.accumulated) > 0 || len(sm.toolDisplays) > 0
}

func (sm *StreamingMessage) Content() string {
	return sm.accumulated
}

func (sm *StreamingMessage) Finalize() *Message {
	return &Message{
		Role:      RoleAssistant,
		Content:   sm.accumulated,
		Tools:     sm.toolDisplays,
		Timestamp: "now",
	}
}

func (sm *StreamingMessage) Height(screenWidth int) int {
	h := 0

	for _, td := range sm.toolDisplays {
		h += td.Height(screenWidth)
	}

	bw := screenWidth - 6
	if bw <= 0 {
		bw = 20
	}
	lines := sm.renderer.Render(sm.accumulated)
	for _, line := range lines {
		wrapped := render.WrapText(sm.plainTextLine(line), bw)
		h += len(wrapped)
		if len(wrapped) == 0 {
			h++
		}
	}
	if sm.streaming {
		h++
	}
	return h
}

func (sm *StreamingMessage) plainTextLine(line StyledLine) string {
	var sb strings.Builder
	for _, seg := range line.Segments {
		sb.WriteString(seg.Text)
	}
	return sb.String()
}

func (sm *StreamingMessage) Draw(s tcell.Screen, y, screenWidth int) {
	currentY := y
	startX := 2
	contentWidth := screenWidth - 6

	if sm.streaming {
		spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		sm.spinnerFrame = (sm.spinnerFrame + 1) % len(spinnerChars)
		spinnerStyle := tcell.StyleDefault.
			Foreground(theme.AccentCyan).
			Background(appBackground)
		render.DrawText(s, startX+2, currentY, spinnerChars[sm.spinnerFrame], spinnerStyle)
		currentY++
	}

	for _, td := range sm.toolDisplays {
		td.Draw(s, currentY, screenWidth)
		currentY += td.Height(screenWidth)
	}

	if sm.accumulated != "" {
		lines := sm.renderer.Render(sm.accumulated)
		for _, line := range lines {
			var segments []render.StyledSegment
			if line.PreRendered {
				segments = line.Segments
			} else {
				segments = sm.renderer.RenderInline(sm.plainTextLine(line))
			}
			curX := startX + 2
			for _, seg := range segments {
				segStyle := seg.Style.ToTcell()
				_, segBg, _ := segStyle.Decompose()
				if segBg == tcell.ColorDefault {
					segStyle = segStyle.Background(appBackground)
				}
				if line.PreRendered {
					for _, r := range seg.Text {
						if curX-startX-2 >= contentWidth {
							break
						}
						s.SetContent(curX, currentY, r, nil, segStyle)
						curX++
					}
				} else {
					wrapped := render.WrapText(seg.Text, contentWidth-(curX-startX-2))
					for wi, wLine := range wrapped {
						if wi > 0 {
							currentY++
							curX = startX + 2
						}
						for _, r := range wLine {
							if curX-startX-2 >= contentWidth {
								break
							}
							s.SetContent(curX, currentY, r, nil, segStyle)
							curX++
						}
					}
				}
			}
			currentY++
		}
	}
}

func FormatToolCall(name string, args string) string {
	if args == "" {
		return name + "()"
	}
	return name + "(" + args + ")"
}

func FormatToolResult(name string, success bool) string {
	if success {
		return theme.Check + " " + name
	}
	return theme.Cross2 + " " + name + " failed"
}
