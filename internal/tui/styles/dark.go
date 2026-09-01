package styles

import "github.com/gdamore/tcell/v2"

func c(v int32) tcell.Color { return tcell.NewHexColor(v) }

func Dark() Theme {
	return Theme{
		Background:  c(0x0A0E14),
		Foreground:  c(0xD4D4D4),
		Accent:      c(0x4AF626),
		AccentDim:   c(0x1F6E2D),
		LogoBlink:   c(0x9BFF7A),
		Border:      c(0x3D4756),
		InputBg:     c(0x11151C),
		InputText:   c(0xE6E6E6),
		Placeholder: c(0x565F89),
		Cursor:      c(0xFFFFFF),
		Selection:   c(0x2F5D35),
		Chip:        c(0xFFB300),
		Mention:     c(0x00D4FF),
		StatusBar:   c(0x1A1F29),
		Hint:        c(0x8A9199),
		Warning:     c(0xFFB300),
	}
}
