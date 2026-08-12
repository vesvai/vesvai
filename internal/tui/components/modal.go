package components

import (
	"image"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui"
)

func CenteredRect(sw, sh, w, h int) image.Rectangle {
	if w > sw-4 {
		w = sw - 4
	}
	if h > sh-4 {
		h = sh - 4
	}
	if w < 12 {
		w = 12
	}
	if h < 5 {
		h = 5
	}
	x0 := (sw - w) / 2
	y0 := (sh - h) / 2
	if y0 < 0 {
		y0 = 0
	}
	return image.Rect(x0, y0, x0+w, y0+h)
}

func drawModalBox(s tcell.Screen, rect image.Rectangle, title string, pal *tui.Palette) image.Rectangle {
	x0, y0 := rect.Min.X, rect.Min.Y
	w, h := rect.Dx(), rect.Dy()
	if w < 3 || h < 3 {
		return image.Rect(x0+1, y0+1, x0+w-1, y0+h-1)
	}

	border := pal.Style(pal.BorderFocus, pal.Surface)
	bg := pal.Style(pal.Foreground, pal.Surface)

	top := tui.Line{{R: '┌', S: border}, {R: '─', S: border}}
	if title != "" {
		top = append(top, tui.Cell{R: ' ', S: border})
		for _, r := range title {
			top = append(top, tui.Cell{R: r, S: border.Foreground(pal.Accent)})
		}
		top = append(top, tui.Cell{R: ' ', S: border})
	}
	for len(top) < w-1 {
		top = append(top, tui.Cell{R: '─', S: border})
	}
	top = append(top, tui.Cell{R: '┐', S: border})
	tui.DrawLine(s, x0, y0, top)

	for y := 1; y < h-1; y++ {
		s.SetContent(x0, y0+y, '│', nil, border)
		s.SetContent(x0+w-1, y0+y, '│', nil, border)
	}

	bottom := tui.Line{{R: '└', S: border}}
	for len(bottom) < w-1 {
		bottom = append(bottom, tui.Cell{R: '─', S: border})
	}
	bottom = append(bottom, tui.Cell{R: '┘', S: border})
	tui.DrawLine(s, x0, y0+h-1, bottom)

	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			s.SetContent(x0+x, y0+y, ' ', nil, bg)
		}
	}

	return image.Rect(x0+1, y0+1, x0+w-1, y0+h-1)
}
