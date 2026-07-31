package components

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type Action struct {
	Category    string
	Name        string
	Label       string
	Description string
	Icon        rune
}

type ActionList struct {
	visible       bool
	selectedIndex int
	scrollOffset  int
	actions       []Action
	filtered      []Action
	search        string
	OnSelect      func(Action)
	OnClose       func()
	textareaBoxX  int
	textareaWidth int
	textareaY     int
	screenWidth   int
	screenHeight  int
}

const (
	ActionListMaxVisible = 8
	ActionListItemHeight = 1
)

func NewActionList() *ActionList {
	return &ActionList{
		visible:       false,
		selectedIndex: 0,
		scrollOffset:  0,
		actions:       mockActions(),
		filtered:      nil,
	}
}

func (al *ActionList) SetActions(actions []Action) {
	al.actions = actions
	if al.visible {
		al.filtered = actions
		al.selectedIndex = 0
		al.scrollOffset = 0
	}
}

func DefaultActions() []Action {
	return mockActions()
}

func mockActions() []Action {
	return []Action{
		{Name: "fix", Label: "Fix bugs", Description: "Automatically fix bugs in the code", Icon: '◆'},
		{Name: "test", Label: "Add tests", Description: "Generate unit tests for the code", Icon: '✓'},
		{Name: "refactor", Label: "Refactor code", Description: "Restructure and improve code", Icon: '✦'},
		{Name: "doc", Label: "Add docs", Description: "Generate documentation for the code", Icon: '○'},
		{Name: "explain", Label: "Explain code", Description: "Explain the selected code in detail", Icon: '?'},
		{Name: "optimize", Label: "Optimize", Description: "Optimize performance of the code", Icon: '◇'},
		{Name: "review", Label: "Review code", Description: "Review code for issues", Icon: '▲'},
		{Name: "commit", Label: "Generate commit", Description: "Generate a commit message", Icon: '●'},
	}
}

func (al *ActionList) Show() {
	al.visible = true
	al.selectedIndex = 0
	al.scrollOffset = 0
	al.filtered = al.actions
}

func (al *ActionList) Hide() {
	al.visible = false
	if al.OnClose != nil {
		al.OnClose()
	}
}

func (al *ActionList) IsVisible() bool {
	return al.visible
}

func (al *ActionList) SetTextareaBox(x, width int) {
	al.textareaBoxX = x
	al.textareaWidth = width
}

func (al *ActionList) SetTextareaY(y int) {
	al.textareaY = y
}

func (al *ActionList) SetScreenSize(width, height int) {
	al.screenWidth = width
	al.screenHeight = height
}

func (al *ActionList) UpdateFilter(query string) {
	al.search = query
	if query == "" {
		al.filtered = al.actions
	} else {
		q := strings.ToLower(query)
		al.filtered = nil
		for _, a := range al.actions {
			if strings.Contains(strings.ToLower(a.Name), q) ||
				strings.Contains(strings.ToLower(a.Label), q) {
				al.filtered = append(al.filtered, a)
			}
		}
	}
	if al.selectedIndex >= len(al.filtered) {
		al.selectedIndex = len(al.filtered) - 1
	}
	if al.selectedIndex < 0 {
		al.selectedIndex = 0
	}
	al.clampScroll()
}

func (al *ActionList) visualRowCount() int {
	if len(al.filtered) == 0 {
		return 0
	}
	count := len(al.filtered)
	currentCategory := ""
	for _, a := range al.filtered {
		if a.Category != "" && a.Category != currentCategory {
			currentCategory = a.Category
			count++
		}
	}
	return count
}

func (al *ActionList) Height() int {
	count := al.visualRowCount()
	if count == 0 {
		return 3
	}
	h := count*ActionListItemHeight + 2
	if h > ActionListMaxVisible+2 {
		h = ActionListMaxVisible + 2
	}
	return h
}

func (al *ActionList) HandleEvent(ev tcell.Event) bool {
	if !al.visible {
		return false
	}

	switch e := ev.(type) {
	case *tcell.EventKey:
		return al.handleKey(e)
	}
	return false
}

func (al *ActionList) handleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyUp:
		al.moveUp()
		return true

	case tcell.KeyDown:
		al.moveDown()
		return true

	case tcell.KeyEnter:
		al.executeSelected()
		return true

	case tcell.KeyTab:
		al.executeSelected()
		return true

	case tcell.KeyEscape:
		al.Hide()
		return true
	}

	return false
}

func (al *ActionList) moveUp() {
	if al.selectedIndex > 0 {
		al.selectedIndex--
		al.clampScroll()
	}
}

func (al *ActionList) moveDown() {
	if al.selectedIndex < len(al.filtered)-1 {
		al.selectedIndex++
		al.clampScroll()
	}
}

func (al *ActionList) visualToItemScroll(targetVis int) int {
	if targetVis <= 0 || len(al.filtered) == 0 {
		return 0
	}
	vis := 0
	cat := ""
	for i := 0; i < len(al.filtered); i++ {
		if al.filtered[i].Category != "" && al.filtered[i].Category != cat {
			cat = al.filtered[i].Category
			vis++
			if vis >= targetVis {
				return i
			}
		}
		vis++
		if vis >= targetVis {
			return i + 1
		}
	}
	return len(al.filtered)
}

func (al *ActionList) clampScroll() {
	if len(al.filtered) == 0 {
		al.scrollOffset = 0
		return
	}
	selVis := al.itemToVisualScroll(al.selectedIndex)
	scrollVis := al.itemToVisualScroll(al.scrollOffset)

	if selVis < scrollVis {
		al.scrollOffset = al.visualToItemScroll(selVis)
	} else if selVis >= scrollVis+ActionListMaxVisible {
		targetVis := selVis - ActionListMaxVisible + 1
		al.scrollOffset = al.visualToItemScroll(targetVis)
	}
	if al.scrollOffset < 0 {
		al.scrollOffset = 0
	}
	if al.scrollOffset >= len(al.filtered) {
		al.scrollOffset = len(al.filtered) - 1
	}
}

func (al *ActionList) executeSelected() {
	if al.selectedIndex >= 0 && al.selectedIndex < len(al.filtered) {
		action := al.filtered[al.selectedIndex]
		al.Hide()
		if al.OnSelect != nil {
			al.OnSelect(action)
		}
	}
}

func (al *ActionList) Draw(s tcell.Screen) {
	if !al.visible || len(al.filtered) == 0 {
		return
	}

	boxWidth := al.textareaWidth
	startX := al.textareaBoxX

	listHeight := al.Height()
	listY := al.textareaY - listHeight

	if listY < 0 {
		listY = 0
		listHeight = al.textareaY - 1
	}

	fillStyle := tcell.StyleDefault.
		Foreground(theme.TextPrimary).
		Background(theme.BgSecondary)

	render.DrawBoxFilled(s, startX, listY, boxWidth, listHeight,
		theme.RoundedBorder,
		theme.ActionListBorder.ToTcell(),
		fillStyle)

	contentX := startX + 2
	contentY := listY + 1

	rowsDrawn := 0
	currentCategory := ""
	if al.scrollOffset > 0 && al.scrollOffset <= len(al.filtered) {
		currentCategory = al.filtered[al.scrollOffset-1].Category
	}

	catStyle := tcell.StyleDefault.
		Foreground(theme.BorderDefault).
		Background(theme.BgSecondary)

	for i := al.scrollOffset; i < len(al.filtered); i++ {
		action := al.filtered[i]

		if action.Category != "" && action.Category != currentCategory {
			currentCategory = action.Category
			if rowsDrawn >= ActionListMaxVisible {
				break
			}
			itemY := contentY + rowsDrawn
			catText := " " + action.Category + " "
			render.DrawText(s, contentX, itemY, catText, catStyle)
			rowsDrawn++
		}

		if rowsDrawn >= ActionListMaxVisible {
			break
		}

		itemY := contentY + rowsDrawn
		isSelected := i == al.selectedIndex

		var iconStyle, labelStyle tcell.Style
		if isSelected {
			iconStyle = theme.ActionListIconSelected.ToTcell()
			labelStyle = theme.ActionListItemSelected.ToTcell()
			selStyle := tcell.StyleDefault.Background(theme.AccentCyan)
			render.FillArea(s, startX+1, itemY, boxWidth-2, 1, selStyle)
		} else {
			iconStyle = theme.ActionListIcon.ToTcell()
			labelStyle = theme.ActionListItem.ToTcell()
		}

		iconStr := string(action.Icon) + " "
		render.DrawText(s, contentX, itemY, iconStr, iconStyle)
		labelEndX := contentX + 2 + render.StringDisplayWidth(action.Label)
		render.DrawText(s, contentX+2, itemY, action.Label, labelStyle)

		descStyle := tcell.StyleDefault.
			Foreground(theme.TextDim).
			Background(theme.BgSecondary)
		if isSelected {
			descStyle = tcell.StyleDefault.
				Foreground(theme.BgPrimary).
				Background(theme.AccentCyan)
		}
		descX := labelEndX + 2
		desc := action.Description
		if descX+render.StringDisplayWidth(desc) >= startX+boxWidth-2 {
			avail := startX + boxWidth - 2 - descX
			if avail > 3 {
				desc = render.TruncateString(desc, avail)
			} else {
				desc = ""
			}
		}
		if desc != "" {
			render.DrawText(s, descX, itemY, desc, descStyle)
		}

		rowsDrawn++
	}

	totalRows := al.visualRowCount()
	if totalRows > ActionListMaxVisible {
		scrollH := ActionListMaxVisible
		thumbSize := (scrollH * scrollH) / totalRows
		if thumbSize < 1 {
			thumbSize = 1
		}
		visualScroll := al.itemToVisualScroll(al.scrollOffset)
		thumbPos := 0
		if totalRows-scrollH > 0 {
			thumbPos = (visualScroll * (scrollH - thumbSize)) / (totalRows - scrollH)
		}

		scrollX := startX + boxWidth - 2
		for i := 0; i < scrollH; i++ {
			if i >= thumbPos && i < thumbPos+thumbSize {
				s.SetContent(scrollX, contentY+i, '█', nil, tcell.StyleDefault.Foreground(theme.BorderDefault))
			} else {
				s.SetContent(scrollX, contentY+i, '░', nil, tcell.StyleDefault.Foreground(theme.BgTertiary))
			}
		}
	}
}

func (al *ActionList) itemToVisualScroll(itemIdx int) int {
	vis := 0
	cat := ""
	for i := 0; i < itemIdx && i < len(al.filtered); i++ {
		if al.filtered[i].Category != "" && al.filtered[i].Category != cat {
			cat = al.filtered[i].Category
			vis++
		}
		vis++
	}
	return vis
}
