package components

import (
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type Title struct {
	visible  bool
	ascii    []string
	gradient theme.Gradient
}

func NewTitle() *Title {
	return &Title{
		visible:  true,
		ascii:    theme.TitleBlock,
		gradient: theme.TitleGradient,
	}
}

func (t *Title) Show() {
	t.visible = true
}

func (t *Title) Hide() {
	t.visible = false
}

func (t *Title) IsVisible() bool {
	return t.visible
}

func (t *Title) Height() int {
	if !t.visible {
		return 0
	}
	return len(t.ascii) + 2
}

func (t *Title) Width() int {
	if len(t.ascii) == 0 {
		return 0
	}
	return utf8.RuneCountInString(t.ascii[0])
}

func (t *Title) Draw(s tcell.Screen, startY, screenWidth int) {
	if !t.visible {
		return
	}

	titleWidth := t.Width()
	startX := render.CenterX(titleWidth, screenWidth)

	for i, line := range t.ascii {
		runes := []rune(line)
		totalWidth := len(runes)
		for j, r := range runes {
			progress := 0.0
			if totalWidth > 1 {
				progress = float64(j) / float64(totalWidth-1)
			}
			color := t.gradient.At(progress)
			style := tcell.StyleDefault.Foreground(color).Bold(true).Background(theme.BgPrimary)
			s.SetContent(startX+j, startY+i, r, nil, style)
		}
	}

	subtitle := "A quiet agent for careful code."
	subX := render.CenterX(len(subtitle), screenWidth)
	subStyle := tcell.StyleDefault.Foreground(theme.SubtitleColor).Background(theme.BgPrimary)
	for i, r := range subtitle {
		s.SetContent(subX+i, startY+len(t.ascii)+1, r, nil, subStyle)
	}
}
