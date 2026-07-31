package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type SessionEntry struct {
	ID           string
	Title        string
	MessageCount int
	Model        string
	UpdatedAt    time.Time
}

type SessionPicker struct {
	visible       bool
	sessions      []SessionEntry
	filtered      []SessionEntry
	search        []rune
	cursorPos     int
	selectedIndex int
	scrollOffset  int
	OnSelect      func(id string)
	OnClose       func()
	screenWidth   int
	screenHeight  int
}

const SessionPickerMaxVisible = 10

func NewSessionPicker() *SessionPicker {
	return &SessionPicker{
		visible:  false,
		sessions: make([]SessionEntry, 0),
	}
}

func (sp *SessionPicker) SetSessions(sessions []SessionEntry) {
	sp.sessions = sessions
	sp.filtered = make([]SessionEntry, len(sessions))
	copy(sp.filtered, sessions)
	sp.selectedIndex = 0
	sp.scrollOffset = 0
	sp.search = nil
	sp.cursorPos = 0
}

func (sp *SessionPicker) Show() {
	sp.visible = true
	sp.search = nil
	sp.cursorPos = 0
	sp.selectedIndex = 0
	sp.scrollOffset = 0
	sp.filtered = make([]SessionEntry, len(sp.sessions))
	copy(sp.filtered, sp.sessions)
}

func (sp *SessionPicker) Hide() {
	sp.visible = false
	if sp.OnClose != nil {
		sp.OnClose()
	}
}

func (sp *SessionPicker) IsVisible() bool {
	return sp.visible
}

func (sp *SessionPicker) SetScreenSize(w, h int) {
	sp.screenWidth = w
	sp.screenHeight = h
}

func (sp *SessionPicker) HandleEvent(ev tcell.Event) bool {
	if !sp.visible {
		return false
	}

	switch e := ev.(type) {
	case *tcell.EventKey:
		return sp.handleKey(e)
	case *tcell.EventMouse:
		return sp.handleMouse(e)
	}
	return false
}

func (sp *SessionPicker) handleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape:
		sp.Hide()
		return true

	case tcell.KeyUp:
		sp.moveUp()
		return true

	case tcell.KeyDown:
		sp.moveDown()
		return true

	case tcell.KeyEnter, tcell.KeyTab:
		sp.executeSelected()
		return true

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(sp.search) > 0 {
			sp.search = sp.search[:len(sp.search)-1]
			sp.cursorPos--
			sp.updateFilter()
		}
		return true

	case tcell.KeyCtrlU:
		sp.search = nil
		sp.cursorPos = 0
		sp.updateFilter()
		return true

	case tcell.KeyRune:
		sp.search = append(sp.search, ev.Rune())
		sp.cursorPos++
		sp.updateFilter()
		return true
	}
	return false
}

func (sp *SessionPicker) handleMouse(ev *tcell.EventMouse) bool {
	buttons := ev.Buttons()
	if buttons&tcell.WheelUp != 0 {
		sp.moveUp()
		return true
	}
	if buttons&tcell.WheelDown != 0 {
		sp.moveDown()
		return true
	}
	return false
}

func (sp *SessionPicker) moveUp() {
	if sp.selectedIndex > 0 {
		sp.selectedIndex--
		sp.clampScroll()
	}
}

func (sp *SessionPicker) moveDown() {
	if sp.selectedIndex < len(sp.filtered)-1 {
		sp.selectedIndex++
		sp.clampScroll()
	}
}

func (sp *SessionPicker) clampScroll() {
	maxVisible := SessionPickerMaxVisible
	if sp.selectedIndex < sp.scrollOffset {
		sp.scrollOffset = sp.selectedIndex
	}
	if sp.selectedIndex >= sp.scrollOffset+maxVisible {
		sp.scrollOffset = sp.selectedIndex - maxVisible + 1
	}
}

func (sp *SessionPicker) updateFilter() {
	query := strings.ToLower(string(sp.search))
	sp.filtered = make([]SessionEntry, 0)
	for _, s := range sp.sessions {
		title := strings.ToLower(s.Title)
		id := strings.ToLower(s.ID)
		model := strings.ToLower(s.Model)
		if query == "" || strings.Contains(title, query) || strings.Contains(id, query) || strings.Contains(model, query) {
			sp.filtered = append(sp.filtered, s)
		}
	}
	sp.selectedIndex = 0
	sp.scrollOffset = 0
}

func (sp *SessionPicker) executeSelected() {
	if sp.selectedIndex >= 0 && sp.selectedIndex < len(sp.filtered) {
		selected := sp.filtered[sp.selectedIndex]
		if sp.OnSelect != nil {
			sp.OnSelect(selected.ID)
		}
		sp.Hide()
	}
}

func (sp *SessionPicker) Height() int {
	if !sp.visible {
		return 0
	}
	items := SessionPickerMaxVisible
	if len(sp.filtered) < items {
		items = len(sp.filtered)
	}
	return items + 6
}

func (sp *SessionPicker) Draw(s tcell.Screen) {
	if !sp.visible {
		return
	}

	boxWidth := 65
	if boxWidth > sp.screenWidth-4 {
		boxWidth = sp.screenWidth - 4
	}
	boxHeight := sp.Height()

	startX := render.CenterX(boxWidth, sp.screenWidth)
	startY := render.CenterY(boxHeight, sp.screenHeight)

	render.DrawBoxFilled(s, startX, startY, boxWidth, boxHeight, theme.RoundedBorder, theme.CommandPaletteBorder.ToTcell(), tcell.StyleDefault.Background(theme.BgSecondary))

	titleStyle := tcell.StyleDefault.
		Foreground(theme.AccentCyan).
		Background(theme.BgSecondary).
		Bold(true)
	title := " Load Session "
	titleX := startX + (boxWidth-len(title))/2
	render.DrawText(s, titleX, startY, title, titleStyle)

	searchY := startY + 2
	searchStyle := tcell.StyleDefault.
		Foreground(theme.TextPrimary).
		Background(theme.BgTertiary)
	render.FillArea(s, startX+2, searchY, boxWidth-4, 1, searchStyle)

	prompt := "> "
	render.DrawText(s, startX+2, searchY, prompt, tcell.StyleDefault.Foreground(theme.AccentCyan).Background(theme.BgTertiary))

	searchText := string(sp.search)
	if len(searchText) > boxWidth-8 {
		searchText = searchText[len(searchText)-(boxWidth-8):]
	}
	render.DrawText(s, startX+4, searchY, searchText, searchStyle)

	cursorStyle := tcell.StyleDefault.Foreground(theme.AccentCyan).Background(theme.BgTertiary).Bold(true)
	cursorX := startX + 4 + sp.cursorPos
	if cursorX < startX+boxWidth-2 {
		s.SetContent(cursorX, searchY, '█', nil, cursorStyle)
	}

	listY := searchY + 2
	visible := SessionPickerMaxVisible
	if len(sp.filtered) < visible {
		visible = len(sp.filtered)
	}

	if len(sp.filtered) > 0 {
		headerStyle := tcell.StyleDefault.
			Foreground(theme.TextDim).
			Background(theme.BgSecondary)
		render.DrawText(s, startX+2, listY, "Title", headerStyle)
		render.DrawText(s, startX+boxWidth-22, listY, "Messages", headerStyle)
		render.DrawText(s, startX+boxWidth-12, listY, "Updated", headerStyle)
		listY++
	}

	for i := 0; i < visible; i++ {
		idx := sp.scrollOffset + i
		if idx >= len(sp.filtered) {
			break
		}

		session := sp.filtered[idx]
		itemY := listY + i

		itemStyle := tcell.StyleDefault.
			Foreground(theme.TextPrimary).
			Background(theme.BgSecondary)

		if idx == sp.selectedIndex {
			itemStyle = tcell.StyleDefault.
				Foreground(theme.AccentCyan).
				Background(theme.BgTertiary).
				Bold(true)
			render.FillArea(s, startX+1, itemY, boxWidth-2, 1, tcell.StyleDefault.Background(theme.BgTertiary))
		}

		_, itemBg, _ := itemStyle.Decompose()

		iconStyle := tcell.StyleDefault.Foreground(theme.AccentGreen).Background(itemBg)
		s.SetContent(startX+2, itemY, '●', nil, iconStyle)

		titleText := session.Title
		if titleText == "" {
			titleText = "(untitled)"
		}
		if len(titleText) > 30 {
			titleText = titleText[:27] + "..."
		}
		render.DrawText(s, startX+4, itemY, titleText, itemStyle)

		msgCount := fmt.Sprintf("%d", session.MessageCount)
		render.DrawText(s, startX+boxWidth-22, itemY, msgCount, itemStyle)

		updatedAt := formatRelativeTime(session.UpdatedAt)
		if len(updatedAt) > 10 {
			updatedAt = updatedAt[:10]
		}
		render.DrawText(s, startX+boxWidth-12, itemY, updatedAt, itemStyle)
	}

	if len(sp.filtered) == 0 {
		emptyStyle := tcell.StyleDefault.Foreground(theme.TextDim).Background(theme.BgSecondary)
		emptyText := "No sessions found"
		emptyX := startX + (boxWidth-len(emptyText))/2
		render.DrawText(s, emptyX, listY+visible/2, emptyText, emptyStyle)
	}

	footerY := startY + boxHeight - 1
	footerStyle := tcell.StyleDefault.Foreground(theme.TextDim).Background(theme.BgSecondary)
	footer := "↑↓ navigate  enter load  esc close"
	footerX := startX + (boxWidth-len(footer))/2
	render.DrawText(s, footerX, footerY, footer, footerStyle)

	if len(sp.filtered) > visible {
		scrollbarHeight := visible
		scrollbarX := startX + boxWidth - 2
		for i := 0; i < scrollbarHeight; i++ {
			scrollChar := '░'
			relPos := 0
			if scrollbarHeight > 0 {
				relPos = (i * 100) / scrollbarHeight
			}
			selRelPos := 0
			if len(sp.filtered) > 0 {
				selRelPos = (sp.selectedIndex * 100) / len(sp.filtered)
			}
			if relPos/10 == selRelPos/10 {
				scrollChar = '█'
			}
			s.SetContent(scrollbarX, listY+i, scrollChar, nil, tcell.StyleDefault.Foreground(theme.BorderDefault).Background(theme.BgSecondary))
		}
	}
}

func formatRelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	default:
		return t.Format("Jan 02")
	}
}
