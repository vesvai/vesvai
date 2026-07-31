package components

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type TodoItem struct {
	ID          string
	Description string
	Status      string
}

type TodoView struct {
	items   []TodoItem
	visible bool
	dirty   bool
}

func NewTodoView() *TodoView {
	return &TodoView{}
}

func (tv *TodoView) IsVisible() bool   { return tv.visible }
func (tv *TodoView) Show()             { tv.visible = true }
func (tv *TodoView) Hide()             { tv.visible = false }
func (tv *TodoView) Toggle()           { tv.visible = !tv.visible }
func (tv *TodoView) SetVisible(v bool) { tv.visible = v }
func (tv *TodoView) Items() []TodoItem { return tv.items }
func (tv *TodoView) IsDirty() bool     { return tv.dirty }

func (tv *TodoView) HandleToolResult(toolName, result string) {
	switch toolName {
	case "list-todos":
		tv.parseListResult(result)
	case "set-todo":
		tv.parseSetResult(result)
	case "update-todo":
		tv.parseUpdateResult(result)
	case "delete-todo":
		tv.parseDeleteResult(result)
	}
	tv.dirty = true
}

func (tv *TodoView) parseListResult(result string) {
	lines := strings.Split(result, "\n")
	var items []TodoItem
	listLineRegex := regexp.MustCompile(`^\[([ x])\]\s+(\S+)\s+-\s+(.*)$`)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		matches := listLineRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		status := "pending"
		if matches[1] == "x" {
			status = "done"
		}
		items = append(items, TodoItem{
			ID:          matches[2],
			Description: matches[3],
			Status:      status,
		})
	}
	if len(items) == 0 && strings.Contains(result, "No todos found") {
		tv.items = nil
		return
	}
	tv.items = items
}

func (tv *TodoView) parseSetResult(result string) {
	id := extractField(result, "ID:")
	desc := extractField(result, "Description:")
	status := extractField(result, "Status:")
	if id == "" {
		return
	}
	for i, item := range tv.items {
		if item.ID == id {
			tv.items[i].Description = desc
			tv.items[i].Status = status
			return
		}
	}
	tv.items = append(tv.items, TodoItem{
		ID:          id,
		Description: desc,
		Status:      status,
	})
}

func (tv *TodoView) parseUpdateResult(result string) {
	id := extractField(result, "ID:")
	desc := extractField(result, "Description:")
	status := extractField(result, "Status:")
	if id == "" {
		return
	}
	for i, item := range tv.items {
		if item.ID == id {
			if desc != "" {
				tv.items[i].Description = desc
			}
			if status != "" {
				tv.items[i].Status = status
			}
			return
		}
	}
	tv.items = append(tv.items, TodoItem{
		ID:          id,
		Description: desc,
		Status:      status,
	})
}

func (tv *TodoView) parseDeleteResult(result string) {
	id := extractField(result, "ID:")
	if id == "" {
		return
	}
	for i, item := range tv.items {
		if item.ID == id {
			tv.items = append(tv.items[:i], tv.items[i+1:]...)
			return
		}
	}
}

func extractField(result, field string) string {
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, field) {
			return strings.TrimSpace(strings.TrimPrefix(line, field))
		}
	}
	return ""
}

func (tv *TodoView) PendingCount() int {
	count := 0
	for _, item := range tv.items {
		if item.Status == "pending" {
			count++
		}
	}
	return count
}

func (tv *TodoView) DoneCount() int {
	count := 0
	for _, item := range tv.items {
		if item.Status == "done" {
			count++
		}
	}
	return count
}

func (tv *TodoView) TotalCount() int {
	return len(tv.items)
}

func (tv *TodoView) Height(width int) int {
	if len(tv.items) == 0 {
		return 0
	}
	return len(tv.items) + 3
}

func (tv *TodoView) Draw(s tcell.Screen, y, width int) {
	if len(tv.items) == 0 {
		return
	}

	panelWidth := 40
	if panelWidth > width-4 {
		panelWidth = width - 4
	}
	startX := width - panelWidth - 1

	panelBg := tcell.StyleDefault.Background(theme.BgOverlay)
	render.FillArea(s, startX, y, panelWidth, 2+len(tv.items), panelBg)

	headerStyle := tcell.StyleDefault.
		Foreground(theme.AccentCyan).
		Background(theme.BgOverlay).
		Bold(true)
	render.DrawText(s, startX+1, y, "□ Todos", headerStyle)

	countStr := strconv.Itoa(tv.DoneCount()) + "/" + strconv.Itoa(tv.TotalCount())
	countStyle := tcell.StyleDefault.Foreground(theme.TextDim).Background(theme.BgOverlay)
	render.DrawText(s, startX+panelWidth-len(countStr)-1, y, countStr, countStyle)

	sepStyle := tcell.StyleDefault.Foreground(theme.BorderMuted).Background(theme.BgOverlay)
	for i := 0; i < panelWidth; i++ {
		s.SetContent(startX+i, y+1, '─', nil, sepStyle)
	}

	itemY := y + 2
	for _, item := range tv.items {
		var icon string
		var itemStyle tcell.Style
		switch item.Status {
		case "done":
			icon = "✓"
			itemStyle = tcell.StyleDefault.Foreground(theme.AccentGreen).Background(theme.BgOverlay)
		case "in-progress":
			icon = "◐"
			itemStyle = tcell.StyleDefault.Foreground(theme.AccentAmber).Background(theme.BgOverlay)
		default:
			icon = "○"
			itemStyle = tcell.StyleDefault.Foreground(theme.TextSecondary).Background(theme.BgOverlay)
		}

		s.SetContent(startX+1, itemY, []rune(icon)[0], nil, itemStyle)

		desc := item.Description
		maxLen := panelWidth - 5
		if len(desc) > maxLen {
			desc = desc[:maxLen-1] + "…"
		}
		descStyle := tcell.StyleDefault.Foreground(theme.TextPrimary).Background(theme.BgOverlay)
		if item.Status == "done" {
			descStyle = descStyle.Foreground(theme.TextDim)
		}
		render.DrawText(s, startX+3, itemY, desc, descStyle)
		itemY++
	}
}
