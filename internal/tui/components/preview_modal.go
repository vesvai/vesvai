package components

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/render"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type PreviewModal struct {
	visible      bool
	attachment   *Attachment
	scrollOffset int
	screenWidth  int
	screenHeight int
}

func NewPreviewModal() *PreviewModal {
	return &PreviewModal{
		visible:      false,
		attachment:   nil,
		scrollOffset: 0,
	}
}

func (pm *PreviewModal) Show(att *Attachment) {
	pm.attachment = att
	pm.scrollOffset = 0
	pm.visible = true
}

func (pm *PreviewModal) Hide() {
	pm.visible = false
	pm.attachment = nil
	pm.scrollOffset = 0
}

func (pm *PreviewModal) IsVisible() bool {
	return pm.visible
}

func (pm *PreviewModal) SetScreenSize(w, h int) {
	pm.screenWidth = w
	pm.screenHeight = h
}

func (pm *PreviewModal) HandleEvent(ev tcell.Event) bool {
	if !pm.visible {
		return false
	}

	switch e := ev.(type) {
	case *tcell.EventKey:
		return pm.handleKey(e)
	case *tcell.EventMouse:
		return pm.handleMouse(e)
	}
	return false
}

func (pm *PreviewModal) handleKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyEnter:
		pm.Hide()
		return true

	case tcell.KeyUp:
		if pm.scrollOffset > 0 {
			pm.scrollOffset--
		}
		return true

	case tcell.KeyDown:
		pm.scrollOffset++
		return true

	case tcell.KeyPgUp:
		pm.scrollOffset -= 10
		if pm.scrollOffset < 0 {
			pm.scrollOffset = 0
		}
		return true

	case tcell.KeyPgDn:
		pm.scrollOffset += 10
		return true

	case tcell.KeyHome:
		pm.scrollOffset = 0
		return true

	case tcell.KeyEnd:
		pm.scrollOffset = 999999
		return true
	}
	return false
}

func (pm *PreviewModal) handleMouse(ev *tcell.EventMouse) bool {
	buttons := ev.Buttons()
	if buttons&tcell.WheelUp != 0 {
		if pm.scrollOffset > 0 {
			pm.scrollOffset--
		}
		return true
	}
	if buttons&tcell.WheelDown != 0 {
		pm.scrollOffset++
		return true
	}
	return false
}

func (pm *PreviewModal) Draw(s tcell.Screen) {
	if !pm.visible || pm.attachment == nil {
		return
	}

	width := pm.screenWidth
	height := pm.screenHeight
	if width == 0 || height == 0 {
		return
	}

	overlayStyle := tcell.StyleDefault.
		Foreground(theme.TextPrimary).
		Background(theme.BgOverlay)
	render.FillArea(s, 0, 0, width, height, overlayStyle)

	modalWidth := width * 75 / 100
	if modalWidth > 90 {
		modalWidth = 90
	}
	if modalWidth < 50 {
		modalWidth = 50
	}

	modalHeight := height * 70 / 100
	if modalHeight < 12 {
		modalHeight = 12
	}

	modalX := (width - modalWidth) / 2
	modalY := (height - modalHeight) / 2

	fillStyle := tcell.StyleDefault.
		Foreground(theme.TextPrimary).
		Background(theme.BgSecondary)
	render.DrawBoxFilled(s, modalX, modalY, modalWidth, modalHeight,
		theme.DoubleBorder,
		theme.CommandPaletteBorder.ToTcell(),
		fillStyle)

	titleStr := " " + pm.attachment.Name + " "
	titleStyle := theme.CommandPaletteTitle.ToTcell()
	titleX := modalX + (modalWidth-len(titleStr))/2
	render.DrawText(s, titleX, modalY, titleStr, titleStyle)

	contentY := modalY + 2
	contentWidth := modalWidth - 4
	contentHeight := modalHeight - 4

	infoStyle := theme.CommandPaletteShortcut.ToTcell()
	iconStr := pm.attachment.Icon() + " "
	render.DrawText(s, modalX+2, contentY, iconStr, infoStyle)
	infoText := pm.attachment.TypeLabel() + " · " + pm.attachment.SizeFormatted()
	render.DrawText(s, modalX+4, contentY, infoText, infoStyle)

	infoLine2 := pm.attachment.MimeType
	render.DrawText(s, modalX+2, contentY+1, infoLine2, infoStyle)

	render.DrawHorizontalLine(s, modalX+2, contentY+2, contentWidth, '─', infoStyle)

	if pm.attachment.Previewable {
		rawLines := strings.Split(pm.attachment.Content, "\n")
		previewY := contentY + 3
		previewHeight := contentHeight - 5

		textWidth := contentWidth - 6
		if textWidth < 10 {
			textWidth = 10
		}

		var displayLines []string
		lineNums := []int{}
		for i, line := range rawLines {
			if line == "" {
				displayLines = append(displayLines, "")
				lineNums = append(lineNums, i+1)
			} else {
				wrapped := render.WrapText(line, textWidth)
				for wi, wl := range wrapped {
					displayLines = append(displayLines, wl)
					if wi == 0 {
						lineNums = append(lineNums, i+1)
					} else {
						lineNums = append(lineNums, -1)
					}
				}
			}
		}

		for li := pm.scrollOffset; li < len(displayLines) && (li-pm.scrollOffset) < previewHeight; li++ {
			line := displayLines[li]
			ln := lineNums[li]
			lineStyle := tcell.StyleDefault.
				Foreground(theme.TextPrimary).
				Background(theme.BgSecondary)

			lineNumStr := "  "
			if ln > 0 {
				if len(rawLines) > 99 {
					lineNumStr = sprintInt(ln, 3) + " "
				} else if len(rawLines) > 9 {
					lineNumStr = sprintInt(ln, 2) + " "
				} else {
					lineNumStr = sprintInt(ln, 1) + " "
				}
			} else {
				if len(rawLines) > 99 {
					lineNumStr = "    "
				} else if len(rawLines) > 9 {
					lineNumStr = "   "
				} else {
					lineNumStr = "  "
				}
			}
			lineNumStr += "│"

			lineNumStyle := tcell.StyleDefault.
				Foreground(theme.TextDim).
				Background(theme.BgSecondary)

			dy := previewY + (li - pm.scrollOffset)
			for j, r := range lineNumStr {
				s.SetContent(modalX+2+j, dy, r, nil, lineNumStyle)
			}

			textStart := modalX + 2 + len(lineNumStr) + 1
			for j, r := range line {
				if j >= textWidth {
					break
				}
				s.SetContent(textStart+j, dy, r, nil, lineStyle)
			}
		}

		totalLines := len(displayLines)
		if totalLines > previewHeight {
			scrollbarHeight := previewHeight
			thumbSize := (scrollbarHeight * scrollbarHeight) / totalLines
			if thumbSize < 1 {
				thumbSize = 1
			}
			thumbPos := 0
			if totalLines-scrollbarHeight > 0 {
				thumbPos = (pm.scrollOffset * (scrollbarHeight - thumbSize)) / (totalLines - scrollbarHeight)
			}

			scrollX := modalX + modalWidth - 2
			for i := 0; i < scrollbarHeight; i++ {
				if i >= thumbPos && i < thumbPos+thumbSize {
					s.SetContent(scrollX, contentY+3+i, '█', nil, tcell.StyleDefault.Foreground(theme.BorderDefault))
				} else {
					s.SetContent(scrollX, contentY+3+i, '░', nil, tcell.StyleDefault.Foreground(theme.BgTertiary))
				}
			}
		}

		footerText := " ↑↓ scroll · esc close "
		footerY := modalY + modalHeight - 1
		footerStyle := theme.CommandPaletteShortcut.ToTcell()
		footerX := modalX + (modalWidth-len(footerText))/2
		render.DrawText(s, footerX, footerY, footerText, footerStyle)
	} else {
		unavailableY := contentY + 4
		unavailableStyle := tcell.StyleDefault.
			Foreground(theme.TextDim).
			Background(theme.BgSecondary)

		msg := "Preview not available for this file type."
		render.DrawText(s, modalX+(modalWidth-len(msg))/2, unavailableY, msg, unavailableStyle)

		detailY := unavailableY + 2
		details := []string{
			"Name: " + pm.attachment.Name,
			"Size: " + pm.attachment.SizeFormatted(),
			"Type: " + pm.attachment.TypeLabel(),
			"MIME: " + pm.attachment.MimeType,
		}
		detailStyle := tcell.StyleDefault.
			Foreground(theme.TextPrimary).
			Background(theme.BgSecondary)
		for i, d := range details {
			render.DrawText(s, modalX+4, detailY+i, d, detailStyle)
		}

		footerText := " esc close "
		footerY := modalY + modalHeight - 1
		footerStyle := theme.CommandPaletteShortcut.ToTcell()
		footerX := modalX + (modalWidth-len(footerText))/2
		render.DrawText(s, footerX, footerY, footerText, footerStyle)
	}
}

func sprintInt(n, width int) string {
	s := ""
	for i := width - 1; i >= 0; i-- {
		digit := n % 10
		s = string(rune('0'+digit)) + s
		n /= 10
		if n == 0 && i == 0 {
			break
		}
	}
	return s
}
