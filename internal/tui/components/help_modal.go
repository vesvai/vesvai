package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type HelpShortcut struct {
	Key         string
	Description string
}

type HelpModal struct {
	visible      bool
	scrollOffset int
	screenWidth  int
	screenHeight int
	shortcuts    []HelpShortcut
}

func NewHelpModal() *HelpModal {
	return &HelpModal{
		visible:   false,
		shortcuts: defaultHelpShortcuts(),
	}
}

func defaultHelpShortcuts() []HelpShortcut {
	return []HelpShortcut{
		{Key: "Ctrl+N", Description: "New session"},
		{Key: "Ctrl+O", Description: "Load session"},
		{Key: "Ctrl+S", Description: "Save session"},
		{Key: "Ctrl+P", Description: "Command palette"},
		{Key: "Ctrl+M", Description: "Change model"},
		{Key: "Ctrl+E", Description: "Export chat"},
		{Key: "Ctrl+T", Description: "Toggle theme"},
		{Key: "F1", Description: "Show this help"},
		{Key: "Esc", Description: "Close modal / Cancel"},
		{Key: "Enter", Description: "Send message / Select item"},
		{Key: "Tab", Description: "Cycle through elements"},
		{Key: "↑/↓", Description: "Navigate list"},
		{Key: "PgUp/PgDn", Description: "Scroll page"},
		{Key: "@", Description: "Mention agent (in input)"},
		{Key: "/", Description: "Slash commands (in input)"},
	}
}

func (hm *HelpModal) Show() {
	hm.visible = true
	hm.scrollOffset = 0
}

func (hm *HelpModal) Hide() {
	hm.visible = false
	hm.scrollOffset = 0
}

func (hm *HelpModal) IsVisible() bool {
	return hm.visible
}

func (hm *HelpModal) SetScreenSize(w, h int) {
	hm.screenWidth = w
	hm.screenHeight = h
}

func (hm *HelpModal) HandleEvent(ev tcell.Event) bool {
	if !hm.visible {
		return false
	}

	switch e := ev.(type) {
	case *tcell.EventKey:
		return hm.handleKey(e)
	case *tcell.EventMouse:
		return hm.handleMouse(e)
	}
	return false
}

func (hm *HelpModal) handleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyEnter, tcell.KeyF1:
		hm.Hide()
		return true

	case tcell.KeyUp:
		if hm.scrollOffset > 0 {
			hm.scrollOffset--
		}
		return true

	case tcell.KeyDown:
		hm.scrollOffset++
		return true

	case tcell.KeyPgUp:
		hm.scrollOffset -= 10
		if hm.scrollOffset < 0 {
			hm.scrollOffset = 0
		}
		return true

	case tcell.KeyPgDn:
		hm.scrollOffset += 10
		return true

	case tcell.KeyHome:
		hm.scrollOffset = 0
		return true

	case tcell.KeyEnd:
		hm.scrollOffset = 999999
		return true
	}
	return false
}

func (hm *HelpModal) handleMouse(ev *tcell.EventMouse) bool {
	buttons := ev.Buttons()
	if buttons&tcell.WheelUp != 0 {
		if hm.scrollOffset > 0 {
			hm.scrollOffset--
		}
		return true
	}
	if buttons&tcell.WheelDown != 0 {
		hm.scrollOffset++
		return true
	}
	return false
}

func (hm *HelpModal) maxVisibleItems() int {
	return 12
}

func (hm *HelpModal) clampScroll() {
	maxVisible := hm.maxVisibleItems()
	if hm.scrollOffset < 0 {
		hm.scrollOffset = 0
	}
	if hm.scrollOffset+maxVisible > len(hm.shortcuts) {
		hm.scrollOffset = len(hm.shortcuts) - maxVisible
		if hm.scrollOffset < 0 {
			hm.scrollOffset = 0
		}
	}
}

func (hm *HelpModal) Draw(s tcell.Screen) {
	if !hm.visible {
		return
	}

	width := hm.screenWidth
	height := hm.screenHeight

	if width == 0 || height == 0 {
		return
	}

	boxWidth := 50
	if boxWidth > width-4 {
		boxWidth = width - 4
	}

	hm.clampScroll()
	visibleItems := hm.maxVisibleItems()
	if len(hm.shortcuts) < visibleItems {
		visibleItems = len(hm.shortcuts)
	}
	if visibleItems < 1 {
		visibleItems = 1
	}
	boxHeight := visibleItems + 6

	startX := render.CenterX(boxWidth, width)
	startY := render.CenterY(boxHeight, height)

	render.DrawBoxFilled(s, startX, startY, boxWidth, boxHeight, theme.RoundedBorder, theme.CommandPaletteBorder.ToTcell(), tcell.StyleDefault.Background(theme.BgSecondary))

	titleStyle := tcell.StyleDefault.
		Foreground(theme.AccentCyan).
		Background(theme.BgSecondary).
		Bold(true)
	title := " Keyboard Shortcuts "
	titleX := startX + (boxWidth-len(title))/2
	render.DrawText(s, titleX, startY, title, titleStyle)

	listY := startY + 2

	for i := 0; i < visibleItems; i++ {
		idx := hm.scrollOffset + i
		if idx >= len(hm.shortcuts) {
			break
		}

		shortcut := hm.shortcuts[idx]
		itemY := listY + i

		itemStyle := tcell.StyleDefault.
			Foreground(theme.TextPrimary).
			Background(theme.BgSecondary)

		keyStyle := tcell.StyleDefault.
			Foreground(theme.AccentCyan).
			Background(theme.BgSecondary).
			Bold(true)

		render.DrawText(s, startX+2, itemY, shortcut.Key, keyStyle)
		render.DrawText(s, startX+16, itemY, shortcut.Description, itemStyle)
	}

	footerY := startY + boxHeight - 1
	footerStyle := tcell.StyleDefault.Foreground(theme.TextDim).Background(theme.BgSecondary)
	footer := "↑↓ scroll  esc/enter close"
	footerX := startX + (boxWidth-len(footer))/2
	render.DrawText(s, footerX, footerY, footer, footerStyle)

	if len(hm.shortcuts) > visibleItems {
		scrollbarHeight := visibleItems
		scrollbarX := startX + boxWidth - 2
		for i := 0; i < scrollbarHeight; i++ {
			scrollChar := '░'
			relPos := 0
			if scrollbarHeight > 0 {
				relPos = (i * 100) / scrollbarHeight
			}
			selRelPos := 0
			if len(hm.shortcuts) > 0 {
				selRelPos = (hm.scrollOffset * 100) / len(hm.shortcuts)
			}
			if relPos/10 == selRelPos/10 {
				scrollChar = '█'
			}
			s.SetContent(scrollbarX, listY+i, scrollChar, nil, tcell.StyleDefault.Foreground(theme.BorderDefault).Background(theme.BgSecondary))
		}
	}
}
