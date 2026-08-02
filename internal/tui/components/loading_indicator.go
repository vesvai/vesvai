package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type LoadingIndicator struct {
	visible      bool
	spinnerFrame int
}

func NewLoadingIndicator() *LoadingIndicator {
	return &LoadingIndicator{
		visible:      false,
		spinnerFrame: 0,
	}
}

func (li *LoadingIndicator) Show() {
	li.visible = true
}

func (li *LoadingIndicator) Hide() {
	li.visible = false
}

func (li *LoadingIndicator) IsVisible() bool {
	return li.visible
}

func (li *LoadingIndicator) Height() int {
	if li.visible {
		return 1
	}
	return 0
}

func (li *LoadingIndicator) Draw(s tcell.Screen, y, screenWidth int) {
	if !li.visible {
		return
	}

	spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	li.spinnerFrame = (li.spinnerFrame + 1) % len(spinnerChars)

	startX := 2
	spinnerStyle := tcell.StyleDefault.
		Foreground(theme.AccentGold).
		Background(theme.BgPrimary)

	text := "thinking..."
	textStyle := tcell.StyleDefault.
		Foreground(theme.TextDim).
		Background(theme.BgPrimary)

	render.DrawText(s, startX, y, spinnerChars[li.spinnerFrame], spinnerStyle)
	render.DrawText(s, startX+2, y, text, textStyle)
}
