package components

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type ApprovalPrompt struct {
	visible  bool
	toolName string
	args     string
	reason   string
}

func NewApprovalPrompt() *ApprovalPrompt {
	return &ApprovalPrompt{}
}

func (ap *ApprovalPrompt) Show(toolName, args, reason string) {
	ap.toolName = toolName
	ap.args = args
	ap.reason = reason
	ap.visible = true
}

func (ap *ApprovalPrompt) Hide() {
	ap.visible = false
}

func (ap *ApprovalPrompt) IsVisible() bool { return ap.visible }

func (ap *ApprovalPrompt) Height() int { return 7 }

func (ap *ApprovalPrompt) Draw(s tcell.Screen, screenWidth, screenHeight int) {
	if !ap.visible {
		return
	}

	const panelWidth = 72
	width := panelWidth
	if width > screenWidth-4 {
		width = screenWidth - 4
	}
	height := ap.Height()

	startX := render.CenterX(width, screenWidth)
	startY := screenHeight - height - 2

	bgColor := tcell.NewRGBColor(0x2A, 0x1C, 0x10)
	bgStyle := tcell.StyleDefault.Background(bgColor)
	render.FillArea(s, startX, startY, width, height, bgStyle)

	borderStyle := tcell.StyleDefault.
		Foreground(theme.AccentAmber).
		Background(bgColor)
	for i := 0; i < width; i++ {
		s.SetContent(startX+i, startY, '─', nil, borderStyle)
		s.SetContent(startX+i, startY+height-1, '─', nil, borderStyle)
	}
	for i := 0; i < height; i++ {
		s.SetContent(startX, startY+i, '│', nil, borderStyle)
		s.SetContent(startX+width-1, startY+i, '│', nil, borderStyle)
	}
	s.SetContent(startX, startY, '╭', nil, borderStyle)
	s.SetContent(startX+width-1, startY, '╮', nil, borderStyle)
	s.SetContent(startX, startY+height-1, '╰', nil, borderStyle)
	s.SetContent(startX+width-1, startY+height-1, '╯', nil, borderStyle)

	titleStyle := tcell.StyleDefault.
		Foreground(theme.AccentAmber).
		Background(bgColor).
		Bold(true)
	render.DrawText(s, startX+2, startY+1, "! Permission required", titleStyle)

	toolStyle := tcell.StyleDefault.
		Foreground(theme.TextPrimary).
		Background(bgColor)
	toolLine := ap.toolName + "(" + ap.args + ")"
	if len(toolLine) > width-6 {
		toolLine = toolLine[:width-7] + "…"
	}
	render.DrawText(s, startX+2, startY+2, toolLine, toolStyle)

	reasonStyle := tcell.StyleDefault.
		Foreground(theme.TextSecondary).
		Background(bgColor)
	reason := ap.reason
	if len(reason) > width-6 {
		reason = reason[:width-7] + "…"
	}
	render.DrawText(s, startX+2, startY+3, reason, reasonStyle)

	divider := strings.Repeat("─", width-2)
	render.DrawText(s, startX+1, startY+4, divider, borderStyle)

	hintStyle := tcell.StyleDefault.
		Foreground(theme.TextPrimary).
		Background(bgColor).
		Bold(true)
	hint := "[y] Allow   [a] Always allow   [n] Deny   Esc = deny"
	render.DrawText(s, startX+2, startY+5, hint, hintStyle)
}
