package render

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
	"github.com/vesvai/vesvai/internal/tui/theme"
)

type StyledSegment struct {
	Text  string
	Style theme.Style
}

func RuneDisplayWidth(r rune) int {
	return uniseg.StringWidth(string(r))
}

func StringDisplayWidth(s string) int {
	return uniseg.StringWidth(s)
}

func DrawText(s tcell.Screen, x, y int, text string, style tcell.Style) {
	curX := x
	for _, r := range text {
		s.SetContent(curX, y, r, nil, style)
		curX += RuneDisplayWidth(r)
	}
}

func DrawTextLimited(s tcell.Screen, x, y, maxWidth int, text string, style tcell.Style) int {
	curX := x
	count := 0
	for _, r := range text {
		w := RuneDisplayWidth(r)
		if count+w > maxWidth {
			break
		}
		s.SetContent(curX, y, r, nil, style)
		curX += w
		count += w
	}
	return count
}

func DrawStyledText(s tcell.Screen, x, y int, segments []StyledSegment) {
	curX := x
	for _, seg := range segments {
		st := seg.Style.ToTcell()
		for _, r := range seg.Text {
			s.SetContent(curX, y, r, nil, st)
			curX++
		}
	}
}

func DrawGradientText(s tcell.Screen, x, y int, text string, gradient theme.Gradient) {
	curX := x
	runes := []rune(text)
	total := len(runes)
	for i, r := range runes {
		progress := 0.0
		if total > 1 {
			progress = float64(i) / float64(total-1)
		}
		color := gradient.At(progress)
		style := tcell.StyleDefault.Foreground(color)
		s.SetContent(curX, y, r, nil, style)
		curX++
	}
}

func FillArea(s tcell.Screen, x, y, width, height int, style tcell.Style) {
	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			s.SetContent(x+dx, y+dy, ' ', nil, style)
		}
	}
}

func DrawBox(s tcell.Screen, x, y, width, height int, border theme.BorderStyle, style tcell.Style) {
	if width < 2 || height < 2 {
		return
	}

	s.SetContent(x, y, []rune(border.TopLeft)[0], nil, style)
	s.SetContent(x+width-1, y, []rune(border.TopRight)[0], nil, style)
	s.SetContent(x, y+height-1, []rune(border.BottomLeft)[0], nil, style)
	s.SetContent(x+width-1, y+height-1, []rune(border.BottomRight)[0], nil, style)

	hChar := []rune(border.Horizontal)[0]
	vChar := []rune(border.Vertical)[0]

	for i := 1; i < width-1; i++ {
		s.SetContent(x+i, y, hChar, nil, style)
		s.SetContent(x+i, y+height-1, hChar, nil, style)
	}

	for i := 1; i < height-1; i++ {
		s.SetContent(x, y+i, vChar, nil, style)
		s.SetContent(x+width-1, y+i, vChar, nil, style)
	}
}

func DrawBoxFilled(s tcell.Screen, x, y, width, height int, border theme.BorderStyle, borderStyle, fillStyle tcell.Style) {
	FillArea(s, x, y, width, height, fillStyle)
	DrawBox(s, x, y, width, height, border, borderStyle)
}

func DrawBoxWithTitle(s tcell.Screen, x, y, width, height int, title string, border theme.BorderStyle, borderStyle, titleStyle, fillStyle tcell.Style) {
	DrawBoxFilled(s, x, y, width, height, border, borderStyle, fillStyle)

	if title != "" && width > len(title)+4 {
		titleX := x + 2
		titleRunes := []rune(title)
		for _, r := range titleRunes {
			s.SetContent(titleX, y, r, nil, titleStyle)
			titleX++
		}
	}
}

func DrawHorizontalLine(s tcell.Screen, x, y, width int, char rune, style tcell.Style) {
	for i := 0; i < width; i++ {
		s.SetContent(x+i, y, char, nil, style)
	}
}

func DrawVerticalLine(s tcell.Screen, x, y, height int, char rune, style tcell.Style) {
	for i := 0; i < height; i++ {
		s.SetContent(x, y+i, char, nil, style)
	}
}

func CenterX(contentWidth, screenWidth int) int {
	if contentWidth >= screenWidth {
		return 0
	}
	return (screenWidth - contentWidth) / 2
}

func CenterY(contentHeight, screenHeight int) int {
	if contentHeight >= screenHeight {
		return 0
	}
	return (screenHeight - contentHeight) / 2
}

func RuneWidth(r rune) int {
	return RuneDisplayWidth(r)
}

func StringWidth(s string) int {
	return StringDisplayWidth(s)
}

func TruncateString(s string, maxWidth int) string {
	if StringDisplayWidth(s) <= maxWidth {
		return s
	}
	var result []rune
	w := 0
	for _, r := range s {
		rw := RuneDisplayWidth(r)
		if w+rw > maxWidth-3 {
			break
		}
		result = append(result, r)
		w += rw
	}
	return string(result) + "..."
}

func WrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}

	lines := []string{}
	paragraphs := strings.Split(text, "\n")

	for _, para := range paragraphs {
		if para == "" {
			lines = append(lines, "")
			continue
		}

		words := strings.Fields(para)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}

		currentLine := words[0]
		for _, word := range words[1:] {
			testLine := currentLine + " " + word
			if StringWidth(testLine) <= maxWidth {
				currentLine = testLine
			} else {
				lines = append(lines, currentLine)
				currentLine = word
			}
		}
		lines = append(lines, currentLine)
	}

	return lines
}

func PaddingLeft(x, screenWidth, contentWidth, padding int) int {
	avail := screenWidth - contentWidth - padding*2
	if avail <= 0 {
		return padding
	}
	return padding
}

func DrawParagraph(s tcell.Screen, x, y, maxWidth int, text string, style tcell.Style) int {
	lines := WrapText(text, maxWidth)
	curY := y
	for _, line := range lines {
		DrawText(s, x, curY, line, style)
		curY++
	}
	return len(lines)
}
