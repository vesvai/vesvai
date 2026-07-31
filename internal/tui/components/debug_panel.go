package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

const (
	debugMaxEntries   = 100
	debugMaxDetailLen = 80
)

type DebugEntry struct {
	Timestamp time.Time
	Category  string
	Detail    string
	Color     tcell.Color
}

type DebugPanel struct {
	visible      bool
	scrollOffset int
	autoScroll   bool
	screenWidth  int
	screenHeight int
	entries      []DebugEntry
}

func NewDebugPanel() *DebugPanel {
	return &DebugPanel{
		autoScroll: true,
		entries:    make([]DebugEntry, 0, debugMaxEntries),
	}
}

func (dp *DebugPanel) Show() {
	dp.visible = true
	dp.autoScroll = true
	dp.scrollToBottom()
}

func (dp *DebugPanel) Hide() {
	dp.visible = false
	dp.scrollOffset = 0
}

func (dp *DebugPanel) IsVisible() bool {
	return dp.visible
}

func (dp *DebugPanel) SetScreenSize(w, h int) {
	dp.screenWidth = w
	dp.screenHeight = h
}

func (dp *DebugPanel) Add(category, detail string, color tcell.Color) {
	entry := DebugEntry{
		Timestamp: time.Now(),
		Category:  category,
		Detail:    detail,
		Color:     color,
	}

	if len(dp.entries) >= debugMaxEntries {
		dp.entries = dp.entries[1:]
	}
	dp.entries = append(dp.entries, entry)

	if dp.autoScroll {
		dp.scrollToBottom()
	}
}

func (dp *DebugPanel) HandleEvent(ev tcell.Event) bool {
	if !dp.visible {
		return false
	}

	switch e := ev.(type) {
	case *tcell.EventKey:
		return dp.handleKey(e)
	case *tcell.EventMouse:
		return dp.handleMouse(e)
	}
	return false
}

func (dp *DebugPanel) handleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyF2:
		dp.Hide()
		return true

	case tcell.KeyUp:
		if dp.scrollOffset > 0 {
			dp.scrollOffset--
			dp.autoScroll = false
		}
		return true

	case tcell.KeyDown:
		if dp.scrollOffset < len(dp.entries)-1 {
			dp.scrollOffset++
		}
		dp.checkAutoScroll()
		return true

	case tcell.KeyPgUp:
		dp.scrollOffset -= 10
		if dp.scrollOffset < 0 {
			dp.scrollOffset = 0
		}
		dp.autoScroll = false
		return true

	case tcell.KeyPgDn:
		dp.scrollOffset += 10
		dp.checkAutoScroll()
		return true

	case tcell.KeyHome:
		dp.scrollOffset = 0
		dp.autoScroll = false
		return true

	case tcell.KeyEnd:
		dp.scrollToBottom()
		return true
	}
	return false
}

func (dp *DebugPanel) handleMouse(ev *tcell.EventMouse) bool {
	buttons := ev.Buttons()
	if buttons&tcell.WheelUp != 0 {
		if dp.scrollOffset > 0 {
			dp.scrollOffset--
		}
		dp.autoScroll = false
		return true
	}
	if buttons&tcell.WheelDown != 0 {
		if dp.scrollOffset < len(dp.entries)-1 {
			dp.scrollOffset++
		}
		dp.checkAutoScroll()
		return true
	}
	return false
}

func (dp *DebugPanel) maxVisibleItems() int {
	return 20
}

func (dp *DebugPanel) scrollToBottom() {
	maxVisible := dp.maxVisibleItems()
	total := len(dp.entries)
	if total <= maxVisible {
		dp.scrollOffset = 0
	} else {
		dp.scrollOffset = total - maxVisible
	}
	dp.autoScroll = true
}

func (dp *DebugPanel) checkAutoScroll() {
	maxVisible := dp.maxVisibleItems()
	if dp.scrollOffset >= len(dp.entries)-maxVisible {
		dp.autoScroll = true
	}
}

func (dp *DebugPanel) clampScroll() {
	maxVisible := dp.maxVisibleItems()
	if dp.scrollOffset < 0 {
		dp.scrollOffset = 0
	}
	if len(dp.entries) > maxVisible && dp.scrollOffset > len(dp.entries)-maxVisible {
		dp.scrollOffset = len(dp.entries) - maxVisible
	}
}

func (dp *DebugPanel) Draw(s tcell.Screen) {
	if !dp.visible {
		return
	}

	width := dp.screenWidth
	height := dp.screenHeight

	if width == 0 || height == 0 {
		return
	}

	boxWidth := width - 4
	if boxWidth > 80 {
		boxWidth = 80
	}
	if boxWidth < 30 {
		boxWidth = 30
	}

	dp.clampScroll()
	visibleItems := dp.maxVisibleItems()
	totalEntries := len(dp.entries)
	if totalEntries < visibleItems {
		visibleItems = totalEntries
	}
	if visibleItems < 1 {
		visibleItems = 1
	}
	boxHeight := visibleItems + 4

	startX := render.CenterX(boxWidth, width)
	startY := render.CenterY(boxHeight, height)

	render.DrawBoxFilled(s, startX, startY, boxWidth, boxHeight,
		theme.RoundedBorder,
		theme.CommandPaletteBorder.ToTcell(),
		tcell.StyleDefault.Background(theme.BgSecondary))

	titleStyle := tcell.StyleDefault.
		Foreground(theme.AccentAmber).
		Background(theme.BgSecondary).
		Bold(true)
	title := fmt.Sprintf(" Debug (%d events) ", totalEntries)
	titleX := startX + (boxWidth-len(title))/2
	if titleX < startX+1 {
		titleX = startX + 1
	}
	render.DrawText(s, titleX, startY, title, titleStyle)

	listY := startY + 2

	for i := 0; i < visibleItems; i++ {
		idx := dp.scrollOffset + i
		if idx >= totalEntries {
			break
		}

		entry := dp.entries[idx]
		itemY := listY + i

		tsStyle := tcell.StyleDefault.
			Foreground(theme.TextDim).
			Background(theme.BgSecondary)
		ts := entry.Timestamp.Format("15:04:05.000")
		render.DrawTextLimited(s, startX+2, itemY, 14, ts, tsStyle)

		catStyle := tcell.StyleDefault.
			Foreground(entry.Color).
			Background(theme.BgSecondary).
			Bold(true)
		cat := fmt.Sprintf("%-12s", entry.Category)
		render.DrawTextLimited(s, startX+17, itemY, 14, cat, catStyle)

		detailStyle := tcell.StyleDefault.
			Foreground(theme.TextPrimary).
			Background(theme.BgSecondary)
		detail := entry.Detail
		if len(detail) > debugMaxDetailLen {
			detail = detail[:debugMaxDetailLen-3] + "..."
		}
		render.DrawTextLimited(s, startX+31, itemY, boxWidth-33, detail, detailStyle)
	}

	footerY := startY + boxHeight - 1
	footerStyle := tcell.StyleDefault.Foreground(theme.TextDim).Background(theme.BgSecondary)
	footer := "↑↓ scroll  esc/f2 close"
	if totalEntries > visibleItems {
		footer = fmt.Sprintf("%d/%d  ↑↓ scroll  esc/f2 close", dp.scrollOffset+visibleItems, totalEntries)
	}
	footerX := startX + (boxWidth-len(footer))/2
	if footerX < startX+1 {
		footerX = startX + 1
	}
	render.DrawText(s, footerX, footerY, footer, footerStyle)

	if totalEntries > visibleItems {
		scrollbarHeight := visibleItems
		scrollbarX := startX + boxWidth - 2
		for i := 0; i < scrollbarHeight; i++ {
			scrollChar := '░'
			relPos := 0
			if scrollbarHeight > 0 {
				relPos = (i * 100) / scrollbarHeight
			}
			selRelPos := 0
			if totalEntries > 0 {
				selRelPos = (dp.scrollOffset * 100) / totalEntries
			}
			if relPos/10 == selRelPos/10 {
				scrollChar = '█'
			}
			s.SetContent(scrollbarX, listY+i, scrollChar, nil,
				tcell.StyleDefault.Foreground(theme.BorderDefault).Background(theme.BgSecondary))
		}
	}
}

func formatDebugDetail(parts ...string) string {
	return strings.Join(parts, " ")
}
