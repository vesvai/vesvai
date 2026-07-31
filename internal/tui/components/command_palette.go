package components

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type Command struct {
	Category    string
	Label       string
	Shortcut    string
	Icon        rune
	Description string
}

type CommandPalette struct {
	visible       bool
	search        []rune
	cursorPos     int
	selectedIndex int
	scrollOffset  int
	commands      []Command
	filtered      []Command
	OnExecute     func(Command)
	OnClose       func()
	screenWidth   int
	screenHeight  int
}

func NewCommandPalette() *CommandPalette {
	palette := &CommandPalette{
		visible:       false,
		search:        []rune{},
		cursorPos:     0,
		selectedIndex: 0,
		commands:      defaultCommands(),
		filtered:      nil,
		OnExecute:     nil,
		OnClose:       nil,
	}
	palette.filtered = palette.commands
	return palette
}

func defaultCommands() []Command {
	return []Command{
		{Category: "Session", Label: "New Session", Shortcut: "Ctrl+N", Icon: '✦', Description: "Start a new chat session"},
		{Category: "Session", Label: "Load Session", Shortcut: "Ctrl+O", Icon: '→', Description: "Load an existing session"},
		{Category: "Session", Label: "Export Chat", Shortcut: "Ctrl+E", Icon: '↓', Description: "Export conversation to markdown file"},
		{Category: "Session", Label: "Delete Session", Shortcut: "Ctrl+D", Icon: '✗', Description: "Delete a session"},

		{Category: "Model", Label: "Change Model", Shortcut: "Ctrl+M", Icon: '◆', Description: "Switch AI model"},

		{Category: "Panels", Label: "Toggle Todo", Shortcut: "Ctrl+T", Icon: '□', Description: "Show/hide todo panel"},
		{Category: "Panels", Label: "Toggle Subagents", Shortcut: "Ctrl+S", Icon: '▣', Description: "Show/hide subagent panel"},

		{Category: "General", Label: "Command Palette", Shortcut: "Ctrl+P", Icon: '>', Description: "Open command palette"},
		{Category: "General", Label: "Show Help", Shortcut: "F1", Icon: '?', Description: "Show keyboard shortcuts"},
		{Category: "General", Label: "Toggle Debug", Shortcut: "F2", Icon: '#', Description: "Toggle debug panel"},
		{Category: "General", Label: "Quit", Shortcut: "Esc", Icon: '×', Description: "Exit application"},
	}
}

func (cp *CommandPalette) Show() {
	cp.visible = true
	cp.search = []rune{}
	cp.cursorPos = 0
	cp.selectedIndex = 0
	cp.scrollOffset = 0
	cp.filtered = cp.commands
}

func (cp *CommandPalette) Hide() {
	cp.visible = false
	if cp.OnClose != nil {
		cp.OnClose()
	}
}

func (cp *CommandPalette) IsVisible() bool {
	return cp.visible
}

func (cp *CommandPalette) SetScreenSize(width, height int) {
	cp.screenWidth = width
	cp.screenHeight = height
}

func (cp *CommandPalette) HandleEvent(ev tcell.Event) bool {
	if !cp.visible {
		return false
	}

	switch e := ev.(type) {
	case *tcell.EventKey:
		return cp.handleKey(e)
	case *tcell.EventMouse:
		return cp.handleMouse(e)
	}
	return false
}

func (cp *CommandPalette) handleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape:
		cp.Hide()
		return true

	case tcell.KeyUp:
		cp.moveUp()
		return true

	case tcell.KeyDown:
		cp.moveDown()
		return true

	case tcell.KeyEnter:
		cp.executeSelected()
		return true

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(cp.search) > 0 {
			cp.search = cp.search[:len(cp.search)-1]
			cp.cursorPos--
			cp.filterCommands()
		}
		return true

	case tcell.KeyCtrlA:
		cp.cursorPos = 0
		return true

	case tcell.KeyCtrlE:
		cp.cursorPos = len(cp.search)
		return true

	case tcell.KeyCtrlU:
		cp.search = []rune{}
		cp.cursorPos = 0
		cp.filterCommands()
		return true

	case tcell.KeyRune:
		ch := ev.Rune()
		if ch >= 32 && ch < 127 {
			cp.search = append(cp.search[:cp.cursorPos], append([]rune{ch}, cp.search[cp.cursorPos:]...)...)
			cp.cursorPos++
			cp.filterCommands()
		}
		return true
	}

	return false
}

func (cp *CommandPalette) handleMouse(ev *tcell.EventMouse) bool {
	return false
}

func (cp *CommandPalette) moveUp() {
	if cp.selectedIndex > 0 {
		cp.selectedIndex--
		cp.clampScroll()
	}
}

func (cp *CommandPalette) moveDown() {
	if cp.selectedIndex < len(cp.filtered)-1 {
		cp.selectedIndex++
		cp.clampScroll()
	}
}

func (cp *CommandPalette) maxVisibleItems() int {
	return 10
}

func (cp *CommandPalette) clampScroll() {
	maxVisible := cp.maxVisibleItems()
	if cp.selectedIndex < cp.scrollOffset {
		cp.scrollOffset = cp.selectedIndex
	}
	if cp.selectedIndex >= cp.scrollOffset+maxVisible {
		cp.scrollOffset = cp.selectedIndex - maxVisible + 1
	}
}

func (cp *CommandPalette) executeSelected() {
	if cp.selectedIndex >= 0 && cp.selectedIndex < len(cp.filtered) {
		cmd := cp.filtered[cp.selectedIndex]
		cp.Hide()
		if cp.OnExecute != nil {
			cp.OnExecute(cmd)
		}
	}
}

func (cp *CommandPalette) filterCommands() {
	if len(cp.search) == 0 {
		cp.filtered = cp.commands
		cp.selectedIndex = 0
		cp.scrollOffset = 0
		return
	}

	query := strings.ToLower(string(cp.search))
	cp.filtered = nil

	for _, cmd := range cp.commands {
		if strings.Contains(strings.ToLower(cmd.Label), query) ||
			strings.Contains(strings.ToLower(cmd.Category), query) ||
			strings.Contains(strings.ToLower(cmd.Description), query) {
			cp.filtered = append(cp.filtered, cmd)
		}
	}

	cp.selectedIndex = 0
	cp.scrollOffset = 0
}

func (cp *CommandPalette) Draw(s tcell.Screen) {
	if !cp.visible {
		return
	}

	width := cp.screenWidth
	height := cp.screenHeight

	if width == 0 || height == 0 {
		return
	}

	boxWidth := 60
	if boxWidth > width-4 {
		boxWidth = width - 4
	}

	visibleItems := cp.maxVisibleItems()
	if len(cp.filtered) < visibleItems {
		visibleItems = len(cp.filtered)
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
	title := " Commands "
	titleX := startX + (boxWidth-len(title))/2
	render.DrawText(s, titleX, startY, title, titleStyle)

	searchY := startY + 2
	searchStyle := tcell.StyleDefault.
		Foreground(theme.TextPrimary).
		Background(theme.BgTertiary)
	render.FillArea(s, startX+2, searchY, boxWidth-4, 1, searchStyle)

	prompt := "> "
	render.DrawText(s, startX+2, searchY, prompt, tcell.StyleDefault.Foreground(theme.AccentCyan).Background(theme.BgTertiary))

	searchText := string(cp.search)
	if len(searchText) > boxWidth-8 {
		searchText = searchText[len(searchText)-(boxWidth-8):]
	}
	render.DrawText(s, startX+4, searchY, searchText, searchStyle)

	cursorStyle := tcell.StyleDefault.Foreground(theme.AccentCyan).Background(theme.BgTertiary).Bold(true)
	cursorX := startX + 4 + cp.cursorPos
	if cursorX < startX+boxWidth-2 {
		s.SetContent(cursorX, searchY, '█', nil, cursorStyle)
	}

	listY := searchY + 2

	for i := 0; i < visibleItems; i++ {
		idx := cp.scrollOffset + i
		if idx >= len(cp.filtered) {
			break
		}

		cmd := cp.filtered[idx]
		itemY := listY + i

		isSelected := idx == cp.selectedIndex

		itemStyle := tcell.StyleDefault.
			Foreground(theme.TextPrimary).
			Background(theme.BgSecondary)

		if isSelected {
			itemStyle = tcell.StyleDefault.
				Foreground(theme.AccentCyan).
				Background(theme.BgTertiary).
				Bold(true)
			render.FillArea(s, startX+1, itemY, boxWidth-2, 1, tcell.StyleDefault.Background(theme.BgTertiary))
		}

		_, itemBg, _ := itemStyle.Decompose()

		iconStyle := tcell.StyleDefault.Foreground(theme.AccentCyan).Background(itemBg)
		if !isSelected {
			iconStyle = tcell.StyleDefault.Foreground(theme.TextDim).Background(itemBg)
		}
		s.SetContent(startX+2, itemY, cmd.Icon, nil, iconStyle)

		render.DrawText(s, startX+4, itemY, cmd.Label, itemStyle)

		if cmd.Shortcut != "" {
			shortcutStyle := tcell.StyleDefault.
				Foreground(theme.TextDim).
				Background(itemBg)
			if isSelected {
				shortcutStyle = tcell.StyleDefault.
					Foreground(theme.AccentCyan).
					Background(theme.BgTertiary)
			}
			shortcutX := startX + boxWidth - len(cmd.Shortcut) - 3
			render.DrawText(s, shortcutX, itemY, cmd.Shortcut, shortcutStyle)
		}
	}

	if len(cp.filtered) == 0 {
		emptyStyle := tcell.StyleDefault.Foreground(theme.TextDim).Background(theme.BgSecondary)
		emptyText := "No commands found"
		emptyX := startX + (boxWidth-len(emptyText))/2
		render.DrawText(s, emptyX, listY+visibleItems/2, emptyText, emptyStyle)
	}

	footerY := startY + boxHeight - 1
	footerStyle := tcell.StyleDefault.Foreground(theme.TextDim).Background(theme.BgSecondary)
	footer := "↑↓ navigate  enter select  esc close"
	footerX := startX + (boxWidth-len(footer))/2
	render.DrawText(s, footerX, footerY, footer, footerStyle)

	if len(cp.filtered) > visibleItems {
		scrollbarHeight := visibleItems
		scrollbarX := startX + boxWidth - 2
		for i := 0; i < scrollbarHeight; i++ {
			scrollChar := '░'
			relPos := 0
			if scrollbarHeight > 0 {
				relPos = (i * 100) / scrollbarHeight
			}
			selRelPos := 0
			if len(cp.filtered) > 0 {
				selRelPos = (cp.selectedIndex * 100) / len(cp.filtered)
			}
			if relPos/10 == selRelPos/10 {
				scrollChar = '█'
			}
			s.SetContent(scrollbarX, listY+i, scrollChar, nil, tcell.StyleDefault.Foreground(theme.BorderDefault).Background(theme.BgSecondary))
		}
	}
}
