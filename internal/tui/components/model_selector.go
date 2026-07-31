package components

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type ModelEntry struct {
	Provider string
	ModelID  string
}

type ModelSelector struct {
	visible       bool
	models        []ModelEntry
	filtered      []ModelEntry
	search        []rune
	cursorPos     int
	selectedIndex int
	scrollOffset  int
	currentModel  string
	OnSelect      func(model string)
	OnClose       func()
	screenWidth   int
	screenHeight  int
}

const ModelSelectorMaxVisible = 10

func NewModelSelector() *ModelSelector {
	return &ModelSelector{
		visible: false,
		models:  make([]ModelEntry, 0),
	}
}

func (ms *ModelSelector) SetModels(models []ModelEntry) {
	ms.models = models
	ms.filtered = make([]ModelEntry, len(models))
	copy(ms.filtered, models)
	ms.selectedIndex = 0
	ms.scrollOffset = 0
	ms.search = nil
	ms.cursorPos = 0
}

func (ms *ModelSelector) SetCurrentModel(model string) {
	ms.currentModel = model
	for i, m := range ms.filtered {
		if m.ModelID == model {
			ms.selectedIndex = i
			ms.clampScroll()
			break
		}
	}
}

func (ms *ModelSelector) Show() {
	ms.visible = true
	ms.search = nil
	ms.cursorPos = 0
	ms.selectedIndex = 0
	ms.scrollOffset = 0
	ms.filtered = make([]ModelEntry, len(ms.models))
	copy(ms.filtered, ms.models)
}

func (ms *ModelSelector) Hide() {
	ms.visible = false
	if ms.OnClose != nil {
		ms.OnClose()
	}
}

func (ms *ModelSelector) IsVisible() bool {
	return ms.visible
}

func (ms *ModelSelector) SetScreenSize(w, h int) {
	ms.screenWidth = w
	ms.screenHeight = h
}

func (ms *ModelSelector) HandleEvent(ev tcell.Event) bool {
	if !ms.visible {
		return false
	}

	switch e := ev.(type) {
	case *tcell.EventKey:
		return ms.handleKey(e)
	case *tcell.EventMouse:
		return ms.handleMouse(e)
	}
	return false
}

func (ms *ModelSelector) handleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape:
		ms.Hide()
		return true

	case tcell.KeyUp:
		ms.moveUp()
		return true

	case tcell.KeyDown:
		ms.moveDown()
		return true

	case tcell.KeyEnter, tcell.KeyTab:
		ms.executeSelected()
		return true

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(ms.search) > 0 {
			ms.search = ms.search[:len(ms.search)-1]
			ms.cursorPos--
			ms.updateFilter()
		}
		return true

	case tcell.KeyCtrlU:
		ms.search = nil
		ms.cursorPos = 0
		ms.updateFilter()
		return true

	case tcell.KeyRune:
		ms.search = append(ms.search, ev.Rune())
		ms.cursorPos++
		ms.updateFilter()
		return true
	}
	return false
}

func (ms *ModelSelector) handleMouse(ev *tcell.EventMouse) bool {
	buttons := ev.Buttons()
	if buttons&tcell.WheelUp != 0 {
		ms.moveUp()
		return true
	}
	if buttons&tcell.WheelDown != 0 {
		ms.moveDown()
		return true
	}
	return false
}

func (ms *ModelSelector) moveUp() {
	if ms.selectedIndex > 0 {
		ms.selectedIndex--
		ms.clampScroll()
	}
}

func (ms *ModelSelector) moveDown() {
	if ms.selectedIndex < len(ms.filtered)-1 {
		ms.selectedIndex++
		ms.clampScroll()
	}
}

func (ms *ModelSelector) clampScroll() {
	maxVisible := ms.maxVisibleItems()
	if ms.selectedIndex < ms.scrollOffset {
		ms.scrollOffset = ms.selectedIndex
	}
	if ms.selectedIndex >= ms.scrollOffset+maxVisible {
		ms.scrollOffset = ms.selectedIndex - maxVisible + 1
	}
}

func (ms *ModelSelector) maxVisibleItems() int {
	return ModelSelectorMaxVisible
}

func (ms *ModelSelector) updateFilter() {
	query := strings.ToLower(string(ms.search))
	ms.filtered = make([]ModelEntry, 0)
	for _, m := range ms.models {
		if query == "" || strings.Contains(strings.ToLower(m.ModelID), query) || strings.Contains(strings.ToLower(m.Provider), query) {
			ms.filtered = append(ms.filtered, m)
		}
	}
	ms.selectedIndex = 0
	ms.scrollOffset = 0
}

func (ms *ModelSelector) executeSelected() {
	if ms.selectedIndex >= 0 && ms.selectedIndex < len(ms.filtered) {
		selected := ms.filtered[ms.selectedIndex]
		if ms.OnSelect != nil {
			ms.OnSelect(selected.ModelID)
		}
		ms.Hide()
	}
}

func (ms *ModelSelector) Height() int {
	if !ms.visible {
		return 0
	}
	items := ms.maxVisibleItems()
	if len(ms.filtered) < items {
		items = len(ms.filtered)
	}
	return items + 6
}

func (ms *ModelSelector) Draw(s tcell.Screen) {
	if !ms.visible {
		return
	}

	boxWidth := 60
	if boxWidth > ms.screenWidth-4 {
		boxWidth = ms.screenWidth - 4
	}
	boxHeight := ms.Height()

	startX := render.CenterX(boxWidth, ms.screenWidth)
	startY := render.CenterY(boxHeight, ms.screenHeight)

	render.DrawBoxFilled(s, startX, startY, boxWidth, boxHeight, theme.RoundedBorder, theme.CommandPaletteBorder.ToTcell(), tcell.StyleDefault.Background(theme.BgSecondary))

	titleStyle := tcell.StyleDefault.
		Foreground(theme.AccentCyan).
		Background(theme.BgSecondary).
		Bold(true)
	title := " Select Model "
	titleX := startX + (boxWidth-len(title))/2
	render.DrawText(s, titleX, startY, title, titleStyle)

	searchY := startY + 2
	searchStyle := tcell.StyleDefault.
		Foreground(theme.TextPrimary).
		Background(theme.BgTertiary)
	render.FillArea(s, startX+2, searchY, boxWidth-4, 1, searchStyle)

	prompt := "> "
	render.DrawText(s, startX+2, searchY, prompt, tcell.StyleDefault.Foreground(theme.AccentCyan).Background(theme.BgTertiary))

 searchText := string(ms.search)
 render.DrawText(s, startX+4, searchY, searchText, searchStyle)

	cursorStyle := tcell.StyleDefault.Foreground(theme.AccentCyan).Background(theme.BgTertiary).Bold(true)
	s.SetContent(startX+4+ms.cursorPos, searchY, '█', nil, cursorStyle)

	listY := searchY + 2
	visible := ms.maxVisibleItems()
	if len(ms.filtered) < visible {
		visible = len(ms.filtered)
	}

	for i := 0; i < visible; i++ {
		idx := ms.scrollOffset + i
		if idx >= len(ms.filtered) {
			break
		}

		item := ms.filtered[idx]
		itemY := listY + i

		itemStyle := tcell.StyleDefault.
			Foreground(theme.TextPrimary).
			Background(theme.BgSecondary)

		if idx == ms.selectedIndex {
			itemStyle = tcell.StyleDefault.
				Foreground(theme.AccentCyan).
				Background(theme.BgTertiary).
				Bold(true)
			render.FillArea(s, startX+1, itemY, boxWidth-2, 1, tcell.StyleDefault.Background(theme.BgTertiary))
		}

		isCurrent := item.ModelID == ms.currentModel

		_, itemBg, _ := itemStyle.Decompose()

		icon := '○'
		if isCurrent {
			icon = '●'
		}
		iconStyle := tcell.StyleDefault.Foreground(theme.AccentGreen).Background(itemBg)
		if !isCurrent {
			iconStyle = tcell.StyleDefault.Foreground(theme.TextDim).Background(itemBg)
		}
		s.SetContent(startX+2, itemY, icon, nil, iconStyle)

		modelText := item.ModelID
		if len(modelText) > boxWidth-10 {
			modelText = modelText[:boxWidth-13] + "..."
		}
		render.DrawText(s, startX+4, itemY, modelText, itemStyle)

		providerText := item.Provider
		if len(providerText) > 15 {
			providerText = providerText[:12] + "..."
		}
		providerStyle := tcell.StyleDefault.
			Foreground(theme.TextDim).
			Background(itemBg)
		render.DrawText(s, startX+boxWidth-2-len(providerText), itemY, providerText, providerStyle)
	}

	if len(ms.filtered) == 0 {
		emptyY := listY + visible/2
		emptyStyle := tcell.StyleDefault.Foreground(theme.TextDim).Background(theme.BgSecondary)
		emptyText := "No models found"
		emptyX := startX + (boxWidth-len(emptyText))/2
		render.DrawText(s, emptyX, emptyY, emptyText, emptyStyle)
	}

	footerY := startY + boxHeight - 1
	footerStyle := tcell.StyleDefault.Foreground(theme.TextDim).Background(theme.BgSecondary)
	footer := "↑↓ navigate  enter select  esc close"
	footerX := startX + (boxWidth-len(footer))/2
	render.DrawText(s, footerX, footerY, footer, footerStyle)

	if len(ms.filtered) > visible {
		scrollbarHeight := visible
		scrollbarX := startX + boxWidth - 2
		for i := 0; i < scrollbarHeight; i++ {
			scrollChar := '░'
			scrollItemIdx := ms.scrollOffset + i
			if scrollItemIdx >= ms.scrollOffset && scrollItemIdx < ms.scrollOffset+visible {
				relPos := 0
				if scrollbarHeight > 0 {
					relPos = (i * 100) / scrollbarHeight
				}
				selRelPos := 0
				if len(ms.filtered) > 0 {
					selRelPos = (ms.selectedIndex * 100) / len(ms.filtered)
				}
				if relPos/10 == selRelPos/10 {
					scrollChar = '█'
				}
			}
			s.SetContent(scrollbarX, listY+i, scrollChar, nil, tcell.StyleDefault.Foreground(theme.BorderDefault).Background(theme.BgSecondary))
		}
	}
}
