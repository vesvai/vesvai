package styles

func Light() Theme {
	return Theme{
		Background:  c(0xF5F5F5),
		Foreground:  c(0x1A1A1A),
		Accent:      c(0x0A7C46),
		AccentDim:   c(0x9FD4B0),
		LogoBlink:   c(0x1FA05B),
		Border:      c(0xC9C9C9),
		InputBg:     c(0xFFFFFF),
		InputText:   c(0x1A1A1A),
		Placeholder: c(0x9E9E9E),
		Cursor:      c(0x000000),
		Selection:   c(0xBFE3C0),
		Chip:        c(0xC99700),
		Mention:     c(0x0099CC),
		StatusBar:   c(0xE8E8E8),
		Hint:        c(0x757575),
		Warning:     c(0xE67E00),
	}
}
