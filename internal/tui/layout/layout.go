package layout

type Region struct {
	Left   int
	Top    int
	Width  int
	Height int
}

func (r Region) Right() int { return r.Left + r.Width }

func (r Region) Bottom() int { return r.Top + r.Height }

func (r Region) Empty() bool { return r.Width <= 0 || r.Height <= 0 }

func (r Region) Clamp(c Region) Region {
	if r.Left < c.Left {
		r.Left = c.Left
	}
	if r.Top < c.Top {
		r.Top = c.Top
	}
	if r.Right() > c.Right() {
		r.Width = c.Right() - r.Left
	}
	if r.Bottom() > c.Bottom() {
		r.Height = c.Bottom() - r.Top
	}
	if r.Width < 0 {
		r.Width = 0
	}
	if r.Height < 0 {
		r.Height = 0
	}
	return r
}

func CenterIn(outer Region, w, h int) Region {
	w, h = min(w, outer.Width), min(h, outer.Height)
	return Region{
		Left:   outer.Left + (outer.Width-w)/2,
		Top:    outer.Top + (outer.Height-h)/2,
		Width:  w,
		Height: h,
	}
}

func CenterH(outer Region, w, top, h int) Region {
	w = min(w, outer.Width)
	return Region{
		Left:   outer.Left + (outer.Width-w)/2,
		Top:    top,
		Width:  w,
		Height: h,
	}
}

func TopAligned(outer Region, height int) Region {
	return Region{outer.Left, outer.Top, outer.Width, min(height, outer.Height)}
}

func BottomAligned(outer Region, height int) Region {
	height = min(height, outer.Height)
	return Region{outer.Left, outer.Bottom() - height, outer.Width, height}
}

func Pad(r Region, h, v int) Region {
	if h*2 >= r.Width {
		h = r.Width / 2
	}
	if v*2 >= r.Height {
		v = r.Height / 2
	}
	return Region{r.Left + h, r.Top + v, r.Width - 2*h, r.Height - 2*v}
}

func StackV(outer Region, heights ...int) []Region {
	out := make([]Region, len(heights))
	y := outer.Top
	for i, h := range heights {
		out[i] = Region{outer.Left, y, outer.Width, h}
		y += h
	}
	return out
}

func StackH(outer Region, widths ...int) []Region {
	out := make([]Region, len(widths))
	x := outer.Left
	for i, w := range widths {
		out[i] = Region{x, outer.Top, w, outer.Height}
		x += w
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
