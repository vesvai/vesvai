package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/config"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type StatusBar struct {
	shortcuts []Shortcut
	version   string
	model     string
	isHeader  bool
}

type Shortcut struct {
	Key  string
	Desc string
}

func NewStatusBar() *StatusBar {
	return &StatusBar{
		shortcuts: []Shortcut{
			{Key: "⌘P", Desc: "COMMANDS"},
			{Key: "·", Desc: "SESSION"},
			{Key: "·", Desc: "LOCAL"},
		},
		version:  config.AppVersion,
		isHeader: true,
	}
}

func (sb *StatusBar) SetModel(model string) {
	sb.model = model
}

func (sb *StatusBar) Height() int {
	return 2
}

func (sb *StatusBar) Draw(s tcell.Screen, y, width int) {
	if sb.isHeader {
		sb.drawHeader(s, y, width)
	} else {
		sb.drawFooter(s, y, width)
	}
}

func (sb *StatusBar) drawHeader(s tcell.Screen, y, width int) {
	emptyStyle := tcell.StyleDefault.Background(theme.BgPrimary)

	for i := 0; i < width; i++ {
		s.SetContent(i, y, ' ', nil, emptyStyle)
	}

	brandText := "V E S V A I"
	brandStyle := tcell.StyleDefault.Foreground(theme.TextDim).Background(theme.BgPrimary)
	drawX := 2
	for _, r := range brandText {
		s.SetContent(drawX, y, r, nil, brandStyle)
		drawX++
	}

	if sb.model != "" {
		modelStyle := tcell.StyleDefault.Foreground(theme.TextDim).Background(theme.BgPrimary)
		modelStr := sb.model
		if len(modelStr) > 40 {
			modelStr = modelStr[:37] + "..."
		}
		modelX := width - len(modelStr) - 2
		for i, r := range modelStr {
			s.SetContent(modelX+i, y, r, nil, modelStyle)
		}
	}

	sepStyle := tcell.StyleDefault.Foreground(theme.BorderDefault).Background(theme.BgPrimary)
	for i := 0; i < width; i++ {
		s.SetContent(i, y+1, '─', nil, sepStyle)
	}
}

func (sb *StatusBar) drawFooter(s tcell.Screen, y, width int) {
	separatorStyle := tcell.StyleDefault.Foreground(theme.BorderDefault).Background(theme.BgPrimary)
	for i := 0; i < width; i++ {
		s.SetContent(i, y, []rune(theme.RoundedBorder.Horizontal)[0], nil, separatorStyle)
	}

	drawX := 1
	for _, sc := range sb.shortcuts {
		keyStyle := tcell.StyleDefault.Foreground(theme.AccentGold).Background(theme.BgPrimary).Bold(true)
		descStyle := tcell.StyleDefault.Foreground(theme.TextDim).Background(theme.BgPrimary)

		for _, r := range sc.Key {
			s.SetContent(drawX, y+1, r, nil, keyStyle)
			drawX++
		}
		s.SetContent(drawX, y+1, ' ', nil, keyStyle)
		drawX++

		for _, r := range sc.Desc {
			s.SetContent(drawX, y+1, r, nil, descStyle)
			drawX++
		}
		s.SetContent(drawX, y+1, ' ', nil, descStyle)
		s.SetContent(drawX+1, y+1, ' ', nil, descStyle)
		drawX += 2
	}

	rightX := width - 1

	versionStyle := tcell.StyleDefault.Foreground(theme.TextDim).Background(theme.BgPrimary)
	versionStr := sb.version
	versionX := rightX - len(versionStr)
	for i, r := range versionStr {
		s.SetContent(versionX+i, y+1, r, nil, versionStyle)
	}
	rightX = versionX - 2

	if sb.model != "" {
		modelStyle := tcell.StyleDefault.Foreground(theme.AccentGold).Background(theme.BgPrimary)
		modelStr := sb.model
		if len(modelStr) > 30 {
			modelStr = modelStr[:27] + "..."
		}
		modelX := rightX - len(modelStr)
		for i, r := range modelStr {
			s.SetContent(modelX+i, y+1, r, nil, modelStyle)
		}
		modelIconX := modelX - 2
		iconStyle := tcell.StyleDefault.Foreground(theme.AccentGold).Background(theme.BgPrimary)
		s.SetContent(modelIconX, y+1, '●', nil, iconStyle)
	}
}
